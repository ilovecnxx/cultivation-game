package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// ArtifactRepo 法宝数据访问
type ArtifactRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewArtifactRepo 创建 ArtifactRepo
func NewArtifactRepo(db *sql.DB, log *zap.Logger) *ArtifactRepo {
	return &ArtifactRepo{db: db, log: log}
}

// Create 插入法宝记录
func (r *ArtifactRepo) Create(a *model.Artifact) error {
	query := `INSERT INTO player_artifacts (player_id, name, type, quality, level, exp,
		attack_bonus, defense_bonus, hp_bonus, mp_bonus, speed_bonus, dodge_bonus,
		skill_id, awaken_skills, potential, spirit_id, power_bonus, bound_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	a.BoundAt = now
	a.AwakenSkills = []int{} // default empty

	awakenJSON := "[]"
	if len(a.AwakenSkills) > 0 {
		b, _ := json.Marshal(a.AwakenSkills)
		awakenJSON = string(b)
	}

	result, err := r.db.Exec(query,
		a.PlayerID, a.Name, a.Type, a.Quality, a.Level, a.Exp,
		a.AttackBonus, a.DefenseBonus, a.HPBonus, a.MpBonus, a.SpeedBonus, a.DodgeBonus,
		a.SkillID, awakenJSON, a.Potential, a.SpiritID, a.PowerBonus, a.BoundAt,
	)
	if err != nil {
		r.log.Error("创建法宝失败", zap.Error(err))
		return fmt.Errorf("创建法宝失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取法宝自增ID失败: %w", err)
	}
	a.ID = id
	return nil
}

// GetByPlayerID 根据玩家ID查询法宝
func (r *ArtifactRepo) GetByPlayerID(playerID int64) (*model.Artifact, error) {
	query := `SELECT id, player_id, name, type, quality, level, exp,
		attack_bonus, defense_bonus, hp_bonus, mp_bonus, speed_bonus, dodge_bonus,
		skill_id, COALESCE(awaken_skills,'[]'), COALESCE(potential,0), COALESCE(spirit_id,0),
		COALESCE(power_bonus,0), bound_at
		FROM player_artifacts WHERE player_id = ?`

	a := &model.Artifact{}
	var awakenJSON string
	err := r.db.QueryRow(query, playerID).Scan(
		&a.ID, &a.PlayerID, &a.Name, &a.Type, &a.Quality, &a.Level, &a.Exp,
		&a.AttackBonus, &a.DefenseBonus, &a.HPBonus, &a.MpBonus, &a.SpeedBonus, &a.DodgeBonus,
		&a.SkillID, &awakenJSON, &a.Potential, &a.SpiritID, &a.PowerBonus, &a.BoundAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询法宝失败", zap.Error(err))
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}

	if len(awakenJSON) > 2 {
		if err := json.Unmarshal([]byte(awakenJSON), &a.AwakenSkills); err != nil {
			a.AwakenSkills = []int{}
		}
	} else {
		a.AwakenSkills = []int{}
	}

	return a, nil
}

// GetByID 根据ID查询法宝
func (r *ArtifactRepo) GetByID(id int64) (*model.Artifact, error) {
	query := `SELECT id, player_id, name, type, quality, level, exp,
		attack_bonus, defense_bonus, hp_bonus, mp_bonus, speed_bonus, dodge_bonus,
		skill_id, COALESCE(awaken_skills,'[]'), COALESCE(potential,0), COALESCE(spirit_id,0),
		COALESCE(power_bonus,0), bound_at
		FROM player_artifacts WHERE id = ?`

	a := &model.Artifact{}
	var awakenJSON string
	err := r.db.QueryRow(query, id).Scan(
		&a.ID, &a.PlayerID, &a.Name, &a.Type, &a.Quality, &a.Level, &a.Exp,
		&a.AttackBonus, &a.DefenseBonus, &a.HPBonus, &a.MpBonus, &a.SpeedBonus, &a.DodgeBonus,
		&a.SkillID, &awakenJSON, &a.Potential, &a.SpiritID, &a.PowerBonus, &a.BoundAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询法宝失败", zap.Error(err))
		return nil, fmt.Errorf("查询法宝失败: %w", err)
	}

	if len(awakenJSON) > 2 {
		json.Unmarshal([]byte(awakenJSON), &a.AwakenSkills)
	} else {
		a.AwakenSkills = []int{}
	}
	return a, nil
}

// GetMultipleByPlayerID 获取玩家所有法宝
func (r *ArtifactRepo) GetMultipleByPlayerID(playerID int64) ([]*model.Artifact, error) {
	query := `SELECT id, player_id, name, type, quality, level, exp,
		attack_bonus, defense_bonus, hp_bonus, mp_bonus, speed_bonus, dodge_bonus,
		skill_id, COALESCE(awaken_skills,'[]'), COALESCE(potential,0), COALESCE(spirit_id,0),
		COALESCE(power_bonus,0), bound_at
		FROM player_artifacts WHERE player_id = ? ORDER BY type ASC`

	rows, err := r.db.Query(query, playerID)
	if err != nil {
		r.log.Error("查询法宝列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询法宝列表失败: %w", err)
	}
	defer rows.Close()

	var artifacts []*model.Artifact
	for rows.Next() {
		a := &model.Artifact{}
		var awakenJSON string
		if err := rows.Scan(
			&a.ID, &a.PlayerID, &a.Name, &a.Type, &a.Quality, &a.Level, &a.Exp,
			&a.AttackBonus, &a.DefenseBonus, &a.HPBonus, &a.MpBonus, &a.SpeedBonus, &a.DodgeBonus,
			&a.SkillID, &awakenJSON, &a.Potential, &a.SpiritID, &a.PowerBonus, &a.BoundAt,
		); err != nil {
			r.log.Error("扫描法宝行失败", zap.Error(err))
			continue
		}
		if len(awakenJSON) > 2 {
			json.Unmarshal([]byte(awakenJSON), &a.AwakenSkills)
		} else {
			a.AwakenSkills = []int{}
		}
		artifacts = append(artifacts, a)
	}
	return artifacts, nil
}

// Update 更新法宝
func (r *ArtifactRepo) Update(a *model.Artifact) error {
	awakenJSON := "[]"
	if len(a.AwakenSkills) > 0 {
		b, _ := json.Marshal(a.AwakenSkills)
		awakenJSON = string(b)
	}

	query := `UPDATE player_artifacts SET name=?, type=?, quality=?, level=?, exp=?,
		attack_bonus=?, defense_bonus=?, hp_bonus=?, mp_bonus=?, speed_bonus=?, dodge_bonus=?,
		skill_id=?, awaken_skills=?, potential=?, spirit_id=?, power_bonus=?
		WHERE id=?`

	_, err := r.db.Exec(query,
		a.Name, a.Type, a.Quality, a.Level, a.Exp,
		a.AttackBonus, a.DefenseBonus, a.HPBonus, a.MpBonus, a.SpeedBonus, a.DodgeBonus,
		a.SkillID, awakenJSON, a.Potential, a.SpiritID, a.PowerBonus, a.ID,
	)
	if err != nil {
		r.log.Error("更新法宝失败", zap.Error(err))
		return fmt.Errorf("更新法宝失败: %w", err)
	}
	return nil
}

// DeleteByPlayerID 根据玩家ID删除法宝（解绑）
func (r *ArtifactRepo) DeleteByPlayerID(playerID int64) error {
	_, err := r.db.Exec("DELETE FROM player_artifacts WHERE player_id = ?", playerID)
	if err != nil {
		r.log.Error("删除法宝失败", zap.Error(err))
		return fmt.Errorf("删除法宝失败: %w", err)
	}
	return nil
}

// ============================================================
// 器灵 CRUD
// ============================================================

// CreateSpirit 创建器灵
func (r *ArtifactRepo) CreateSpirit(s *model.ArtifactSpirit) error {
	query := `INSERT INTO artifact_spirits (artifact_id, player_id, name, personality,
		bond_level, bond_exp, bond_unlocked, last_dialogue, last_event, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	s.CreatedAt = now

	result, err := r.db.Exec(query,
		s.ArtifactID, s.PlayerID, s.Name, s.Personality,
		s.BondLevel, s.BondExp, s.BondUnlocked, s.LastDialogue, s.LastEvent, s.CreatedAt,
	)
	if err != nil {
		r.log.Error("创建器灵失败", zap.Error(err))
		return fmt.Errorf("创建器灵失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取器灵自增ID失败: %w", err)
	}
	s.ID = id
	return nil
}

// GetSpiritByArtifactID 根据法宝ID查询器灵
func (r *ArtifactRepo) GetSpiritByArtifactID(artifactID int64) (*model.ArtifactSpirit, error) {
	query := `SELECT id, artifact_id, player_id, name, personality,
		bond_level, bond_exp, bond_unlocked, last_dialogue, last_event, created_at
		FROM artifact_spirits WHERE artifact_id = ?`

	s := &model.ArtifactSpirit{}
	err := r.db.QueryRow(query, artifactID).Scan(
		&s.ID, &s.ArtifactID, &s.PlayerID, &s.Name, &s.Personality,
		&s.BondLevel, &s.BondExp, &s.BondUnlocked, &s.LastDialogue, &s.LastEvent, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询器灵失败", zap.Error(err))
		return nil, fmt.Errorf("查询器灵失败: %w", err)
	}
	return s, nil
}

// GetSpiritByPlayerID 根据玩家ID查询器灵
func (r *ArtifactRepo) GetSpiritByPlayerID(playerID int64) (*model.ArtifactSpirit, error) {
	query := `SELECT id, artifact_id, player_id, name, personality,
		bond_level, bond_exp, bond_unlocked, last_dialogue, last_event, created_at
		FROM artifact_spirits WHERE player_id = ?`

	s := &model.ArtifactSpirit{}
	err := r.db.QueryRow(query, playerID).Scan(
		&s.ID, &s.ArtifactID, &s.PlayerID, &s.Name, &s.Personality,
		&s.BondLevel, &s.BondExp, &s.BondUnlocked, &s.LastDialogue, &s.LastEvent, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询器灵失败", zap.Error(err))
		return nil, fmt.Errorf("查询器灵失败: %w", err)
	}
	return s, nil
}

// UpdateSpirit 更新器灵
func (r *ArtifactRepo) UpdateSpirit(s *model.ArtifactSpirit) error {
	query := `UPDATE artifact_spirits SET name=?, personality=?, bond_level=?, bond_exp=?,
		bond_unlocked=?, last_dialogue=?, last_event=? WHERE id=?`

	_, err := r.db.Exec(query,
		s.Name, s.Personality, s.BondLevel, s.BondExp,
		s.BondUnlocked, s.LastDialogue, s.LastEvent, s.ID,
	)
	if err != nil {
		r.log.Error("更新器灵失败", zap.Error(err))
		return fmt.Errorf("更新器灵失败: %w", err)
	}
	return nil
}

// ============================================================
// 试炼进度 CRUD
// ============================================================

// CreateTrialProgress 创建试炼进度
func (r *ArtifactRepo) CreateTrialProgress(tp *model.ArtifactTrialProgress) error {
	completedJSON := "[]"
	if len(tp.CompletedStages) > 0 {
		b, _ := json.Marshal(tp.CompletedStages)
		completedJSON = string(b)
	}

	query := `INSERT INTO artifact_trial_progress (player_id, artifact_id, completed_stages,
		last_completed_stage, today_attempts, last_attempt_date)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		tp.PlayerID, tp.ArtifactID, completedJSON,
		tp.LastCompletedStage, tp.TodayAttempts, tp.LastAttemptDate,
	)
	if err != nil {
		r.log.Error("创建试炼进度失败", zap.Error(err))
		return fmt.Errorf("创建试炼进度失败: %w", err)
	}
	return nil
}

// GetTrialProgress 查询试炼进度
func (r *ArtifactRepo) GetTrialProgress(artifactID int64) (*model.ArtifactTrialProgress, error) {
	query := `SELECT id, player_id, artifact_id, COALESCE(completed_stages,'[]'),
		COALESCE(last_completed_stage,0), COALESCE(today_attempts,0),
		COALESCE(last_attempt_date,'')
		FROM artifact_trial_progress WHERE artifact_id = ?`

	tp := &model.ArtifactTrialProgress{}
	var completedJSON string
	err := r.db.QueryRow(query, artifactID).Scan(
		&tp.PlayerID, &tp.ArtifactID, &completedJSON,
		&tp.LastCompletedStage, &tp.TodayAttempts, &tp.LastAttemptDate,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		// Fallback: try without id column
		query2 := `SELECT player_id, artifact_id, COALESCE(completed_stages,'[]'),
			COALESCE(last_completed_stage,0), COALESCE(today_attempts,0),
			COALESCE(last_attempt_date,'')
			FROM artifact_trial_progress WHERE artifact_id = ?`
		err2 := r.db.QueryRow(query2, artifactID).Scan(
			&tp.PlayerID, &tp.ArtifactID, &completedJSON,
			&tp.LastCompletedStage, &tp.TodayAttempts, &tp.LastAttemptDate,
		)
		if err2 == sql.ErrNoRows {
			return nil, nil
		}
		if err2 != nil {
			r.log.Error("查询试炼进度失败", zap.Error(err))
			return nil, fmt.Errorf("查询试炼进度失败: %w", err)
		}
	}

	if len(completedJSON) > 2 {
		json.Unmarshal([]byte(completedJSON), &tp.CompletedStages)
	} else {
		tp.CompletedStages = []int{}
	}
	return tp, nil
}

// UpdateTrialProgress 更新试炼进度
func (r *ArtifactRepo) UpdateTrialProgress(tp *model.ArtifactTrialProgress) error {
	completedJSON := "[]"
	if len(tp.CompletedStages) > 0 {
		b, _ := json.Marshal(tp.CompletedStages)
		completedJSON = string(b)
	}

	query := `UPDATE artifact_trial_progress SET completed_stages=?, last_completed_stage=?,
		today_attempts=?, last_attempt_date=? WHERE artifact_id=? AND player_id=?`

	_, err := r.db.Exec(query,
		completedJSON, tp.LastCompletedStage,
		tp.TodayAttempts, tp.LastAttemptDate,
		tp.ArtifactID, tp.PlayerID,
	)
	if err != nil {
		r.log.Error("更新试炼进度失败", zap.Error(err))
		return fmt.Errorf("更新试炼进度失败: %w", err)
	}
	return nil
}

// DeleteTrialProgress 删除试炼进度
func (r *ArtifactRepo) DeleteTrialProgress(artifactID int64) error {
	_, err := r.db.Exec("DELETE FROM artifact_trial_progress WHERE artifact_id = ?", artifactID)
	if err != nil {
		r.log.Error("删除试炼进度失败", zap.Error(err))
		return fmt.Errorf("删除试炼进度失败: %w", err)
	}
	return nil
}
