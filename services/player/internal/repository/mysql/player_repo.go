package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// PlayerRepo 玩家数据访问
type PlayerRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewPlayerRepo 创建 PlayerRepo
func NewPlayerRepo(db *sql.DB, log *zap.Logger) *PlayerRepo {
	return &PlayerRepo{db: db, log: log}
}

// Create 插入玩家
func (r *PlayerRepo) Create(p *model.Player) error {
	query := `INSERT INTO players (uid, nickname, gender, profession, profession_level, profession_exp, realm_id, realm_level, realm_stage, spirit_root, root_quality,
		hp, max_hp, mp, max_mp, attack, defense, speed, crit_rate, crit_dmg, dodge, hit, cult_bonus, break_bonus, mp_regen, lifespan, comprehension, luck, spirit_sense, last_luck_date, money, immortal_jade, exp, max_spirit, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	result, err := r.db.Exec(query,
		p.UserID, p.Name, p.Gender, p.Profession, p.ProfessionLevel, p.ProfessionExp, p.Realm, p.Level, p.RealmStage, p.SpiritRoot, p.RootQuality,
		p.HP, p.MaxHP, p.MP, p.MaxMP, p.Attack, p.Defense, p.Speed, p.CritRate, p.CritDmg, p.Dodge, p.Hit, p.CultBonus, p.BreakBonus, p.MPRegen, p.Lifespan, p.Comprehension, p.Luck, p.SpiritSense, p.LastLuckDate,
		p.Gold, p.Jade, p.SpiritPower, p.MaxSpirit,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		r.log.Error("创建玩家失败", zap.Error(err))
		return fmt.Errorf("创建玩家失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取自增ID失败: %w", err)
	}
	p.ID = id
	return nil
}

// GetByID 根据ID查询玩家
func (r *PlayerRepo) GetByID(id int64) (*model.Player, error) {
	query := `SELECT id, uid, nickname, gender, profession, profession_level, profession_exp, realm_id, realm_level, realm_stage, spirit_root, root_quality,
		hp, max_hp, mp, max_mp, attack, defense, speed, crit_rate, crit_dmg, dodge, hit, cult_bonus, break_bonus, mp_regen, lifespan, comprehension, luck, spirit_sense, last_luck_date, money, immortal_jade, exp, max_spirit,
		created_at, updated_at
		FROM players WHERE id = ?`

	p := &model.Player{}
	var exp int64
	err := r.db.QueryRow(query, id).Scan(
		&p.ID, &p.UserID, &p.Name, &p.Gender, &p.Profession, &p.ProfessionLevel, &p.ProfessionExp, &p.Realm, &p.Level, &p.RealmStage, &p.SpiritRoot, &p.RootQuality,
		&p.HP, &p.MaxHP, &p.MP, &p.MaxMP, &p.Attack, &p.Defense, &p.Speed, &p.CritRate, &p.CritDmg, &p.Dodge, &p.Hit, &p.CultBonus, &p.BreakBonus, &p.MPRegen, &p.Lifespan, &p.Comprehension, &p.Luck, &p.SpiritSense, &p.LastLuckDate,
		&p.Gold, &p.Jade, &exp, &p.MaxSpirit,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询玩家失败", zap.Error(err))
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	p.SpiritPower = exp
	return p, nil
}

// GetByUserID 根据用户ID查询玩家
func (r *PlayerRepo) GetByUserID(userID string) (*model.Player, error) {
	query := `SELECT id, uid, nickname, gender, profession, profession_level, profession_exp, realm_id, realm_level, realm_stage, spirit_root, root_quality,
		hp, mp, attack, defense, speed, money, immortal_jade, exp, max_spirit,
		created_at, updated_at
		FROM players WHERE uid = ?`

	p := &model.Player{}
	var exp int64
	err := r.db.QueryRow(query, userID).Scan(
		&p.ID, &p.UserID, &p.Name, &p.Gender, &p.Profession, &p.ProfessionLevel, &p.ProfessionExp, &p.Realm, &p.Level, &p.RealmStage, &p.SpiritRoot, &p.RootQuality,
		&p.HP, &p.MaxHP, &p.MP, &p.MaxMP, &p.Attack, &p.Defense, &p.Speed, &p.CritRate, &p.CritDmg, &p.Dodge, &p.Hit, &p.CultBonus, &p.BreakBonus, &p.MPRegen, &p.Lifespan, &p.Comprehension, &p.Luck, &p.SpiritSense, &p.LastLuckDate,
		&p.Gold, &p.Jade, &exp, &p.MaxSpirit,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("根据UserID查询玩家失败", zap.Error(err))
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	p.SpiritPower = exp
	return p, nil
}

// GetByName 根据昵称查询玩家（检查重名）
func (r *PlayerRepo) GetByName(nickname string) (*model.Player, error) {
	query := `SELECT id FROM players WHERE nickname = ?`
	p := &model.Player{}
	err := r.db.QueryRow(query, nickname).Scan(&p.ID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询重名失败: %w", err)
	}
	return p, nil
}

// Update 更新玩家
func (r *PlayerRepo) Update(p *model.Player) error {
	query := `UPDATE players SET gender=?, profession=?, profession_level=?, profession_exp=?, realm_id=?, realm_level=?, realm_stage=?, spirit_root=?, root_quality=?,
		hp=?, max_hp=?, mp=?, max_mp=?, attack=?, defense=?, speed=?, crit_rate=?, crit_dmg=?, dodge=?, hit=?, cult_bonus=?, break_bonus=?, mp_regen=?, lifespan=?, comprehension=?, luck=?, spirit_sense=?, last_luck_date=?,
		exp=?, max_spirit=?, money=?, immortal_jade=?,
		updated_at=?
		WHERE id=?`

	p.UpdatedAt = time.Now()

	_, err := r.db.Exec(query,
		p.Gender, p.Profession, p.ProfessionLevel, p.ProfessionExp, p.Realm, p.Level, p.RealmStage, p.SpiritRoot, p.RootQuality,
		p.HP, p.MaxHP, p.MP, p.MaxMP, p.Attack, p.Defense, p.Speed, p.CritRate, p.CritDmg, p.Dodge, p.Hit, p.CultBonus, p.BreakBonus, p.MPRegen, p.Lifespan, p.Comprehension, p.Luck, p.SpiritSense, p.LastLuckDate,
		p.SpiritPower, p.MaxSpirit, p.Gold, p.Jade,
		p.UpdatedAt, p.ID,
	)
	if err != nil {
		r.log.Error("更新玩家失败", zap.Error(err))
		return fmt.Errorf("更新玩家失败: %w", err)
	}
	return nil
}

// UpdateCurrency 仅更新货币字段
func (r *PlayerRepo) UpdateCurrency(playerID int64, gold, boundGold, jade int64) error {
	query := `UPDATE players SET money=?, bind_money=?, immortal_jade=?, updated_at=? WHERE id=?`
	_, err := r.db.Exec(query, gold, boundGold, jade, time.Now(), playerID)
	if err != nil {
		r.log.Error("更新货币失败", zap.Error(err))
		return fmt.Errorf("更新货币失败: %w", err)
	}
	return nil
}

// Delete 删除玩家
func (r *PlayerRepo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM players WHERE id = ?", id)
	if err != nil {
		r.log.Error("删除玩家失败", zap.Error(err))
		return fmt.Errorf("删除玩家失败: %w", err)
	}
	return nil
}
