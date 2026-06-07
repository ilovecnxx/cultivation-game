package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// FormationRepo 阵法数据访问
type FormationRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewFormationRepo 创建 FormationRepo
func NewFormationRepo(db *sql.DB, log *zap.Logger) *FormationRepo {
	return &FormationRepo{db: db, log: log}
}

// LoadTemplates 从 JSON 文件加载阵法图谱模板
func (r *FormationRepo) LoadTemplates(path string) ([]*model.FormationTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取阵法数据文件失败: %w", err)
	}

	var wrapper struct {
		Formations []*model.FormationTemplate `json:"formations"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("解析阵法数据失败: %w", err)
	}

	r.log.Info("加载阵法图谱", zap.Int("count", len(wrapper.Formations)))
	return wrapper.Formations, nil
}

// Create 玩家习得阵法（插入记录）
func (r *FormationRepo) Create(f *model.Formation) error {
	effectsJSON, err := json.Marshal(f.Effects)
	if err != nil {
		return fmt.Errorf("序列化阵法效果失败: %w", err)
	}

	query := `INSERT INTO player_formations
		(player_id, tmpl_id, name, type, level, quality,
		 deployed, guardian, exp, effects_json, learned_at,
		 mastery_exp, mastery_level, guardian_pet_id, link_group)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	f.LearnedAt = now
	// Default mastery to 0
	if f.MasteryLevel == 0 && f.MasteryExp == 0 {
		// leave defaults
	}

	result, err := r.db.Exec(query,
		f.PlayerID, f.TmplID, f.Name, f.Type, f.Level, f.Quality,
		f.Deployed, f.Guardian, f.Exp, string(effectsJSON), f.LearnedAt,
		f.MasteryExp, f.MasteryLevel, f.GuardianPetID, f.LinkGroup,
	)
	if err != nil {
		r.log.Error("习得阵法失败", zap.Error(err))
		return fmt.Errorf("习得阵法失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取阵法自增ID失败: %w", err)
	}
	f.ID = id
	return nil
}

// scanFormation 从 row 扫描 Formation 并解析 effects_json
func (r *FormationRepo) scanFormation(scanner interface {
	Scan(dest ...any) error
}) (*model.Formation, error) {
	f := &model.Formation{}
	var effectsJSON string
	var guardianPetID sql.NullInt64

	err := scanner.Scan(
		&f.ID, &f.PlayerID, &f.TmplID, &f.Name, &f.Type,
		&f.Level, &f.Quality, &f.Deployed, &f.Guardian,
		&f.Exp, &effectsJSON, &f.LearnedAt,
		&f.MasteryExp, &f.MasteryLevel, &guardianPetID, &f.LinkGroup,
	)
	if err != nil {
		return nil, err
	}

	if guardianPetID.Valid {
		f.GuardianPetID = &guardianPetID.Int64
	}

	if effectsJSON != "" {
		if err := json.Unmarshal([]byte(effectsJSON), &f.Effects); err != nil {
			r.log.Error("解析阵法效果失败", zap.Error(err))
			return nil, fmt.Errorf("解析阵法效果失败: %w", err)
		}
	}
	return f, nil
}

const formationColumns = `id, player_id, tmpl_id, name, type,
	level, quality, deployed, guardian, exp, effects_json, learned_at,
	mastery_exp, mastery_level, guardian_pet_id, link_group`

// GetByID 根据ID查询阵法
func (r *FormationRepo) GetByID(id int64) (*model.Formation, error) {
	query := `SELECT ` + formationColumns + ` FROM player_formations WHERE id = ?`

	row := r.db.QueryRow(query, id)
	f, err := r.scanFormation(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询阵法失败", zap.Error(err))
		return nil, fmt.Errorf("查询阵法失败: %w", err)
	}
	return f, nil
}

// ListByPlayerID 查询玩家所有阵法
func (r *FormationRepo) ListByPlayerID(playerID int64) ([]*model.Formation, error) {
	query := `SELECT ` + formationColumns + ` FROM player_formations
		WHERE player_id = ? ORDER BY deployed DESC, link_group DESC, quality DESC, level DESC`

	rows, err := r.db.Query(query, playerID)
	if err != nil {
		r.log.Error("查询阵法列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询阵法列表失败: %w", err)
	}
	defer rows.Close()

	var formations []*model.Formation
	for rows.Next() {
		f, err := r.scanFormation(rows)
		if err != nil {
			r.log.Error("扫描阵法行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描阵法行失败: %w", err)
		}
		formations = append(formations, f)
	}
	if formations == nil {
		formations = []*model.Formation{}
	}
	return formations, nil
}

// GetDeployedByPlayerID 查询玩家已部署的阵法
func (r *FormationRepo) GetDeployedByPlayerID(playerID int64) ([]*model.Formation, error) {
	query := `SELECT ` + formationColumns + ` FROM player_formations
		WHERE player_id = ? AND deployed = TRUE ORDER BY link_group DESC, id`

	rows, err := r.db.Query(query, playerID)
	if err != nil {
		r.log.Error("查询已部署阵法失败", zap.Error(err))
		return nil, fmt.Errorf("查询已部署阵法失败: %w", err)
	}
	defer rows.Close()

	var formations []*model.Formation
	for rows.Next() {
		f, err := r.scanFormation(rows)
		if err != nil {
			r.log.Error("扫描已部署阵法行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描已部署阵法行失败: %w", err)
		}
		formations = append(formations, f)
	}
	return formations, nil
}

// GetGuardianByPlayerID 查询玩家设为护法的阵法
func (r *FormationRepo) GetGuardianByPlayerID(playerID int64) (*model.Formation, error) {
	query := `SELECT ` + formationColumns + ` FROM player_formations
		WHERE player_id = ? AND guardian = TRUE LIMIT 1`

	row := r.db.QueryRow(query, playerID)
	f, err := r.scanFormation(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询护法阵法失败", zap.Error(err))
		return nil, fmt.Errorf("查询护法阵法失败: %w", err)
	}
	return f, nil
}

// Update 更新阵法
func (r *FormationRepo) Update(f *model.Formation) error {
	effectsJSON, err := json.Marshal(f.Effects)
	if err != nil {
		return fmt.Errorf("序列化阵法效果失败: %w", err)
	}

	query := `UPDATE player_formations SET
		level=?, quality=?, deployed=?, guardian=?,
		exp=?, effects_json=?,
		mastery_exp=?, mastery_level=?, guardian_pet_id=?, link_group=?
		WHERE id=?`

	_, err = r.db.Exec(query,
		f.Level, f.Quality, f.Deployed, f.Guardian,
		f.Exp, string(effectsJSON),
		f.MasteryExp, f.MasteryLevel, f.GuardianPetID, f.LinkGroup,
		f.ID,
	)
	if err != nil {
		r.log.Error("更新阵法失败", zap.Error(err))
		return fmt.Errorf("更新阵法失败: %w", err)
	}
	return nil
}

// ClearDeployedByPlayerID 清除玩家所有阵法的部署状态
func (r *FormationRepo) ClearDeployedByPlayerID(playerID int64) error {
	_, err := r.db.Exec("UPDATE player_formations SET deployed = FALSE WHERE player_id = ?", playerID)
	if err != nil {
		r.log.Error("清除部署状态失败", zap.Error(err))
		return fmt.Errorf("清除部署状态失败: %w", err)
	}
	return nil
}

// ClearGuardianByPlayerID 清除玩家所有阵法的护法状态
func (r *FormationRepo) ClearGuardianByPlayerID(playerID int64) error {
	_, err := r.db.Exec("UPDATE player_formations SET guardian = FALSE WHERE player_id = ?", playerID)
	if err != nil {
		r.log.Error("清除护法状态失败", zap.Error(err))
		return fmt.Errorf("清除护法状态失败: %w", err)
	}
	return nil
}

// Delete 删除阵法
func (r *FormationRepo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM player_formations WHERE id = ?", id)
	if err != nil {
		r.log.Error("删除阵法失败", zap.Error(err))
		return fmt.Errorf("删除阵法失败: %w", err)
	}
	return nil
}

// ============================================================
// 熟练度相关
// ============================================================

// AddMasteryExp 增加阵法熟练度经验
func (r *FormationRepo) AddMasteryExp(id int64, exp int64) error {
	_, err := r.db.Exec(
		"UPDATE player_formations SET mastery_exp = mastery_exp + ? WHERE id = ?", exp, id,
	)
	if err != nil {
		r.log.Error("增加熟练度失败", zap.Error(err))
		return fmt.Errorf("增加熟练度失败: %w", err)
	}
	return nil
}

// ============================================================
// 守护灵兽相关
// ============================================================

// GetByGuardianPetID 根据守护灵兽ID查询阵法
func (r *FormationRepo) GetByGuardianPetID(petID int64) (*model.Formation, error) {
	query := `SELECT ` + formationColumns + ` FROM player_formations WHERE guardian_pet_id = ? LIMIT 1`
	row := r.db.QueryRow(query, petID)
	f, err := r.scanFormation(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询守护灵兽关联阵法失败", zap.Error(err))
		return nil, fmt.Errorf("查询守护灵兽关联阵法失败: %w", err)
	}
	return f, nil
}

// ClearGuardianPet 清除阵法守护灵兽
func (r *FormationRepo) ClearGuardianPet(formationID int64) error {
	_, err := r.db.Exec(
		"UPDATE player_formations SET guardian_pet_id = NULL WHERE id = ?", formationID,
	)
	if err != nil {
		r.log.Error("清除守护灵兽失败", zap.Error(err))
		return fmt.Errorf("清除守护灵兽失败: %w", err)
	}
	return nil
}

// ============================================================
// 联动组相关
// ============================================================

// GetDeployedByLinkGroup 按联动组查询
func (r *FormationRepo) GetDeployedByLinkGroup(playerID int64, group int) ([]*model.Formation, error) {
	query := `SELECT ` + formationColumns + ` FROM player_formations
		WHERE player_id = ? AND deployed = TRUE AND link_group = ? ORDER BY id`

	rows, err := r.db.Query(query, playerID, group)
	if err != nil {
		r.log.Error("查询联动组阵法失败", zap.Error(err))
		return nil, fmt.Errorf("查询联动组阵法失败: %w", err)
	}
	defer rows.Close()

	var formations []*model.Formation
	for rows.Next() {
		f, err := r.scanFormation(rows)
		if err != nil {
			return nil, err
		}
		formations = append(formations, f)
	}
	return formations, nil
}

// ClearLinkGroup 清除某玩家的所有联动组
func (r *FormationRepo) ClearLinkGroup(playerID int64) error {
	_, err := r.db.Exec(
		"UPDATE player_formations SET link_group = 0 WHERE player_id = ? AND deployed = TRUE", playerID,
	)
	if err != nil {
		r.log.Error("清除联动组失败", zap.Error(err))
		return fmt.Errorf("清除联动组失败: %w", err)
	}
	return nil
}

// ============================================================
// GuardianTask 操作
// ============================================================

// CreateGuardianTask 创建护法记录
func (r *FormationRepo) CreateGuardianTask(t *model.GuardianTask) error {
	query := `INSERT INTO guardian_tasks (guardian_id, beneficiary_id, formation_id, bonus_rate, success, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	now := time.Now()
	t.CreatedAt = now

	result, err := r.db.Exec(query,
		t.GuardianID, t.BeneficiaryID, t.FormationID, t.BonusRate, t.Success, t.CreatedAt,
	)
	if err != nil {
		r.log.Error("创建护法记录失败", zap.Error(err))
		return fmt.Errorf("创建护法记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取护法记录自增ID失败: %w", err)
	}
	t.ID = id
	return nil
}

// ListGuardianTasksByPlayer 查询玩家的护法记录（作为护法方）
func (r *FormationRepo) ListGuardianTasksByPlayer(guardianID int64, limit int) ([]*model.GuardianTask, error) {
	query := `SELECT id, guardian_id, beneficiary_id, formation_id, bonus_rate, success, created_at
		FROM guardian_tasks WHERE guardian_id = ? ORDER BY created_at DESC LIMIT ?`

	rows, err := r.db.Query(query, guardianID, limit)
	if err != nil {
		r.log.Error("查询护法记录失败", zap.Error(err))
		return nil, fmt.Errorf("查询护法记录失败: %w", err)
	}
	defer rows.Close()

	var tasks []*model.GuardianTask
	for rows.Next() {
		t := &model.GuardianTask{}
		if err := rows.Scan(&t.ID, &t.GuardianID, &t.BeneficiaryID, &t.FormationID,
			&t.BonusRate, &t.Success, &t.CreatedAt); err != nil {
			r.log.Error("扫描护法记录行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描护法记录行失败: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
