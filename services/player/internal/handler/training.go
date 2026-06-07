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

type TrainingHandler struct {
	playerService *service.PlayerService
	log           *zap.Logger
}

func NewTrainingHandler(playerService *service.PlayerService, log *zap.Logger) *TrainingHandler {
	return &TrainingHandler{playerService: playerService, log: log}
}

type CombatRound struct {
	Round       int    `json:"round"`
	PlayerDmg   int64  `json:"player_dmg"`
	MonsterDmg  int64  `json:"monster_dmg"`
	PlayerHP    int64  `json:"player_hp"`
	MonsterHP   int64  `json:"monster_hp"`
	PlayerMP    int64  `json:"player_mp"`
	MonsterName string `json:"monster_name"`
	Description string `json:"desc"`
}

type TrainRequest struct {
	Multiplier int `json:"multiplier" binding:"required,min=1,max=10"`
}

// generateMonster 根据境界生成怪物属性（同境界玩家基准）
func generateMonster(realm int32) (atk, def, hp, mp, spd, cr, cd, dg, ht int64, name string) {
	atk, def, hp, mp, spd, cr, cd, dg, ht, _, _, _, _ = model.CalcRealmAttributes(realm, 1)
	// 怪物有 ±20% 的随机浮动
	jitter := func(v int64) int64 { return v*80/100 + rand.Int63n(v*40/100+1) }
	return jitter(atk), jitter(def), jitter(hp), jitter(mp), jitter(spd), jitter(cr), jitter(cd), jitter(dg), jitter(ht), model.RealmNames[realm]
}

func (h *TrainingHandler) Train(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的玩家ID"})
		return
	}
	jwtPlayerID, exists := c.Get("player_id")
	if !exists || jwtPlayerID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作此角色"})
		return
	}
	var req TrainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	mult := req.Multiplier

	player, err := h.playerService.GetPlayer(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "玩家不存在"})
		return
	}
	if player.HP <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": gin.H{"success": false, "reason": "角色已死亡，等待复活", "dead": true}})
		return
	}
	cost := int64(mult)
	if player.SpiritSense < cost {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": gin.H{"success": false, "reason": "神识不足"}})
		return
	}
	player.SpiritSense -= cost

	cultGain := (int64(player.Realm)*10 + rand.Int63n(int64(player.Realm)*20+1)) * int64(mult)
	player.SpiritPower += cultGain
	if player.SpiritPower > player.MaxSpirit {
		player.SpiritPower = player.MaxSpirit
	}
	goldGain := (int64(player.Realm)*5 + rand.Int63n(int64(player.Realm)*15+1)) * int64(mult)
	player.Gold += goldGain

	// 战斗遭遇：基础30% × 倍率，上限100%
	combatChance := 30 * mult
	if combatChance > 100 {
		combatChance = 100
	}
	var combatRounds []CombatRound
	var combatResult string
	var encounter string
	playerDead := false

	if rand.Intn(100) < combatChance {
		// 怪物等级：可上2级下3级。气运越高越不容易遇强怪
		// 基础强怪概率45%，气运/100 降低强怪概率（气运100→强怪20%，气运0→强怪45%）
		strongChance := 45 - int(player.Luck*25/100)
		if strongChance < 15 {
			strongChance = 15
		}
		stronger := rand.Intn(100) < strongChance
		var msLevel int32
		if stronger {
			offset := 1 + rand.Intn(2) // +1 or +2
			msLevel = int32(player.Realm) + int32(offset)
			if msLevel > model.RealmTrib {
				msLevel = model.RealmTrib
			}
		} else {
			offset := 1 + rand.Intn(3) // -1 to -3
			msLevel = int32(player.Realm) - int32(offset)
			if msLevel < 1 {
				msLevel = 1
			}
		}
		msAtk, msDef, msHP, _, msSpd, _, _, msDG, msHT, msName := generateMonster(msLevel)
		// 怪物额外奖励 = 境界差 × 境界基础灵石
		if stronger {
			goldGain += int64(msLevel-player.Realm) * 10 * int64(mult)
		}

		combatRounds = make([]CombatRound, 0)
		round := 0
		maxRounds := 10
		pHP := player.HP
		pMP := player.MP

		for round < maxRounds && pHP > 0 && msHP > 0 {
			round++
			cr := CombatRound{Round: round, MonsterName: msName}

			// 玩家攻击：先手判定（速度高先出手）
			playerFirst := player.Speed >= msSpd
			if !playerFirst {
				// 怪物先手
				mDmg := msAtk - player.Defense/4 + rand.Int63n(msAtk/2+1)
				if mDmg < 1 {
					mDmg = 1
				}
				isDodge := rand.Intn(100) < int(player.Dodge/100)
				if isDodge {
					mDmg = 0
					cr.Description = "闪避！"
				}
				// 怪物命中判定
				if rand.Intn(100) >= int(msHT/100) {
					mDmg = 0
					if cr.Description == "" {
						cr.Description = "怪物未命中"
					}
				}
				cr.MonsterDmg = mDmg
				pHP -= mDmg
			}

			// 玩家攻击（如果还活着）
			if pHP > 0 {
				pDmg := player.Attack - msDef/4 + rand.Int63n(player.Attack/2+1)
				if pDmg < 1 {
					pDmg = 1
				}
				// 命中判定
				if rand.Intn(100) >= int(player.Hit/100) {
					pDmg = 0
					if cr.Description == "" {
						cr.Description = "未命中"
					}
				} else {
					// 暴击判定（含气运加成：气运/200）
					luckBonus := int(player.Luck) / 200
					critChance := int(player.CritRate/100) + luckBonus
					if rand.Intn(100) < critChance {
						pDmg = pDmg * player.CritDmg / 10000
						s := "暴击！"
						if cr.Description != "" {
							s = " " + s
						}
						cr.Description += s
					}
				}
				cr.PlayerDmg = pDmg
				msHP -= pDmg
				// 闪避判定（怪物也有闪避）
				if pDmg > 0 && rand.Intn(100) < int(msDG/100) {
					msHP += pDmg // 恢复伤害
					cr.PlayerDmg = 0
					if cr.Description != "" {
						cr.Description += " "
					}
					cr.Description += "怪物闪避！"
				}
			}

			// 怪物攻击（如果玩家先手且怪物还活着）
			if playerFirst && msHP > 0 {
				mDmg := msAtk - player.Defense/4 + rand.Int63n(msAtk/2+1)
				if mDmg < 1 {
					mDmg = 1
				}
				isDodge := rand.Intn(100) < int(player.Dodge/100)
				if isDodge {
					mDmg = 0
					if cr.Description != "" {
						cr.Description += " "
					}
					cr.Description += "闪避！"
				}
				if rand.Intn(100) >= int(msHT/100) {
					mDmg = 0
				}
				cr.MonsterDmg = mDmg
				pHP -= mDmg
			}

			// 灵力消耗
			mpCost := int64(5)
			pMP -= mpCost
			if pMP < 0 {
				pMP = 0
				if cr.PlayerDmg > 0 {
					cr.PlayerDmg = cr.PlayerDmg * 2 / 3
				}
			}

			cr.PlayerHP = pHP
			cr.MonsterHP = msHP
			cr.PlayerMP = pMP
			combatRounds = append(combatRounds, cr)
		}

		player.HP = pHP
		player.MP = pMP
		if pHP <= 0 {
			playerDead = true
			player.HP = 0
			combatResult = "战败"
			cultGain = 0
			goldGain = 0
		} else if stronger {
			combatResult = "苦战获胜"
		} else {
			combatResult = "轻松取胜"
		}
	}

	_ = h.playerService.UpdatePlayer(c.Request.Context(), player)

	resp := gin.H{
		"success":    true,
		"cult_gain":  cultGain,
		"gold_gain":  goldGain,
		"hp":         player.HP,
		"mp":         player.MP,
		"sense_left": player.SpiritSense,
		"dead":       playerDead,
		"combat":     combatResult,
	}
	if encounter != "" {
		resp["encounter"] = encounter
		resp["new_quality"] = player.RootQuality
		resp["quality_name"] = model.RootQualityNames[player.RootQuality]
		resp["spirit_name"] = model.SpiritRootNames[player.SpiritRoot]
		resp["comprehension"] = player.Comprehension
		resp["spirit_sense"] = player.SpiritSense
		resp["luck"] = player.Luck
	}
	if len(combatRounds) > 0 {
		resp["rounds"] = combatRounds
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": resp})
}
