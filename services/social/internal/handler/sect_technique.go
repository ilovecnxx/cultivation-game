// Package handler 功法阁 HTTP 处理器
package handler

import (
	"net/http"
	"strconv"

	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// SectTechniqueHandler 功法阁 HTTP 处理器
type SectTechniqueHandler struct {
	svc *service.SectTechniqueService
}

// NewSectTechniqueHandler 创建功法阁处理器
func NewSectTechniqueHandler(svc *service.SectTechniqueService) *SectTechniqueHandler {
	return &SectTechniqueHandler{svc: svc}
}

// GetTechniques 获取可兑换功法列表
// @Router GET /api/v1/sect/technique/list [get]
func (h *SectTechniqueHandler) GetTechniques(c *gin.Context) {
	sectID := c.Query("sect_id")
	realmStr := c.DefaultQuery("realm", "1")

	realm, _ := strconv.Atoi(realmStr)
	if realm < 1 {
		realm = 1
	}
	if realm > 10 {
		realm = 10
	}

	techniques, err := h.svc.GetTechniquesByRealm(c.Request.Context(), sectID, realm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": techniques})
}

// ExchangeTechnique 兑换功法
// @Router POST /api/v1/sect/technique/exchange [post]
func (h *SectTechniqueHandler) ExchangeTechnique(c *gin.Context) {
	var req struct {
		SectID      string `json:"sect_id"`
		UserID      string `json:"user_id"`
		TechniqueID string `json:"technique_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	mt, err := h.svc.ExchangeTechnique(c.Request.Context(), req.SectID, req.UserID, req.TechniqueID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "兑换成功", "data": mt})
}

// UpgradeTechnique 升级功法层数
// @Router POST /api/v1/sect/technique/upgrade [post]
func (h *SectTechniqueHandler) UpgradeTechnique(c *gin.Context) {
	var req struct {
		SectID        string `json:"sect_id"`
		UserID        string `json:"user_id"`
		MemberTechID  string `json:"member_tech_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	mt, err := h.svc.UpgradeTechnique(c.Request.Context(), req.SectID, req.UserID, req.MemberTechID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "升级成功", "data": mt})
}

// GetMyTechniques 获取我的功法
// @Router GET /api/v1/sect/technique/my [get]
func (h *SectTechniqueHandler) GetMyTechniques(c *gin.Context) {
	userID := c.Query("user_id")
	techniques, err := h.svc.GetMyTechniques(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": techniques})
}
