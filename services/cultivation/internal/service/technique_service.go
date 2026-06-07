package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cultivation-game/services/cultivation/internal/config"
	"cultivation-game/services/cultivation/internal/model"
)

// TechniqueService 功法管理服务
type TechniqueService struct {
	config            *config.ConfigLoader
	playerServiceAddr string // Player 服务 HTTP 地址
}

// NewTechniqueService 创建功法服务实例
func NewTechniqueService(cfg *config.ConfigLoader) *TechniqueService {
	playerAddr := os.Getenv("PLAYER_SERVICE_ADDR")
	if playerAddr == "" {
		playerAddr = "http://127.0.0.1:8082"
	}
	return &TechniqueService{
		config:            cfg,
		playerServiceAddr: playerAddr,
	}
}

// LearnTechnique 学习功法
// 验证：1) 功法是否存在 2) 玩家境界是否满足要求
func (s *TechniqueService) LearnTechnique(player *model.Player, techniqueID int) *model.TechniqueLearnResult {
	gc := s.config.GetConfig()

	tech, ok := gc.GetTechnique(techniqueID)
	if !ok {
		return &model.TechniqueLearnResult{
			Success: false,
			Message: "功法不存在",
		}
	}

	// 检查境界要求
	if player.RealmID < tech.RequiredRealmID ||
		(player.RealmID == tech.RequiredRealmID && player.RealmLevel < tech.RequiredRealmLevel) {
		return &model.TechniqueLearnResult{
			Success: false,
			Message: fmt.Sprintf("境界不足，需要%d级及以上", tech.RequiredRealmID),
		}
	}

	// 检查玩家当前是否已有功法（只能装备一种）
	if player.TechniqueID > 0 && player.TechniqueID != techniqueID {
		// 允许切换功法，但提示旧功法失效
		_ = player.TechniqueID
	}

	// 学习/切换功法
	player.TechniqueID = techniqueID
	player.TechniqueLevel = 1

	// 通知 Player 服务更新装备的功法信息（异步，不阻塞）
	go s.notifyTechniqueChange(player.ID, techniqueID, tech.Name)

	return &model.TechniqueLearnResult{
		Success:   true,
		Message:   fmt.Sprintf("成功学习功法 %s", tech.Name),
		Technique: tech,
	}
}

// CalculateEfficiency 计算玩家修炼效率
// 效率 = 基础速度 * 功法速度加成 * (1 + 灵根匹配加成) * (1 + 丹药加成)
func (s *TechniqueService) CalculateEfficiency(player *model.Player) *model.CultivationEfficiency {
	gc := s.config.GetConfig()

	efficiency := &model.CultivationEfficiency{
		BaseSpeed:    1.0,
		TechniqueSpeed: 1.0,
		SpiritRootBonus: 0.0,
		PillBonus:    0.0,
	}

	// 基础修炼速度（境界越高基础越快）
	realm, ok := gc.GetRealm(player.RealmID)
	if ok {
		efficiency.BaseSpeed = 1.0 + float64(realm.ID-1)*0.2 // 每提升一个大境界，基础速度+20%
	}

	// 功法加成
	if player.TechniqueID > 0 {
		if tech, ok := gc.GetTechnique(player.TechniqueID); ok {
			efficiency.TechniqueSpeed = tech.CultivationSpeed

			// 灵根与功法元素亲和判定
			spiritBonus := 0.0
			if rootVal, hasRoot := player.SpiritRoots[tech.Element]; hasRoot {
				spiritBonus = rootVal * 0.5 // 灵根匹配最多50%加成
			}
			// 检查其他元素的亲和/排斥
			for element, affinity := range tech.ElementAffinity {
				if rootVal, hasRoot := player.SpiritRoots[element]; hasRoot {
					spiritBonus += rootVal * affinity * 0.3
				}
			}
			if spiritBonus < -0.5 {
				spiritBonus = -0.5 // 最低减益限制
			}
			efficiency.SpiritRootBonus = spiritBonus
		}
	}

	// 丹药加成
	for _, bonus := range player.PillBonuses {
		efficiency.PillBonus += bonus
	}
	if efficiency.PillBonus > 1.0 {
		efficiency.PillBonus = 1.0 // 丹药加成上限100%
	}

	// 最终速度
	efficiency.FinalSpeed = efficiency.BaseSpeed * efficiency.TechniqueSpeed * (1 + efficiency.SpiritRootBonus) * (1 + efficiency.PillBonus)

	// 修为/秒 = 基础值 * 最终速度
	basePerMin := float64(1)
	if realm, ok := s.config.GetConfig().GetRealm(player.RealmID); ok && realm.BaseSpeed > 0 {
		basePerMin = realm.BaseSpeed
	}
	spiritMult := float64(1.0)
	if player.SpiritDensity > 0 {
		spiritMult = player.SpiritDensity
	}
	// 修正: 用浮点计算每分钟修为，不截短为整数。exp_per_second 只用于前端展示
	efficiency.ExpPerMinute = basePerMin * efficiency.FinalSpeed * spiritMult
	efficiency.ExpPerSecond = int64(efficiency.ExpPerMinute / 60.0)
	if efficiency.ExpPerMinute < 0.01 {
		efficiency.ExpPerMinute = 0.01
	}

	return efficiency
}

// GetAvailableTechniques 获取玩家当前可学习的功法列表
func (s *TechniqueService) GetAvailableTechniques(player *model.Player) []*model.Technique {
	gc := s.config.GetConfig()
	var available []*model.Technique

	for i := range gc.Techniques {
		tech := &gc.Techniques[i]
		if player.RealmID > tech.RequiredRealmID ||
			(player.RealmID == tech.RequiredRealmID && player.RealmLevel >= tech.RequiredRealmLevel) {
			available = append(available, tech)
		}
	}

	return available
}

// notifyTechniqueChange 通知 Player 服务玩家装备的功法发生变化
func (s *TechniqueService) notifyTechniqueChange(playerID uint64, techniqueID int, techniqueName string) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"technique_id":   techniqueID,
		"technique_name": techniqueName,
	})

	url := fmt.Sprintf("%s/api/v1/player/%d/technique", s.playerServiceAddr, playerID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}
