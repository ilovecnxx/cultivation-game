// Package handler 宗门科技树 HTTP 处理器
package handler

import (
	"net/http"

	"cultivation-game/services/social/internal/service"

	"github.com/gin-gonic/gin"
)

// SectTechHandler 宗门科技树 HTTP 处理器
type SectTechHandler struct {
	sectTechService *service.SectTechService
}

// NewSectTechHandler 创建 SectTechHandler
func NewSectTechHandler(sectTechService *service.SectTechService) *SectTechHandler {
	return &SectTechHandler{sectTechService: sectTechService}
}

// ListTechs 获取宗门科技列表
// GET /api/v1/sect/tech/list?sect_id=xxx
func (h *SectTechHandler) ListTechs(c *gin.Context) {
	sectID := c.Query("sect_id")
	if sectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "sect_id 不能为空"})
		return
	}

	techs, err := h.sectTechService.GetTechList(c.Request.Context(), sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": techs,
	})
}

// UpgradeTech 升级宗门科技
// POST /api/v1/sect/tech/upgrade
func (h *SectTechHandler) UpgradeTech(c *gin.Context) {
	var req struct {
		SectID string `json:"sect_id" binding:"required"`
		UserID string `json:"user_id" binding:"required"`
		Branch string `json:"branch" binding:"required"` // 分支ID
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误", "detail": err.Error()})
		return
	}

	tech, err := h.sectTechService.UpgradeTech(c.Request.Context(), req.SectID, req.UserID, req.Branch)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "科技升级成功",
		"data": tech,
	})
}
