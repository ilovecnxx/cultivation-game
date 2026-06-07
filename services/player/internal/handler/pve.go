package handler

import (
	"math/rand"
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PveHandler struct {
	playerService *service.PlayerService
	log           *zap.Logger
}

func NewPveHandler(playerService *service.PlayerService, log *zap.Logger) *PveHandler {
	return &PveHandler{playerService: playerService, log: log}
}

type PveRound struct {
	Round       int    `json:"round"`
	PlayerDmg   int64  `json:"player_dmg"`
	MonsterDmg  int64  `json:"monster_dmg"`
	PlayerHP    int64  `json:"player_hp"`
	MonsterHP   int64  `json:"monster_hp"`
	PlayerMP    int64  `json:"player_mp"`
	Description string `json:"desc"`
}

// Fight PVE 战斗 — 主动挑战地图怪物
func (h *PveHandler) Fight(c *gin.Context) {
	idStr := c.Param("id")
	playerID, _ := strconv.ParseInt(idStr, 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	var req struct{ LocationKey string `json:"loc_key"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil || player.HP <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": gin.H{"success": false, "reason": "无法战斗"}})
		return
	}
	if player.SpiritSense < 2 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": gin.H{"success": false, "reason": "神识不足"}})
		return
	}
	player.SpiritSense -= 2

	// 根据位置随机怪物等级(±1级浮动)
	locRealm := player.Realm
	if req.LocationKey != "" {
		locRealm = int32(rand.Intn(3) + int(player.Realm) - 1)
		if locRealm < 1 { locRealm = 1 }
		if locRealm > model.RealmTrib { locRealm = model.RealmTrib }
	}
	msAtk, msDef, msHP, _, msSpd, _, _, msDG, msHT, msName := generateMonster(locRealm)

	pHP, pMP := player.HP, player.MP
	var rounds []PveRound
	playerDead, playerWon := false, false
	for round := 0; round < 10 && pHP > 0 && msHP > 0; round++ {
		r := PveRound{Round: round + 1}
		playerFirst := player.Speed >= msSpd

		if !playerFirst {
			mDmg := msAtk - player.Defense/4 + rand.Int63n(msAtk/2+1)
			if mDmg < 1 { mDmg = 1 }
			if rand.Intn(100) < int(player.Dodge/100) { mDmg = 0; r.Description = "闪避！" }
			if rand.Intn(100) >= int(msHT/100) { mDmg = 0 }
			r.MonsterDmg = mDmg; pHP -= mDmg
		}
		if pHP > 0 {
			pDmg := player.Attack - msDef/4 + rand.Int63n(player.Attack/2+1)
			if pDmg < 1 { pDmg = 1 }
			if rand.Intn(100) < int(player.CritRate/100) { pDmg = pDmg * player.CritDmg / 10000; r.Description = "暴击！" }
			r.PlayerDmg = pDmg; msHP -= pDmg
			if pDmg > 0 && rand.Intn(100) < int(msDG/100) { msHP += pDmg; r.PlayerDmg = 0; r.Description += " 怪物闪避！" }
		}
		if playerFirst && msHP > 0 {
			mDmg := msAtk - player.Defense/4 + rand.Int63n(msAtk/2+1)
			if mDmg < 1 { mDmg = 1 }
			if rand.Intn(100) < int(player.Dodge/100) { mDmg = 0; if r.Description == "" { r.Description = "闪避！" } }
			r.MonsterDmg = mDmg; pHP -= mDmg
		}
		pMP -= 5; if pMP < 0 { pMP = 0; if r.PlayerDmg > 0 { r.PlayerDmg = r.PlayerDmg * 2 / 3 } }
		r.PlayerHP = pHP; r.MonsterHP = msHP; r.PlayerMP = pMP
		rounds = append(rounds, r)
	}
	player.HP = pHP; player.MP = pMP
	if pHP <= 0 { player.HP = 0; playerDead = true; playerWon = false } else { playerWon = msHP <= 0 }

	cultGain := int64(0); goldGain := int64(0)
	if playerWon {
		cultGain = int64(player.Realm)*15 + rand.Int63n(int64(player.Realm)*30+1)
		goldGain = int64(player.Realm)*8 + rand.Int63n(int64(player.Realm)*20+1)
		player.SpiritPower += cultGain; if player.SpiritPower > player.MaxSpirit { player.SpiritPower = player.MaxSpirit }
		player.Gold += goldGain
	}
	_ = h.playerService.UpdatePlayer(c.Request.Context(), player)

	resp := gin.H{"success": true, "won": playerWon, "dead": playerDead, "cult_gain": cultGain, "gold_gain": goldGain,
		"hp": player.HP, "mp": player.MP, "sense_left": player.SpiritSense, "monster": msName, "rounds": rounds}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": resp})
}
