package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"

	"cultivation-game/services/player/internal/model"
	"cultivation-game/services/player/internal/repository/mysql"

	"go.uber.org/zap"
)

// 灵兽相关物品ID（对应 items.json 中的配置）
const (
	ItemPetTrap    = 100 // 灵兽圈：捕捉灵兽的消耗品
	ItemPetExpPill = 101 // 灵兽经验丹：增加灵兽经验
)

// PetSpeciesData 加载的灵兽物种列表（全局缓存）
var PetSpeciesData []model.PetSpecies

func init() {
	loadPetSpecies()
}

// loadPetSpecies 从 pets.json 加载灵兽物种配置
func loadPetSpecies() {
	data, err := os.ReadFile("internal/data/pets.json")
	if err != nil {
		// 开发环境可能路径不同，尝试相对路径
		data, err = os.ReadFile("../player/internal/data/pets.json")
		if err != nil {
			// 运行时可能通过工作目录调整，静默失败让后续调用报错
			return
		}
	}
	var wrapper struct {
		Species []model.PetSpecies `json:"species"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return
	}
	PetSpeciesData = wrapper.Species
}

// PetService 灵兽业务逻辑
type PetService struct {
	petRepo     *mysql.PetRepo
	playerRepo  *mysql.PlayerRepo
	invService  *InventoryService
	log         *zap.Logger
}

// NewPetService 创建 PetService
func NewPetService(petRepo *mysql.PetRepo, playerRepo *mysql.PlayerRepo, invService *InventoryService, log *zap.Logger) *PetService {
	return &PetService{
		petRepo:    petRepo,
		playerRepo: playerRepo,
		invService: invService,
		log:        log,
	}
}

// -------- 内部辅助 --------

// findSpecies 按ID查找物种配置
func (s *PetService) findSpecies(id string) *model.PetSpecies {
	for i := range PetSpeciesData {
		if PetSpeciesData[i].ID == id {
			return &PetSpeciesData[i]
		}
	}
	return nil
}

// speciesByStar 按星级筛选物种
func (s *PetService) speciesByStar(star int) []model.PetSpecies {
	var result []model.PetSpecies
	for _, sp := range PetSpeciesData {
		if sp.Star == star {
			result = append(result, sp)
		}
	}
	return result
}

// checkRealmBase 检查玩家是否达到筑基期（RealmBase=3）
func (s *PetService) checkRealmBase(playerID int64) error {
	player, err := s.playerRepo.GetByID(playerID)
	if err != nil {
		return fmt.Errorf("查询玩家失败: %w", err)
	}
	if player == nil {
		return fmt.Errorf("玩家不存在")
	}
	if player.Realm < model.RealmBase {
		return fmt.Errorf("筑基期以上才能使用灵兽功能，当前境界 %s", model.RealmNames[player.Realm])
	}
	return nil
}

// pickRandomSpecies 根据星级概率随机选取一个物种
// 先按概率选中星级，再在该星级中随机选一个物种
func (s *PetService) pickRandomSpecies() (*model.PetSpecies, error) {
	if len(PetSpeciesData) == 0 {
		return nil, fmt.Errorf("灵兽物种数据未加载")
	}

	// 按 EncounterStarWeights 概率选择星级
	roll := rand.Float64()
	var cumulative float64
	chosenStar := 1
	// 按星级排序确保确定性
	var stars []int
	for k := range model.EncounterStarWeights {
		stars = append(stars, k)
	}
	sort.Ints(stars)
	for _, star := range stars {
		cumulative += model.EncounterStarWeights[star]
		if roll < cumulative {
			chosenStar = star
			break
		}
	}

	// 在该星级中随机选一个物种
	candidates := s.speciesByStar(chosenStar)
	if len(candidates) == 0 {
		// 容错：降级到任意物种
		idx := rand.Intn(len(PetSpeciesData))
		return &PetSpeciesData[idx], nil
	}
	idx := rand.Intn(len(candidates))
	return &candidates[idx], nil
}

// -------- 业务方法 --------

// TryEncounter 游历中尝试遭遇野生灵兽
// 返回值：encountered=true 表示遇到，species 为遇到的物种信息
func (s *PetService) TryEncounter(ctx context.Context, playerID int64) (encountered bool, species *model.PetSpecies, err error) {
	if err := s.checkRealmBase(playerID); err != nil {
		return false, nil, err
	}

	// 10% 概率遇到
	if rand.Float64() >= 0.10 {
		return false, nil, nil
	}

	sp, err := s.pickRandomSpecies()
	if err != nil {
		return false, nil, err
	}

	s.log.Info("玩家遭遇野生灵兽",
		zap.Int64("player_id", playerID),
		zap.String("species", sp.Name),
		zap.Int("star", sp.Star),
	)
	return true, sp, nil
}

// Capture 捕捉灵兽
// 消耗灵兽圈，根据星级概率判定是否成功
func (s *PetService) Capture(ctx context.Context, playerID int64, speciesID string) (*model.Pet, error) {
	if err := s.checkRealmBase(playerID); err != nil {
		return nil, err
	}

	// 查找物种配置
	species := s.findSpecies(speciesID)
	if species == nil {
		return nil, fmt.Errorf("未知的灵兽物种: %s", speciesID)
	}

	// 检查玩家是否有灵兽圈
	// 从背包查找 ItemPetTrap（灵兽圈）
	inventory, err := s.invService.GetInventory(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询背包失败: %w", err)
	}
	var invTrapItemID int64
	found := false
	for _, item := range inventory {
		if item.ItemID == ItemPetTrap && item.Quantity > 0 {
			invTrapItemID = item.ID
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("需要「灵兽圈」才能捕捉灵兽")
	}

	// 根据星级计算捕捉成功率
	rate, ok := model.CaptureRateByStar[species.Star]
	if !ok {
		rate = 0.30
	}
	success := rand.Float64() < rate

	// 消耗一个灵兽圈
	if err := s.invService.RemoveItem(ctx, playerID, invTrapItemID, 1); err != nil {
		return nil, fmt.Errorf("扣除灵兽圈失败: %w", err)
	}

	if !success {
		s.log.Info("捕捉灵兽失败",
			zap.Int64("player_id", playerID),
			zap.String("species", species.Name),
		)
		return nil, fmt.Errorf("捕捉失败，灵兽逃脱了")
	}

	// 获取技能定义
	skillDef, hasSkill := model.PetSkillDefs[species.SkillID]
	if !hasSkill {
		skillDef = model.PetSkill{ID: species.SkillID, Name: "未知技能", Type: "attack", Value: 10}
	}

	// 计算初始属性（1级）
	hp, atk, def := model.CalcPetStats(species, species.Star, 1)

	pet := &model.Pet{
		PlayerID: playerID,
		Name:     species.Name,
		Species:  species.ID,
		Star:     species.Star,
		Level:    1,
		Exp:      0,
		HP:       hp,
		Atk:      atk,
		Def:      def,
		Skill:    skillDef,
		Active:   false,
	}

	if err := s.petRepo.Create(pet); err != nil {
		return nil, err
	}

	s.log.Info("捕捉灵兽成功",
		zap.Int64("player_id", playerID),
		zap.String("species", species.Name),
		zap.Int("star", species.Star),
	)
	return pet, nil
}

// Rename 重命名灵兽
func (s *PetService) Rename(ctx context.Context, playerID int64, petID int64, newName string) (*model.Pet, error) {
	pet, err := s.petRepo.GetByID(petID)
	if err != nil {
		return nil, fmt.Errorf("查询灵兽失败: %w", err)
	}
	if pet == nil {
		return nil, fmt.Errorf("灵兽不存在")
	}
	if pet.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此灵兽")
	}

	pet.Name = newName
	if err := s.petRepo.Update(pet); err != nil {
		return nil, err
	}
	return pet, nil
}

// Feed 喂食灵兽经验丹，增加经验
// 消耗指定数量的灵兽经验丹，按星级和等级计算实际获得经验
func (s *PetService) Feed(ctx context.Context, playerID int64, petID int64, quantity int32) (*model.Pet, []int, error) {
	if quantity <= 0 {
		return nil, nil, fmt.Errorf("数量必须大于0")
	}

	pet, err := s.petRepo.GetByID(petID)
	if err != nil {
		return nil, nil, fmt.Errorf("查询灵兽失败: %w", err)
	}
	if pet == nil {
		return nil, nil, fmt.Errorf("灵兽不存在")
	}
	if pet.PlayerID != playerID {
		return nil, nil, fmt.Errorf("无权操作此灵兽")
	}
	if pet.Level >= 100 {
		return nil, nil, fmt.Errorf("灵兽已达最高等级(100级)")
	}

	// 检查背包中灵兽经验丹数量
	inventory, err := s.invService.GetInventory(ctx, playerID)
	if err != nil {
		return nil, nil, fmt.Errorf("查询背包失败: %w", err)
	}
	var invItemID int64
	haveQty := int32(0)
	for _, item := range inventory {
		if item.ItemID == ItemPetExpPill && item.Quantity > 0 {
			invItemID = item.ID
			haveQty = item.Quantity
			break
		}
	}
	if haveQty < quantity {
		return nil, nil, fmt.Errorf("灵兽经验丹不足，拥有 %d，需要 %d", haveQty, quantity)
	}

	// 每颗经验丹提供固定经验值 * 星级系数
	expPerPill := int64(50 * pet.Star)
	totalExp := expPerPill * int64(quantity)

	// 消耗经验丹
	if err := s.invService.RemoveItem(ctx, playerID, invItemID, quantity); err != nil {
		return nil, nil, fmt.Errorf("扣除灵兽经验丹失败: %w", err)
	}

	// 累加经验并升级
	pet.Exp += totalExp
	leveledUp := s.applyLevelUp(pet)

	// 持久化
	if err := s.petRepo.Update(pet); err != nil {
		return nil, nil, err
	}

	s.log.Info("喂食灵兽经验丹",
		zap.Int64("player_id", playerID),
		zap.Int64("pet_id", petID),
		zap.Int32("quantity", quantity),
		zap.Int("levels_gained", len(leveledUp)),
	)
	return pet, leveledUp, nil
}

// applyLevelUp 检查并执行升级，返回升级到的等级列表
func (s *PetService) applyLevelUp(pet *model.Pet) []int {
	var newLevels []int
	for pet.Level < 100 {
		needExp := model.PetLevelUpExp(pet.Star, pet.Level)
		if pet.Exp < needExp {
			break
		}
		pet.Exp -= needExp
		pet.Level++
		newLevels = append(newLevels, pet.Level)
	}
	if len(newLevels) > 0 {
		// 重新计算属性
		species := s.findSpecies(pet.Species)
		if species != nil {
			pet.HP, pet.Atk, pet.Def = model.CalcPetStats(species, pet.Star, pet.Level)
		}
	}
	return newLevels
}

// GetLevelUpInfo 查询升级进度
func (s *PetService) GetLevelUpInfo(ctx context.Context, playerID int64, petID int64) (currentExp, needExp int64, level int, err error) {
	pet, err := s.petRepo.GetByID(petID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("查询灵兽失败: %w", err)
	}
	if pet == nil {
		return 0, 0, 0, fmt.Errorf("灵兽不存在")
	}
	if pet.PlayerID != playerID {
		return 0, 0, 0, fmt.Errorf("无权操作此灵兽")
	}

	if pet.Level >= 100 {
		return pet.Exp, 0, pet.Level, nil // 满级无需求
	}
	return pet.Exp, model.PetLevelUpExp(pet.Star, pet.Level), pet.Level, nil
}

// Evolve 灵兽进化（提升星级）
// 条件：满级(100级) + 消耗特殊材料
// 进化后：星级+1，等级重置为1，属性重新计算
func (s *PetService) Evolve(ctx context.Context, playerID int64, petID int64) (*model.Pet, error) {
	pet, err := s.petRepo.GetByID(petID)
	if err != nil {
		return nil, fmt.Errorf("查询灵兽失败: %w", err)
	}
	if pet == nil {
		return nil, fmt.Errorf("灵兽不存在")
	}
	if pet.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此灵兽")
	}

	// 检查等级
	if pet.Level < 100 {
		return nil, fmt.Errorf("灵兽需要达到100级才能进化，当前 %d 级", pet.Level)
	}
	// 检查星级上限
	if pet.Star >= 5 {
		return nil, fmt.Errorf("灵兽已达最高星级(5星)，无法继续进化")
	}

	// 计算进化所需材料
	materialID, materialName := s.evolveMaterial(pet.Star)
	inventory, err := s.invService.GetInventory(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询背包失败: %w", err)
	}
	var invMatID int64
	hasMaterial := false
	for _, item := range inventory {
		if item.ItemID == materialID && item.Quantity > 0 {
			invMatID = item.ID
			hasMaterial = true
			break
		}
	}
	if !hasMaterial {
		return nil, fmt.Errorf("进化需要 %s，当前背包中无此材料", materialName)
	}

	// 消耗材料
	if err := s.invService.RemoveItem(ctx, playerID, invMatID, 1); err != nil {
		return nil, fmt.Errorf("扣除进化材料失败: %w", err)
	}

	// 执行进化
	oldStar := pet.Star
	pet.Star++
	pet.Level = 1
	pet.Exp = 0

	// 根据新星级升级技能
	if def, ok := model.PetSkillDefs[pet.Skill.ID]; ok {
		// 技能随星级提升增强
		def.Value = def.Value * int64(pet.Star) / int64(oldStar)
		pet.Skill = def
	}

	// 重新计算属性
	species := s.findSpecies(pet.Species)
	if species != nil {
		pet.HP, pet.Atk, pet.Def = model.CalcPetStats(species, pet.Star, pet.Level)
	}

	if err := s.petRepo.Update(pet); err != nil {
		return nil, err
	}

	s.log.Info("灵兽进化成功",
		zap.Int64("player_id", playerID),
		zap.Int64("pet_id", petID),
		zap.Int("old_star", oldStar),
		zap.Int("new_star", pet.Star),
	)
	return pet, nil
}

// evolveMaterial 根据当前星级返回进化所需材料ID和名称
func (s *PetService) evolveMaterial(currentStar int) (int64, string) {
	switch currentStar {
	case 1:
		return 102, "灵蕴石"
	case 2:
		return 103, "玄天晶"
	case 3:
		return 104, "紫府玉"
	case 4:
		return 105, "混沌精魄"
	default:
		return 102, "灵蕴石"
	}
}

// SetActive 设置出战灵兽（一个玩家只能出战一只）
func (s *PetService) SetActive(ctx context.Context, playerID int64, petID int64) (*model.Pet, error) {
	pet, err := s.petRepo.GetByID(petID)
	if err != nil {
		return nil, fmt.Errorf("查询灵兽失败: %w", err)
	}
	if pet == nil {
		return nil, fmt.Errorf("灵兽不存在")
	}
	if pet.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此灵兽")
	}
	if pet.Active {
		return pet, nil // 已是出战状态
	}

	// 先取消所有灵兽的出战状态
	if err := s.petRepo.DeactivateAll(playerID); err != nil {
		return nil, err
	}

	// 设置当前灵兽为出战
	pet.Active = true
	if err := s.petRepo.Update(pet); err != nil {
		return nil, err
	}

	s.log.Info("设置出战灵兽",
		zap.Int64("player_id", playerID),
		zap.Int64("pet_id", petID),
		zap.String("name", pet.Name),
	)
	return pet, nil
}

// UnsetActive 取消出战
func (s *PetService) UnsetActive(ctx context.Context, playerID int64, petID int64) (*model.Pet, error) {
	pet, err := s.petRepo.GetByID(petID)
	if err != nil {
		return nil, fmt.Errorf("查询灵兽失败: %w", err)
	}
	if pet == nil {
		return nil, fmt.Errorf("灵兽不存在")
	}
	if pet.PlayerID != playerID {
		return nil, fmt.Errorf("无权操作此灵兽")
	}

	pet.Active = false
	if err := s.petRepo.Update(pet); err != nil {
		return nil, err
	}
	return pet, nil
}

// ListPets 列出玩家所有灵兽
func (s *PetService) ListPets(ctx context.Context, playerID int64) ([]*model.Pet, error) {
	if err := s.checkRealmBase(playerID); err != nil {
		return nil, err
	}
	return s.petRepo.ListByPlayerID(playerID)
}

// GetActivePet 获取玩家当前出战的灵兽
func (s *PetService) GetActivePet(ctx context.Context, playerID int64) (*model.Pet, error) {
	return s.petRepo.GetActiveByPlayerID(playerID)
}

// GetPetByID 按ID获取单只灵兽
func (s *PetService) GetPetByID(ctx context.Context, playerID int64, petID int64) (*model.Pet, error) {
	pet, err := s.petRepo.GetByID(petID)
	if err != nil {
		return nil, err
	}
	if pet == nil {
		return nil, fmt.Errorf("灵兽不存在")
	}
	if pet.PlayerID != playerID {
		return nil, fmt.Errorf("无权查看此灵兽")
	}
	return pet, nil
}

// CalcBattleBonus 计算出战灵兽的战斗加成（供战斗系统调用）
// 返回 (额外攻击, 额外防御, 技能触发概率, 技能对象)
func CalcBattleBonus(pet *model.Pet) (atkBonus, defBonus int64, triggerRate float64, skill model.PetSkill) {
	if pet == nil || !pet.Active {
		return 0, 0, 0, model.PetSkill{}
	}
	// 灵兽提供其攻击/防御的 30% 作为主人加成
	atkBonus = pet.Atk * 30 / 100
	defBonus = pet.Def * 30 / 100
	// 技能触发基础概率 15% + 每级 0.1%
	triggerRate = 0.15 + float64(pet.Level)*0.001
	if triggerRate > 0.30 {
		triggerRate = 0.30
	}
	return atkBonus, defBonus, triggerRate, pet.Skill
}
