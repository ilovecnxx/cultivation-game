package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BackpackHandler struct {
	db            *sql.DB
	playerService *service.PlayerService
	log           *zap.Logger
}

func NewBackpackHandler(db *sql.DB, ps *service.PlayerService, log *zap.Logger) *BackpackHandler {
	return &BackpackHandler{db: db, playerService: ps, log: log}
}

func (h *BackpackHandler) ListItems(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	type Item struct {
		ID       int64  `json:"id"`
		ItemKey  string `json:"item_key"`
		Name     string `json:"name"`
		Icon     string `json:"icon"`
		ItemType string `json:"item_type"`
		Quantity int    `json:"quantity"`
		Quality  int    `json:"quality"`
	}
	var items []Item
	rows, _ := h.db.Query("SELECT pp.id,pp.pill_key,pr.name,pr.icon,'pill',pp.quantity,pp.quality FROM player_pills pp JOIN pill_recipes pr ON pp.pill_key=pr.pill_key WHERE pp.player_id=? AND pp.quantity>0", playerID)
	if rows != nil { defer rows.Close(); for rows.Next() { var i Item; rows.Scan(&i.ID,&i.ItemKey,&i.Name,&i.Icon,&i.ItemType,&i.Quantity,&i.Quality); items=append(items,i) } }
	rows2, _ := h.db.Query("SELECT id,item_key,name,icon,item_type,quantity,quality FROM player_items WHERE player_id=? AND quantity>0", playerID)
	if rows2 != nil { defer rows2.Close(); for rows2.Next() { var i Item; rows2.Scan(&i.ID,&i.ItemKey,&i.Name,&i.Icon,&i.ItemType,&i.Quantity,&i.Quality); items=append(items,i) } }
	if items == nil { items = []Item{} }
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}
