// Package analytics 游戏分析/埋点系统 — HTTP Handler。
//
// 提供分析数据的采集和查询接口。
// 采集接口通过网关代理到后端分析引擎，管理后台接口直接查询 MongoDB。
package analytics

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ============================================================
// Handler 分析 HTTP 处理器
// ============================================================

// Handler 分析 HTTP 处理器，注册到网关路由。
type Handler struct {
	analytics  *Analytics
	mongoDB    *mongo.Database // 可选：用于管理后台的 MongoDB 直连查询
	logger     *slog.Logger
}

// NewHandler 创建分析 HTTP 处理器。
func NewHandler(analytics *Analytics, mongoDB *mongo.Database) *Handler {
	return &Handler{
		analytics: analytics,
		mongoDB:   mongoDB,
		logger:    slog.Default().With("module", "analytics_handler"),
	}
}

// RegisterRoutes 注册分析相关 HTTP 路由。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	// 事件采集（前端调用）
	r.POST("/api/v1/analytics/track", h.TrackEvent)

	// 管理后台分析查询
	admin := r.Group("/api/v1/admin/analytics")
	{
		admin.GET("/dau", h.DAU)
		admin.GET("/retention", h.Retention)
		admin.GET("/revenue", h.Revenue)
		admin.GET("/funnels", h.Funnels)
		// 引擎状态
		admin.GET("/stats", h.EngineStats)
	}

	h.logger.Info("analytics routes registered")
}

// ============================================================
// 事件采集
// ============================================================

// trackEventRequest 事件采集请求体。
type trackEventRequest struct {
	EventType  string                 `json:"event_type"  binding:"required"`
	PlayerID   uint64                 `json:"player_id"`
	Realm      string                 `json:"realm,omitempty"`
	Level      int                    `json:"level,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	ClientTime string                 `json:"client_time,omitempty"` // 客户端时间（RFC3339）
}

// TrackEvent 采集一个分析事件。
// 路径: POST /api/v1/analytics/track
func (h *Handler) TrackEvent(c *gin.Context) {
	var req trackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid body", "error": err.Error()})
		return
	}

	if req.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "event_type is required"})
		return
	}

	// 解析客户端时间
	ts := time.Now()
	if req.ClientTime != "" {
		if parsed, err := time.Parse(time.RFC3339, req.ClientTime); err == nil {
			ts = parsed
		}
	}

	event := &AnalyticsEvent{
		EventType:   req.EventType,
		PlayerID:    req.PlayerID,
		Realm:       req.Realm,
		PlayerLevel: req.Level,
		Timestamp:   ts,
		EventDate:   EventDateString(ts),
		SessionID:   req.SessionID,
		Properties:  req.Properties,
	}

	h.analytics.Track(event)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

// ============================================================
// 管理后台 — DAU 查询
// ============================================================

// DAURequest DAU 查询参数。
type DAURequest struct {
	StartDate string `form:"start_date"` // YYYY-MM-DD
	EndDate   string `form:"end_date"`   // YYYY-MM-DD
}

// DAU 查询每日活跃用户数。
// 路径: GET /api/v1/admin/analytics/dau?start_date=2025-01-01&end_date=2025-01-31
func (h *Handler) DAU(c *gin.Context) {
	if h.mongoDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "MongoDB not available"})
		return
	}

	var req DAURequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid params"})
		return
	}

	// 默认最近 7 天
	now := time.Now()
	if req.EndDate == "" {
		req.EndDate = now.Format("2006-01-02")
	}
	if req.StartDate == "" {
		req.StartDate = now.AddDate(0, 0, -7).Format("2006-01-02")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"event_type", EventPlayerLogin},
			{"event_date", bson.D{{"$gte", req.StartDate}, {"$lte", req.EndDate}}},
		}}},
		{{"$group", bson.D{
			{"_id", "$event_date"},
			{"dau", bson.D{{"$addToSet", "$player_id"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
		{{"$project", bson.D{
			{"_id", 0},
			{"date", "$_id"},
			{"dau", bson.D{{"$size", "$dau"}}},
			{"total_events", "$count"},
		}}},
		{{"$sort", bson.D{{"date", 1}}}},
	}

	cursor, err := h.mongoDB.Collection("analytics_events").Aggregate(ctx, pipeline)
	if err != nil {
		h.logger.Warn("DAU query failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "query failed"})
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		h.logger.Warn("DAU cursor failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "cursor failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": results})
}

// ============================================================
// 管理后台 — 留存查询
// ============================================================

// RetentionRequest 留存查询参数。
type RetentionRequest struct {
	Date string `form:"date" binding:"required"` // 基准日期 YYYY-MM-DD
}

// retentionResult 留存结果。
type retentionResult struct {
	Date     string `json:"date"`
	NewUsers int    `json:"new_users"`
	D1       *float64 `json:"d1,omitempty"`       // 次日留存率
	D3       *float64 `json:"d3,omitempty"`       // 3日留存率
	D7       *float64 `json:"d7,omitempty"`       // 7日留存率
	D14      *float64 `json:"d14,omitempty"`      // 14日留存率
	D30      *float64 `json:"d30,omitempty"`      // 30日留存率
}

// Retention 查询用户留存率。
// 路径: GET /api/v1/admin/analytics/retention?date=2025-01-01
func (h *Handler) Retention(c *gin.Context) {
	if h.mongoDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "MongoDB not available"})
		return
	}

	var req RetentionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "date is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 1. 获取基准日期注册的用户
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"event_type", EventPlayerRegister},
			{"event_date", req.Date},
		}}},
		{{"$group", bson.D{
			{"_id", nil},
			{"new_users", bson.D{{"$addToSet", "$player_id"}}},
		}}},
	}
	cursor, err := h.mongoDB.Collection("analytics_events").Aggregate(ctx, pipeline)
	if err != nil {
		h.logger.Warn("Retention query failed (register)", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "query failed"})
		return
	}

	type regResult struct {
		NewUsers []uint64 `bson:"new_users"`
	}
	var regResults []regResult
	if err := cursor.All(ctx, &regResults); err != nil {
		cursor.Close(ctx)
		h.logger.Warn("Retention cursor failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "cursor failed"})
		return
	}
	cursor.Close(ctx)

	if len(regResults) == 0 || len(regResults[0].NewUsers) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "success",
			"data": retentionResult{
				Date:     req.Date,
				NewUsers: 0,
			},
		})
		return
	}

	newUsers := regResults[0].NewUsers
	newUserCount := len(newUsers)

	// 2. 查询各周期的留存
	retentionDays := []int{1, 3, 7, 14, 30}
	baseDate, _ := time.Parse("2006-01-02", req.Date)

	result := retentionResult{
		Date:     req.Date,
		NewUsers: newUserCount,
	}

	for _, days := range retentionDays {
		targetDate := baseDate.AddDate(0, 0, days)
		dateStr := targetDate.Format("2006-01-02")

		count, err := h.mongoDB.Collection("analytics_events").CountDocuments(ctx, bson.D{
			{"event_type", EventPlayerLogin},
			{"event_date", dateStr},
			{"player_id", bson.D{{"$in", newUsers}}},
		})
		if err != nil {
			h.logger.Warn("Retention count failed", "error", err, "days", days)
			continue
		}

		if newUserCount > 0 {
			retentionRate := float64(count) / float64(newUserCount) * 100
			rate := retentionRate
			switch days {
			case 1:
				result.D1 = &rate
			case 3:
				result.D3 = &rate
			case 7:
				result.D7 = &rate
			case 14:
				result.D14 = &rate
			case 30:
				result.D30 = &rate
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": result})
}

// ============================================================
// 管理后台 — 收入查询
// ============================================================

// RevenueRequest 收入查询参数。
type RevenueRequest struct {
	StartDate string `form:"start_date"` // YYYY-MM-DD
	EndDate   string `form:"end_date"`   // YYYY-MM-DD
	GroupBy   string `form:"group_by"`   // day / week / month
}

// Revenue 查询收入数据。
// 路径: GET /api/v1/admin/analytics/revenue?start_date=2025-01-01&end_date=2025-01-31&group_by=day
func (h *Handler) Revenue(c *gin.Context) {
	if h.mongoDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "MongoDB not available"})
		return
	}

	var req RevenueRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid params"})
		return
	}

	now := time.Now()
	if req.EndDate == "" {
		req.EndDate = now.Format("2006-01-02")
	}
	if req.StartDate == "" {
		req.StartDate = now.AddDate(0, 0, -30).Format("2006-01-02")
	}
	if req.GroupBy == "" {
		req.GroupBy = "day"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 日期分组格式
	var dateProjection bson.D
	switch req.GroupBy {
	case "week":
		dateProjection = bson.D{
			{"$dateToString", bson.D{
				{"format", "%Y-W%V"},
				{"date", "$timestamp"},
			}},
		}
	case "month":
		dateProjection = bson.D{
			{"$dateToString", bson.D{
				{"format", "%Y-%m"},
				{"date", "$timestamp"},
			}},
		}
	default:
		dateProjection = bson.D{
			{"$dateToString", bson.D{
				{"format", "%Y-%m-%d"},
				{"date", "$timestamp"},
			}},
		}
	}

	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"event_type", bson.D{{"$in", bson.A{EventRechargeSuccess, EventShopPurchase}}}},
			{"event_date", bson.D{{"$gte", req.StartDate}, {"$lte", req.EndDate}}},
		}}},
		{{"$addFields", bson.D{
			{"amount", bson.D{
				{"$ifNull", bson.A{
					"$properties.amount",
					bson.D{{"$ifNull", bson.A{"$properties.price", 0}}},
				}},
			}},
			{"currency", bson.D{
				{"$ifNull", bson.A{"$properties.currency", "CNY"}},
			}},
		}}},
		{{"$group", bson.D{
			{"_id", bson.D{
				{"period", dateProjection},
				{"event_type", "$event_type"},
				{"currency", "$currency"},
			}},
			{"total_amount", bson.D{{"$sum", "$amount"}}},
			{"count", bson.D{{"$sum", 1}}},
			{"unique_players", bson.D{{"$addToSet", "$player_id"}}},
		}}},
		{{"$project", bson.D{
			{"_id", 0},
			{"period", "$_id.period"},
			{"event_type", "$_id.event_type"},
			{"currency", "$_id.currency"},
			{"total_amount", 1},
			{"count", 1},
			{"unique_players", bson.D{{"$size", "$unique_players"}}},
		}}},
		{{"$sort", bson.D{{"period", 1}}}},
	}

	cursor, err := h.mongoDB.Collection("analytics_events").Aggregate(ctx, pipeline)
	if err != nil {
		h.logger.Warn("Revenue query failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "query failed"})
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		h.logger.Warn("Revenue cursor failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "cursor failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": results})
}

// ============================================================
// 管理后台 — 漏斗分析
// ============================================================

// FunnelsRequest 漏斗查询参数。
type FunnelsRequest struct {
	StartDate string `form:"start_date"` // YYYY-MM-DD
	EndDate   string `form:"end_date"`   // YYYY-MM-DD
}

// Funnels 查询事件漏斗数据。
// 路径: GET /api/v1/admin/analytics/funnels?start_date=2025-01-01&end_date=2025-01-31
func (h *Handler) Funnels(c *gin.Context) {
	if h.mongoDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "MongoDB not available"})
		return
	}

	var req FunnelsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid params"})
		return
	}

	now := time.Now()
	if req.EndDate == "" {
		req.EndDate = now.Format("2006-01-02")
	}
	if req.StartDate == "" {
		req.StartDate = now.AddDate(0, 0, -7).Format("2006-01-02")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 注册 → 登录 → 修炼 → 战斗 转化漏斗
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"event_type", bson.D{{"$in", bson.A{
				EventPlayerRegister,
				EventPlayerLogin,
				EventCultivationStart,
				EventCombatStart,
			}}}},
			{"event_date", bson.D{{"$gte", req.StartDate}, {"$lte", req.EndDate}}},
		}}},
		{{"$group", bson.D{
			{"_id", "$event_type"},
			{"unique_players", bson.D{{"$addToSet", "$player_id"}}},
			{"total_events", bson.D{{"$sum", 1}}},
		}}},
		{{"$project", bson.D{
			{"_id", 0},
			{"event_type", "$_id"},
			{"unique_players", bson.D{{"$size", "$unique_players"}}},
			{"total_events", 1},
		}}},
		{{"$sort", bson.D{{"total_events", -1}}}},
	}

	cursor, err := h.mongoDB.Collection("analytics_events").Aggregate(ctx, pipeline)
	if err != nil {
		h.logger.Warn("Funnels query failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "query failed"})
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		h.logger.Warn("Funnels cursor failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "cursor failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": results})
}

// ============================================================
// 管理后台 — 引擎状态
// ============================================================

// EngineStats 返回分析引擎运行状态。
// 路径: GET /api/v1/admin/analytics/stats
func (h *Handler) EngineStats(c *gin.Context) {
	stats := h.analytics.GetStats()
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": stats})
}

// ============================================================
// 工具函数 — 列表转指针
// ============================================================

// Float64Ptr 将 float64 转换为指针。
func Float64Ptr(v float64) *float64 {
	return &v
}
