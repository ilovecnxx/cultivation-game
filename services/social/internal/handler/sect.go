package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/social/internal/model"
	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// SectHandler 宗门 HTTP 处理器
type SectHandler struct {
	svc *service.SectService
}

// NewSectHandler 创建宗门处理器
func NewSectHandler(svc *service.SectService) *SectHandler {
	return &SectHandler{svc: svc}
}

// CreateSect 创建宗门
// @Router POST /api/v1/sect/create [post]
func (h *SectHandler) CreateSect(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Notice      string `json:"notice"`
		LeaderID    string `json:"leader_id"`
		LeaderName  string `json:"leader_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	sect, err := h.svc.CreateSect(c.Request.Context(), req.Name, req.Description, req.Notice, req.LeaderID, req.LeaderName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sect})
}

// GetSect 获取宗门信息
// @Router GET /api/v1/sect/:id [get]
func (h *SectHandler) GetSect(c *gin.Context) {
	sectID := c.Param("id")
	sect, err := h.svc.GetSect(c.Request.Context(), sectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sect})
}

// GetUserSect 获取玩家所在宗门
// @Router GET /api/v1/sect/my [get]
func (h *SectHandler) GetUserSect(c *gin.Context) {
	userID := c.Query("user_id")
	sect, member, err := h.svc.GetUserSect(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sect": sect, "member": member})
}

// SearchSect 搜索宗门
// @Router GET /api/v1/sect/search [get]
func (h *SectHandler) SearchSect(c *gin.Context) {
	keyword := c.Query("keyword")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.ParseInt(pageStr, 10, 64)
	pageSize, _ := strconv.ParseInt(pageSizeStr, 10, 64)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	sects, total, err := h.svc.SearchSect(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sects, "total": total})
}

// JoinSect 申请加入宗门
// @Router POST /api/v1/sect/join [post]
func (h *SectHandler) JoinSect(c *gin.Context) {
	var req struct {
		SectID   string `json:"sect_id"`
		UserID   string `json:"user_id"`
		UserName string `json:"user_name"`
		Message  string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.JoinSect(c.Request.Context(), req.SectID, req.UserID, req.UserName, req.Message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "申请已提交"})
}

// HandleApply 处理宗门申请
// @Router POST /api/v1/sect/handle-apply [post]
func (h *SectHandler) HandleApply(c *gin.Context) {
	var req struct {
		ApplyID    string `json:"apply_id"`
		Accept     bool   `json:"accept"`
		OperatorID string `json:"operator_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.HandleApply(c.Request.Context(), req.ApplyID, req.Accept, req.OperatorID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "操作成功"})
}

// LeaveSect 退出宗门
// @Router POST /api/v1/sect/leave [post]
func (h *SectHandler) LeaveSect(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id"`
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.LeaveSect(c.Request.Context(), req.SectID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已退出宗门"})
}

// KickMember 踢出成员
// @Router POST /api/v1/sect/kick [post]
func (h *SectHandler) KickMember(c *gin.Context) {
	var req struct {
		SectID     string `json:"sect_id"`
		OperatorID string `json:"operator_id"`
		TargetID   string `json:"target_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.KickMember(c.Request.Context(), req.SectID, req.OperatorID, req.TargetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已踢出成员"})
}

// TransferLeader 转让宗主
// @Router POST /api/v1/sect/transfer-leader [post]
func (h *SectHandler) TransferLeader(c *gin.Context) {
	var req struct {
		SectID          string `json:"sect_id"`
		CurrentLeaderID string `json:"current_leader_id"`
		NewLeaderID     string `json:"new_leader_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.TransferLeader(c.Request.Context(), req.SectID, req.CurrentLeaderID, req.NewLeaderID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "转让成功"})
}

// SetMemberRank 设置成员职位
// @Router POST /api/v1/sect/set-rank [post]
func (h *SectHandler) SetMemberRank(c *gin.Context) {
	var req struct {
		SectID     string          `json:"sect_id"`
		OperatorID string          `json:"operator_id"`
		TargetID   string          `json:"target_id"`
		NewRank    model.SectRank  `json:"new_rank"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.SetMemberRank(c.Request.Context(), req.SectID, req.OperatorID, req.TargetID, req.NewRank); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "设置成功"})
}

// AddContribution 增加贡献(由其他服务调用)
// @Router POST /api/v1/sect/contribution [post]
func (h *SectHandler) AddContribution(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id"`
		UserID string `json:"user_id"`
		Amount int64  `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	amount, err := h.svc.AddContribution(c.Request.Context(), req.SectID, req.UserID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"amount": amount})
}

// GetContributionRank 获取贡献排行
// @Router GET /api/v1/sect/contribution-rank [get]
func (h *SectHandler) GetContributionRank(c *gin.Context) {
	sectID := c.Query("sect_id")
	limitStr := c.DefaultQuery("limit", "10")

	limit, _ := strconv.ParseInt(limitStr, 10, 64)
	if limit < 1 || limit > 50 {
		limit = 10
	}

	members, err := h.svc.GetContributionRank(c.Request.Context(), sectID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": members})
}

// GetSectSkills 获取宗门技能列表
// @Router GET /api/v1/sect/skills [get]
func (h *SectHandler) GetSectSkills(c *gin.Context) {
	sectID := c.Query("sect_id")
	skills, err := h.svc.GetSectSkills(c.Request.Context(), sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": skills})
}

// LearnSkill 学习/升级技能
// @Router POST /api/v1/sect/learn-skill [post]
func (h *SectHandler) LearnSkill(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id"`
		UserID string `json:"user_id"`
		SkillID string `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.svc.LearnSkill(c.Request.Context(), req.SectID, req.UserID, req.SkillID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "学习成功"})
}

// GetMemberSkills 获取成员技能
// @Router GET /api/v1/sect/my-skills [get]
func (h *SectHandler) GetMemberSkills(c *gin.Context) {
	userID := c.Query("user_id")
	skills, err := h.svc.GetMemberSkills(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": skills})
}
