package handler

import (
	"database/sql"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type EquipHandler struct{ db *sql.DB }

func NewEquipHandler(db *sql.DB) *EquipHandler { return &EquipHandler{db: db} }

var equipQualityMult = []float64{0.6, 1.0, 1.4, 1.8, 2.5}

// ListPlayerEquipment 列出玩家已装备和可制作的装备
func (h *EquipHandler) ListEquipment(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权"})
		return
	}
	// 玩家已装备
	type Equipped struct {
		Slot    string `json:"slot"`
		ItemID  int    `json:"item_id"`
		Name    string `json:"name"`
		Icon    string `json:"icon"`
		Quality int    `json:"quality"`
		Tier    int    `json:"tier"`
	}
	var equipped []Equipped
	rows, _ := h.db.Query(`SELECT pe.slot, eb.id, eb.name, eb.icon, pe.quality, eb.tier FROM player_equipment pe JOIN equipment_base eb ON pe.item_id=eb.id WHERE pe.player_id=?`, playerID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() { var e Equipped; rows.Scan(&e.Slot, &e.ItemID, &e.Name, &e.Icon, &e.Quality, &e.Tier); equipped = append(equipped, e) }
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": equipped})
}

// CraftEquipment 打造装备
func (h *EquipHandler) CraftEquipment(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权"})
		return
	}
	var req struct{ Slot string `json:"slot"`; Tier int `json:"tier"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.Tier < 1 || req.Tier > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 查模板
	var itemID, atk, def, hp, mp, spd, cr, cd, dg, ht, mr int
	var name, icon string
	err := h.db.QueryRow("SELECT id,name,icon,atk,def,hp,mp,spd,crit,critdmg,dodge,hit,mpregen FROM equipment_base WHERE slot=? AND tier=?", req.Slot, req.Tier).
		Scan(&itemID, &name, &icon, &atk, &def, &hp, &mp, &spd, &cr, &cd, &dg, &ht, &mr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "该部位暂无此等级装备"})
		return
	}
	// 品质判定
	quality := rollEquipQuality()
	// 存/更新装备
	h.db.Exec("INSERT INTO player_equipment (player_id,slot,item_id,quality) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE item_id=?,quality=?", playerID, req.Slot, itemID, quality, itemID, quality)
	mult := equipQualityMult[quality]
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"slot": req.Slot, "name": name, "icon": icon, "quality": quality,
		"atk": int(float64(atk) * mult), "def": int(float64(def) * mult),
		"hp": int(float64(hp) * mult), "mp": int(float64(mp) * mult), "spd": int(float64(spd) * mult),
	}})
}

// Unequip 卸下装备
func (h *EquipHandler) Unequip(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权"})
		return
	}
	var req struct{ Slot string `json:"slot"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	h.db.Exec("DELETE FROM player_equipment WHERE player_id=? AND slot=?", playerID, req.Slot)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已卸下"})
}

// GetEquipTemplates 获取当前境界可打造的装备模板
func (h *EquipHandler) GetTemplates(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// 查玩家境界
	var realm int
	h.db.QueryRow("SELECT realm_id FROM players WHERE id=?", playerID).Scan(&realm)
	if realm < 1 { realm = 1 }
	rows, _ := h.db.Query("SELECT id,slot,name,icon,tier,atk,def,hp,mp,spd,crit,critdmg,dodge,hit,mpregen FROM equipment_base WHERE tier<=? ORDER BY slot,tier", realm)
	defer rows.Close()
	type Tmpl struct {
		ID     int    `json:"id"`
		Slot   string `json:"slot"`
		Name   string `json:"name"`
		Icon   string `json:"icon"`
		Tier   int    `json:"tier"`
		Atk    int    `json:"atk"`
		Def    int    `json:"def"`
		HP     int    `json:"hp"`
		MP     int    `json:"mp"`
		Spd    int    `json:"spd"`
		Crit   int    `json:"crit"`
	}
	var list []Tmpl
	for rows.Next() { var t Tmpl; rows.Scan(&t.ID, &t.Slot, &t.Name, &t.Icon, &t.Tier, &t.Atk, &t.Def, &t.HP, &t.MP, &t.Spd, &t.Crit); list = append(list, t) }
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

func rollEquipQuality() int {
	r := rand.Intn(100)
	if r < 5 { return 4 }  // 完美 5%
	if r < 15 { return 3 }  // 精良 10%
	if r < 35 { return 2 }  // 优良 20%
	if r < 65 { return 1 }  // 普通 30%
	return 0                 // 劣质 35%
}
