package mysql

import (
	"database/sql"
	"fmt"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// PetRepo 灵兽数据访问
type PetRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewPetRepo 创建 PetRepo
func NewPetRepo(db *sql.DB, log *zap.Logger) *PetRepo {
	return &PetRepo{db: db, log: log}
}

// Create 插入灵兽记录
func (r *PetRepo) Create(p *model.Pet) error {
	query := `INSERT INTO player_pets (player_id, name, species, star, level, exp,
		hp, atk, def, skill_id, skill_name, skill_desc, skill_type, skill_value, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query,
		p.PlayerID, p.Name, p.Species, p.Star, p.Level, p.Exp,
		p.HP, p.Atk, p.Def,
		p.Skill.ID, p.Skill.Name, p.Skill.Desc, p.Skill.Type, p.Skill.Value,
		p.Active,
	)
	if err != nil {
		r.log.Error("创建灵兽失败", zap.Error(err))
		return fmt.Errorf("创建灵兽失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取灵兽自增ID失败: %w", err)
	}
	p.ID = id
	return nil
}

// GetByID 根据ID查询灵兽
func (r *PetRepo) GetByID(id int64) (*model.Pet, error) {
	query := `SELECT id, player_id, name, species, star, level, exp,
		hp, atk, def, skill_id, skill_name, skill_desc, skill_type, skill_value, active
		FROM player_pets WHERE id = ?`

	p := &model.Pet{}
	var skillID int
	var skillName, skillDesc, skillType string
	var skillValue int64
	err := r.db.QueryRow(query, id).Scan(
		&p.ID, &p.PlayerID, &p.Name, &p.Species, &p.Star, &p.Level, &p.Exp,
		&p.HP, &p.Atk, &p.Def,
		&skillID, &skillName, &skillDesc, &skillType, &skillValue,
		&p.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询灵兽失败", zap.Error(err))
		return nil, fmt.Errorf("查询灵兽失败: %w", err)
	}
	p.Skill = model.PetSkill{
		ID:    skillID,
		Name:  skillName,
		Desc:  skillDesc,
		Type:  skillType,
		Value: skillValue,
	}
	return p, nil
}

// ListByPlayerID 查询玩家所有灵兽
func (r *PetRepo) ListByPlayerID(playerID int64) ([]*model.Pet, error) {
	query := `SELECT id, player_id, name, species, star, level, exp,
		hp, atk, def, skill_id, skill_name, skill_desc, skill_type, skill_value, active
		FROM player_pets WHERE player_id = ? ORDER BY active DESC, star DESC, level DESC`

	rows, err := r.db.Query(query, playerID)
	if err != nil {
		r.log.Error("查询灵兽列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询灵兽列表失败: %w", err)
	}
	defer rows.Close()

	var pets []*model.Pet
	for rows.Next() {
		p := &model.Pet{}
		var skillID int
		var skillName, skillDesc, skillType string
		var skillValue int64
		if err := rows.Scan(
			&p.ID, &p.PlayerID, &p.Name, &p.Species, &p.Star, &p.Level, &p.Exp,
			&p.HP, &p.Atk, &p.Def,
			&skillID, &skillName, &skillDesc, &skillType, &skillValue,
			&p.Active,
		); err != nil {
			r.log.Error("扫描灵兽行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描灵兽行失败: %w", err)
		}
		p.Skill = model.PetSkill{
			ID:    skillID,
			Name:  skillName,
			Desc:  skillDesc,
			Type:  skillType,
			Value: skillValue,
		}
		pets = append(pets, p)
	}
	if pets == nil {
		pets = []*model.Pet{}
	}
	return pets, nil
}

// GetActiveByPlayerID 查询玩家出战的灵兽
func (r *PetRepo) GetActiveByPlayerID(playerID int64) (*model.Pet, error) {
	query := `SELECT id, player_id, name, species, star, level, exp,
		hp, atk, def, skill_id, skill_name, skill_desc, skill_type, skill_value, active
		FROM player_pets WHERE player_id = ? AND active = TRUE LIMIT 1`

	p := &model.Pet{}
	var skillID int
	var skillName, skillDesc, skillType string
	var skillValue int64
	err := r.db.QueryRow(query, playerID).Scan(
		&p.ID, &p.PlayerID, &p.Name, &p.Species, &p.Star, &p.Level, &p.Exp,
		&p.HP, &p.Atk, &p.Def,
		&skillID, &skillName, &skillDesc, &skillType, &skillValue,
		&p.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询出战灵兽失败", zap.Error(err))
		return nil, fmt.Errorf("查询出战灵兽失败: %w", err)
	}
	p.Skill = model.PetSkill{
		ID:    skillID,
		Name:  skillName,
		Desc:  skillDesc,
		Type:  skillType,
		Value: skillValue,
	}
	return p, nil
}

// Update 更新灵兽
func (r *PetRepo) Update(p *model.Pet) error {
	query := `UPDATE player_pets SET name=?, star=?, level=?, exp=?,
		hp=?, atk=?, def=?, skill_id=?, skill_name=?, skill_desc=?, skill_type=?, skill_value=?, active=?
		WHERE id=?`

	_, err := r.db.Exec(query,
		p.Name, p.Star, p.Level, p.Exp,
		p.HP, p.Atk, p.Def,
		p.Skill.ID, p.Skill.Name, p.Skill.Desc, p.Skill.Type, p.Skill.Value,
		p.Active, p.ID,
	)
	if err != nil {
		r.log.Error("更新灵兽失败", zap.Error(err))
		return fmt.Errorf("更新灵兽失败: %w", err)
	}
	return nil
}

// DeactivateAll 将玩家所有灵兽设为非出战
func (r *PetRepo) DeactivateAll(playerID int64) error {
	_, err := r.db.Exec("UPDATE player_pets SET active = FALSE WHERE player_id = ?", playerID)
	if err != nil {
		r.log.Error("取消所有灵兽出战状态失败", zap.Error(err))
		return fmt.Errorf("取消出战状态失败: %w", err)
	}
	return nil
}

// Delete 删除灵兽
func (r *PetRepo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM player_pets WHERE id = ?", id)
	if err != nil {
		r.log.Error("删除灵兽失败", zap.Error(err))
		return fmt.Errorf("删除灵兽失败: %w", err)
	}
	return nil
}
