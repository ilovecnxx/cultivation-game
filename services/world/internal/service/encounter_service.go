// Package service 提供世界服务的业务逻辑
package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"cultivation-game/services/world/internal/model"
)

// EncounterService 奇遇服务(条件触发引擎)
// 负责奇遇的触发条件匹配、概率修正、分支处理和结果生成
type EncounterService struct {
	mu         sync.RWMutex
	encounters []*model.Encounter
	cooldowns  map[string]time.Time // key: "userID:encounterID" -> 冷却到期时间
	exploreSvc *ExploreService
}

// NewEncounterService 创建奇遇服务
func NewEncounterService(encountersPath string, exploreSvc *ExploreService) (*EncounterService, error) {
	encounters, err := loadEncounters(encountersPath)
	if err != nil {
		return nil, fmt.Errorf("加载奇遇配置失败: %w", err)
	}
	return &EncounterService{
		encounters: encounters,
		cooldowns:  make(map[string]time.Time),
		exploreSvc: exploreSvc,
	}, nil
}

// loadEncounters 从 JSON 加载奇遇配置
func loadEncounters(path string) ([]*model.Encounter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var encounters []*model.Encounter
	if err := json.Unmarshal(data, &encounters); err != nil {
		return nil, err
	}
	return encounters, nil
}

// TriggerEncounter 尝试为玩家触发奇遇
// 返回触发的奇遇(nil表示未触发)
// 触发流程: 区域过滤 -> 等级过滤 -> 冷却检查 -> 条件匹配 -> 概率计算 -> 随机选择
func (s *EncounterService) TriggerEncounter(userID, regionID string, playerLevel int) *model.Encounter {
	// 过滤当前区域可触发的奇遇
	var candidates []*model.Encounter

	s.mu.RLock()
	for _, enc := range s.encounters {
		// 检查区域
		inRegion := false
		for _, rid := range enc.Regions {
			if rid == regionID {
				inRegion = true
				break
			}
		}
		if !inRegion {
			continue
		}

		// 检查等级
		if playerLevel < enc.MinLevel || playerLevel > enc.MaxLevel {
			continue
		}

		// 检查冷却
		if enc.CooldownSec > 0 {
			cooldownKey := userID + ":" + enc.ID
			if expireAt, ok := s.cooldowns[cooldownKey]; ok {
				if time.Now().Before(expireAt) {
					continue // 仍在冷却
				}
			}
		}

		// 计算概率(含福缘修正)
		finalProb := s.calculateFinalProbability(enc.Probability, playerLevel)

		// 主概率判定
		if rand.Float64() > finalProb {
			continue
		}

		// 检查条件
		if s.checkConditions(enc.Conditions, playerLevel) {
			candidates = append(candidates, enc)
		}
	}
	s.mu.RUnlock()

	if len(candidates) == 0 {
		return nil
	}

	// 随机选择一个奇遇
	chosen := candidates[rand.Intn(len(candidates))]

	// 记录冷却
	if chosen.CooldownSec > 0 {
		s.mu.Lock()
		cooldownKey := userID + ":" + chosen.ID
		s.cooldowns[cooldownKey] = time.Now().Add(time.Duration(chosen.CooldownSec) * time.Second)
		s.mu.Unlock()
	}

	return chosen
}

// calculateFinalProbability 计算最终触发概率(基础概率 + 修正因子)
// 修正因子模拟福缘/天机等隐藏属性对奇遇概率的影响
func (s *EncounterService) calculateFinalProbability(baseProb float64, playerLevel int) float64 {
	// 福缘修正: 等级越高福缘越深(模拟), 最高增加 15%
	fortuneBonus := float64(playerLevel) * 0.001
	if fortuneBonus > 0.15 {
		fortuneBonus = 0.15
	}

	finalProb := baseProb + fortuneBonus
	if finalProb > 1.0 {
		finalProb = 1.0
	}
	if finalProb < 0.01 {
		finalProb = 0.01
	}
	return finalProb
}

// checkConditions 检查奇遇触发条件
// 支持条件类型:
//   - level: 玩家等级判定(gt/gte/lt/lte/eq)
//   - cultivation: 修为值判定(用等级简化)
//   - probability: 额外概率判定(叠加福缘修正)
//   - item: 物品检查(需要背包服务, 此处简化)
//   - attribute: 属性检查(福缘/悟性等隐藏属性)
func (s *EncounterService) checkConditions(conditions []model.EncounterCondition, playerLevel int) bool {
	if len(conditions) == 0 {
		return true // 无条件默认触发
	}

	for _, cond := range conditions {
		switch cond.Type {
		case "level":
			levelVal := int(toFloat64(cond.Value))
			switch cond.Operator {
			case "gt":
				if playerLevel <= levelVal {
					return false
				}
			case "gte":
				if playerLevel < levelVal {
					return false
				}
			case "lt":
				if playerLevel >= levelVal {
					return false
				}
			case "lte":
				if playerLevel > levelVal {
					return false
				}
			case "eq":
				if playerLevel != levelVal {
					return false
				}
			default:
				return false
			}

		case "cultivation":
			// 修为检查(用等级简化模拟)
			cultVal := int(toFloat64(cond.Value))
			switch cond.Operator {
			case "gte":
				if playerLevel < cultVal {
					return false
				}
			case "lt":
				if playerLevel >= cultVal {
					return false
				}
			default:
				return false
			}

		case "probability":
			// 额外概率判定(与主概率叠加检查)
			// 这里作为最终概率的补充判定
			prob := toFloat64(cond.Value)
			if prob > 0 && rand.Float64() > prob {
				return false
			}

		case "item":
			// 物品检查(需要背包服务, 此处简化: 高等级玩家默认拥有更多物品)
			if cond.Operator == "has" && playerLevel < 20 {
				return false
			}

		case "attribute":
			// 隐藏属性检查(福缘/悟性, 用随机数模拟)
			attrValue := int(toFloat64(cond.Value))
			// 模拟福缘值: 玩家级别越高基础福缘越高, 加随机浮动
			simulatedFortune := playerLevel/5 + rand.Intn(5)
			switch cond.Operator {
			case "gte":
				if simulatedFortune < attrValue {
					return false
				}
			case "lt":
				if simulatedFortune >= attrValue {
					return false
				}
			default:
				return false
			}

		default:
			// 未知条件类型，默认通过
			continue
		}
	}

	return true
}

// toFloat64 安全转换 interface{} 为 float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	default:
		return 0
	}
}

// ExecuteChoice 执行奇遇选择分支
// 返回结果描述和可能的错误
func (s *EncounterService) ExecuteChoice(userID, encounterID string, choiceIndex int) (string, error) {
	s.mu.RLock()
	// 查找奇遇
	var enc *model.Encounter
	for _, e := range s.encounters {
		if e.ID == encounterID {
			enc = e
			break
		}
	}
	s.mu.RUnlock()

	if enc == nil {
		return "", fmt.Errorf("奇遇不存在")
	}

	// 自动结果(无需选择)
	if enc.AutoOutcome != nil {
		return s.executeOutcome(userID, enc.AutoOutcome)
	}

	// 检查选择索引
	if choiceIndex < 0 || choiceIndex >= len(enc.Choices) {
		return "", fmt.Errorf("无效的选择")
	}

	choice := enc.Choices[choiceIndex]
	if choice.Outcome == nil {
		return "你什么都没做。", nil
	}

	return s.executeOutcome(userID, choice.Outcome)
}

// executeOutcome 执行奇遇结果
// 支持结果类型: exp / item / damage / spirit_stone / buff / teleport / none
func (s *EncounterService) executeOutcome(userID string, outcome *model.EncounterOutcome) (string, error) {
	if outcome == nil {
		return "", nil
	}

	switch outcome.Type {
	case "exp":
		// 获得经验(实际应调用用户服务, 这里返回描述让前端处理)
		return outcome.Description, nil

	case "item":
		// 获得物品(实际应调用背包服务, 这里返回描述让前端处理)
		return outcome.Description, nil

	case "damage":
		// 受到伤害
		if outcome.Amount < 0 {
			return outcome.Description, nil
		}
		return outcome.Description, nil

	case "spirit_stone":
		// 获得灵石
		return outcome.Description, nil

	case "buff":
		// 添加增益状态
		return outcome.Description, nil

	case "teleport":
		// 传送到指定区域
		if outcome.TargetID != "" && s.exploreSvc != nil {
			result, err := s.exploreSvc.MoveTo(userID, outcome.TargetID)
			if err != nil {
				return "", fmt.Errorf("传送失败: %w", err)
			}
			return result.Message, nil
		}
		return outcome.Description, nil

	case "none":
		// 无实际效果
		return outcome.Description, nil

	default:
		return outcome.Description, nil
	}
}

// GetEncountersByRegion 获取指定区域的奇遇列表
func (s *EncounterService) GetEncountersByRegion(regionID string) []*model.Encounter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*model.Encounter
	for _, enc := range s.encounters {
		for _, rid := range enc.Regions {
			if rid == regionID {
				result = append(result, enc)
				break
			}
		}
	}
	return result
}

// GetAllEncounters 获取所有奇遇
func (s *EncounterService) GetAllEncounters() []*model.Encounter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.Encounter, len(s.encounters))
	copy(result, s.encounters)
	return result
}
