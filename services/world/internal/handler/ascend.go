// Package handler 飞升仙界 - 世界服务处理器
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// XianjieRegion 仙界区域数据结构
type XianjieRegion struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Description   string            `json:"description"`
	LevelMin      int               `json:"level_min"`
	LevelMax      int               `json:"level_max"`
	DangerLevel   int               `json:"danger_level"`
	SpiritDensity float64           `json:"spirit_density"`
	Connections   []string          `json:"connections"`
	Resources     XianjieResources  `json:"resources"`
}

// XianjieResources 仙界区域资源
type XianjieResources struct {
	SpiritQi float64               `json:"spirit_qi"`
	Items    []XianjieResourceItem `json:"items"`
}

// XianjieResourceItem 资源物品
type XianjieResourceItem struct {
	ItemID string  `json:"item_id"`
	Rate   float64 `json:"rate"`
}

// AscendAnnouncement 飞升公告
type AscendAnnouncement struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	PlayerID  uint64 `json:"player_id"`
	Timestamp int64  `json:"timestamp"`
}

// CelestialPhenomenon 天降异象
type CelestialPhenomenon struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	StartAt  int64  `json:"start_at"`
	Duration int64  `json:"duration"` // 持续时间（秒）
	RegionID string `json:"region_id"`
}

// AscendHandler 世界服务飞升处理器
type AscendHandler struct {
	xianjieRegions []XianjieRegion
	announcements  []AscendAnnouncement
	phenomena      []CelestialPhenomenon
	mu             sync.RWMutex
}

// NewAscendHandler 创建世界服务飞升处理器
func NewAscendHandler(dataDir string) *AscendHandler {
	h := &AscendHandler{
		announcements: make([]AscendAnnouncement, 0, 100),
		phenomena:     make([]CelestialPhenomenon, 0, 10),
	}
	h.loadXianjieRegions(dataDir)
	return h
}

// loadXianjieRegions 加载仙界区域配置
func (h *AscendHandler) loadXianjieRegions(dataDir string) {
	// 尝试多个路径找到数据文件
	paths := []string{
		dataDir + "/xianjie_regions.json",
		"internal/data/xianjie_regions.json",
		"../data/xianjie_regions.json",
	}
	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			log.Printf("[仙界] 加载区域配置文件: %s", p)
			break
		}
	}
	if err != nil {
		log.Printf("[仙界] 未找到区域配置，使用内置默认区域")
		h.xianjieRegions = h.defaultRegions()
		return
	}

	var regions []XianjieRegion
	if err := json.Unmarshal(data, &regions); err != nil {
		log.Printf("[仙界] 加载区域配置失败: %v，使用内置默认", err)
		h.xianjieRegions = h.defaultRegions()
		return
	}
	h.xianjieRegions = regions
	log.Printf("[仙界] 成功加载 %d 个仙界区域", len(regions))
}

// defaultRegions 内置默认仙界区域
func (h *AscendHandler) defaultRegions() []XianjieRegion {
	return []XianjieRegion{
		{
			ID: "xianjie_01", Name: "南天门", Type: "secret_realm",
			Description: "仙界入口，巍峨天门矗立云端，灵气浓郁如实质。",
			LevelMin: 100, LevelMax: 200, DangerLevel: 5, SpiritDensity: 8.0,
			Connections: []string{"xianjie_02", "xianjie_03"},
		},
		{
			ID: "xianjie_02", Name: "蟠桃园", Type: "secret_realm",
			Description: "广袤千里的仙家果园，三千株蟠桃树灵气氤氲。",
			LevelMin: 120, LevelMax: 280, DangerLevel: 4, SpiritDensity: 9.0,
			Connections: []string{"xianjie_01", "xianjie_04"},
		},
		{
			ID: "xianjie_03", Name: "瑶池仙境", Type: "secret_realm",
			Description: "碧波万顷的瑶池，池水泛着七彩霞光，灵气最为纯净。",
			LevelMin: 150, LevelMax: 320, DangerLevel: 3, SpiritDensity: 10.0,
			Connections: []string{"xianjie_01", "xianjie_05", "xianjie_06"},
		},
		{
			ID: "xianjie_04", Name: "兜率宫", Type: "town",
			Description: "太上老君的道场，宫中炼丹炉昼夜不息。",
			LevelMin: 180, LevelMax: 400, DangerLevel: 2, SpiritDensity: 7.5,
			Connections: []string{"xianjie_02", "xianjie_07"},
		},
		{
			ID: "xianjie_05", Name: "灵霄宝殿", Type: "town",
			Description: "天庭中枢，三十六根通天玉柱撑起金碧辉煌的宝殿。",
			LevelMin: 200, LevelMax: 450, DangerLevel: 1, SpiritDensity: 6.0,
			Connections: []string{"xianjie_03", "xianjie_08"},
		},
		{
			ID: "xianjie_06", Name: "广寒宫", Type: "secret_realm",
			Description: "月宫仙境，桂树飘香，寒玉为砖。月光之下修炼可大幅提升道心。",
			LevelMin: 220, LevelMax: 500, DangerLevel: 6, SpiritDensity: 11.0,
			Connections: []string{"xianjie_03", "xianjie_09"},
		},
		{
			ID: "xianjie_07", Name: "天河畔", Type: "dangerous_land",
			Description: "横贯仙界的浩瀚天河，河水蕴含星辰之力。",
			LevelMin: 250, LevelMax: 550, DangerLevel: 7, SpiritDensity: 9.5,
			Connections: []string{"xianjie_04", "xianjie_08", "xianjie_10"},
		},
		{
			ID: "xianjie_08", Name: "斩妖台", Type: "dangerous_land",
			Description: "上古战场遗迹，无数妖神陨落之地，煞气与仙气交织。",
			LevelMin: 280, LevelMax: 600, DangerLevel: 8, SpiritDensity: 8.5,
			Connections: []string{"xianjie_05", "xianjie_07", "xianjie_09"},
		},
		{
			ID: "xianjie_09", Name: "火云洞", Type: "dangerous_land",
			Description: "太古火神遗留下的洞府，洞中三昧真火永不熄灭。",
			LevelMin: 320, LevelMax: 700, DangerLevel: 9, SpiritDensity: 12.0,
			Connections: []string{"xianjie_06", "xianjie_08", "xianjie_10"},
		},
		{
			ID: "xianjie_10", Name: "蓬莱仙岛", Type: "secret_realm",
			Description: "海上仙山之首，浮于天河尽头，传闻仙帝传承藏于此岛最深处。",
			LevelMin: 380, LevelMax: 999, DangerLevel: 10, SpiritDensity: 15.0,
			Connections: []string{"xianjie_07", "xianjie_09"},
		},
	}
}

// RegisterRoutes 注册仙界相关路由
func (h *AscendHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/xianjie/regions", h.handleXianjieRegions)
	r.GET("/api/v1/xianjie/region/:id", h.handleXianjieRegion)
	r.POST("/api/v1/world/ascend/announce", h.handleAscendAnnounce)
	r.GET("/api/v1/world/ascend/phenomena", h.handleAscendPhenomena)
}

// ----- Handlers -----

// handleXianjieRegions 获取仙界区域列表
// GET /api/v1/xianjie/regions
func (h *AscendHandler) handleXianjieRegions(c *gin.Context) {
	h.mu.RLock()
	regions := h.xianjieRegions
	h.mu.RUnlock()

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取仙界区域列表成功",
		Data:    regions,
	})
}

// handleXianjieRegion 获取仙界区域详情
// GET /api/v1/xianjie/region/{id}
func (h *AscendHandler) handleXianjieRegion(c *gin.Context) {
	regionID := c.Param("id")

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, region := range h.xianjieRegions {
		if region.ID == regionID {
			writeJSON(c, http.StatusOK, &apiResponse{
				Code:    0,
				Message: "获取区域详情成功",
				Data:    region,
			})
			return
		}
	}

	writeError(c, http.StatusNotFound, "仙界区域不存在")
}

// handleAscendAnnounce 接收飞升公告（由修炼服务调用）
// POST /api/v1/world/ascend/announce
func (h *AscendHandler) handleAscendAnnounce(c *gin.Context) {
	var announce AscendAnnouncement
	if err := c.ShouldBindJSON(&announce); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	announce.Timestamp = time.Now().Unix()

	h.mu.Lock()
	h.announcements = append(h.announcements, announce)
	// 限制公告数量
	if len(h.announcements) > 100 {
		h.announcements = h.announcements[len(h.announcements)-100:]
	}
	// 同时生成天降异象
	phenomenon := CelestialPhenomenon{
		Type:     "ascend_phenomenon",
		Title:    "天降祥瑞",
		Detail:   announce.Message,
		StartAt:  time.Now().Unix(),
		Duration: 300, // 持续5分钟
		RegionID: "xianjie_01",
	}
	h.phenomena = append(h.phenomena, phenomenon)
	if len(h.phenomena) > 10 {
		h.phenomena = h.phenomena[len(h.phenomena)-10:]
	}
	h.mu.Unlock()

	log.Printf("[仙界] 飞升公告: %s", announce.Message)

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "公告已广播，天降异象已触发",
		Data: map[string]interface{}{
			"announcement": announce,
			"phenomenon":   phenomenon,
		},
	})
}

// handleAscendPhenomena 获取当前天降异象
// GET /api/v1/world/ascend/phenomena
func (h *AscendHandler) handleAscendPhenomena(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now().Unix()
	activePhenomena := make([]CelestialPhenomenon, 0, len(h.phenomena))
	for _, p := range h.phenomena {
		if now < p.StartAt+p.Duration {
			activePhenomena = append(activePhenomena, p)
		}
	}

	writeJSON(c, http.StatusOK, &apiResponse{
		Code:    0,
		Message: "获取天降异象成功",
		Data:    activePhenomena,
	})
}
