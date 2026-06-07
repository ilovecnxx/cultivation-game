package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PetHandler 灵兽 HTTP 处理器
type PetHandler struct {
	petService *service.PetService
	log        *zap.Logger
}

// NewPetHandler 创建 PetHandler
func NewPetHandler(petService *service.PetService, log *zap.Logger) *PetHandler {
	return &PetHandler{
		petService: petService,
		log:        log,
	}
}

// parsePlayerID 从路径参数解析玩家ID
func (h *PetHandler) parsePlayerID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

// parsePetID 从查询参数或路径参数解析灵兽ID
func (h *PetHandler) parsePetID(c *gin.Context) (int64, error) {
	// 优先从路径 param 取
	if pid := c.Param("pet_id"); pid != "" {
		return strconv.ParseInt(pid, 10, 64)
	}
	// 其次从查询参数取
	pid := c.Query("pet_id")
	if pid != "" {
		return strconv.ParseInt(pid, 10, 64)
	}
	return 0, nil
}

// TryEncounter 游历遭遇野生灵兽
// POST /api/v1/player/:id/pet/encounter
func (h *PetHandler) TryEncounter(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	encountered, species, err := h.petService.TryEncounter(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("灵兽遭遇失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if !encountered {
		c.JSON(http.StatusOK, gin.H{
			"code":       0,
			"msg":        "本次游历未发现野生灵兽",
			"encountered": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"msg":        "发现野生灵兽！",
		"encountered": true,
		"data": gin.H{
			"species_id":  species.ID,
			"name":        species.Name,
			"star":        species.Star,
			"description": species.Description,
		},
	})
}

// Capture 捕捉灵兽
// POST /api/v1/player/:id/pet/capture
func (h *PetHandler) Capture(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	var req struct {
		SpeciesID string `json:"species_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	pet, err := h.petService.Capture(c.Request.Context(), playerID, req.SpeciesID)
	if err != nil {
		h.log.Warn("捕捉灵兽失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "捕捉成功！",
		"data": pet,
	})
}

// Rename 重命名灵兽
// POST /api/v1/player/:id/pet/:pet_id/rename
func (h *PetHandler) Rename(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	petID, err := h.parsePetID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的灵兽ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=16"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	pet, err := h.petService.Rename(c.Request.Context(), playerID, petID, req.Name)
	if err != nil {
		h.log.Warn("重命名灵兽失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "重命名成功",
		"data": pet,
	})
}

// Feed 喂食灵兽经验丹
// POST /api/v1/player/:id/pet/:pet_id/feed
func (h *PetHandler) Feed(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	petID, err := h.parsePetID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的灵兽ID"})
		return
	}

	var req struct {
		Quantity int32 `json:"quantity" binding:"required,min=1,max=999"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	pet, newLevels, err := h.petService.Feed(c.Request.Context(), playerID, petID, req.Quantity)
	if err != nil {
		h.log.Warn("喂食灵兽失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	resp := gin.H{
		"code": 0,
		"msg":  "喂食成功",
		"data": gin.H{
			"pet":          pet,
			"exp":          pet.Exp,
			"level":        pet.Level,
			"new_levels":   newLevels,
			"levels_gained": len(newLevels),
		},
	}
	if len(newLevels) > 0 {
		resp["msg"] = "喂食成功，灵兽升级了！"
	}
	c.JSON(http.StatusOK, resp)
}

// GetLevelUpInfo 获取灵兽升级进度
// GET /api/v1/player/:id/pet/:pet_id/level-info
func (h *PetHandler) GetLevelUpInfo(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	petID, err := h.parsePetID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的灵兽ID"})
		return
	}

	currentExp, needExp, level, err := h.petService.GetLevelUpInfo(c.Request.Context(), playerID, petID)
	if err != nil {
		h.log.Warn("查询灵兽升级信息失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"current_exp": currentExp,
			"need_exp":    needExp,
			"level":       level,
			"is_max_level": needExp == 0 && level >= 100,
		},
	})
}

// Evolve 灵兽进化（提升星级）
// POST /api/v1/player/:id/pet/:pet_id/evolve
func (h *PetHandler) Evolve(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	petID, err := h.parsePetID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的灵兽ID"})
		return
	}

	pet, err := h.petService.Evolve(c.Request.Context(), playerID, petID)
	if err != nil {
		h.log.Warn("灵兽进化失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "进化成功！灵兽星级提升！",
		"data": gin.H{
			"pet":       pet,
			"star":      pet.Star,
			"new_level": pet.Level,
		},
	})
}

// SetActive 设置出战灵兽
// POST /api/v1/player/:id/pet/:pet_id/active
func (h *PetHandler) SetActive(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	petID, err := h.parsePetID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的灵兽ID"})
		return
	}

	pet, err := h.petService.SetActive(c.Request.Context(), playerID, petID)
	if err != nil {
		h.log.Warn("设置出战灵兽失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "出战灵兽设置成功",
		"data": pet,
	})
}

// UnsetActive 取消出战灵兽
// POST /api/v1/player/:id/pet/:pet_id/deactivate
func (h *PetHandler) UnsetActive(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	petID, err := h.parsePetID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的灵兽ID"})
		return
	}

	pet, err := h.petService.UnsetActive(c.Request.Context(), playerID, petID)
	if err != nil {
		h.log.Warn("取消出战灵兽失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "已取消出战",
		"data": pet,
	})
}

// ListPets 获取玩家所有灵兽
// GET /api/v1/player/:id/pet
func (h *PetHandler) ListPets(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}

	pets, err := h.petService.ListPets(c.Request.Context(), playerID)
	if err != nil {
		h.log.Warn("查询灵兽列表失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": pets,
	})
}

// GetPet 获取单只灵兽详情
// GET /api/v1/player/:id/pet/:pet_id
func (h *PetHandler) GetPet(c *gin.Context) {
	playerID, err := h.parsePlayerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	petID, err := h.parsePetID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的灵兽ID"})
		return
	}

	pet, err := h.petService.GetPetByID(c.Request.Context(), playerID, petID)
	if err != nil {
		h.log.Warn("查询灵兽详情失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": pet,
	})
}
