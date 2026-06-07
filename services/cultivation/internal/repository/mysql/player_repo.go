// Package mysql 提供修炼系统 MySQL 持久化层
package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"cultivation-game/services/cultivation/internal/model"
)

// PlayerRepo 修炼系统 MySQL 数据访问
type PlayerRepo struct {
	logger *slog.Logger
	db *sql.DB
}

// NewPlayerRepo 创建 PlayerRepo
func NewPlayerRepo(db *sql.DB, logger *slog.Logger) *PlayerRepo {
	return &PlayerRepo{db: db, logger: logger}
}

// SavePlayer 保存玩家数据（INSERT ON DUPLICATE KEY UPDATE）
func (r *PlayerRepo) SavePlayer(p *model.Player) error {
	spiritRootJSON, err := json.Marshal(p.SpiritRoots)
	if err != nil {
		return fmt.Errorf("序列化灵根数据失败: %w", err)
	}

	pillJSON, err := json.Marshal(p.PillBonuses)
	if err != nil {
		return fmt.Errorf("序列化丹药数据失败: %w", err)
	}

	artifactJSON, err := json.Marshal(p.ArtifactBonuses)
	if err != nil {
		return fmt.Errorf("序列化法宝数据失败: %w", err)
	}

	query := `INSERT INTO cultivation_players
		(player_id, nickname, realm_id, realm_level, exp, spirit_root,
		 base_attack, base_defense, base_hp,
		 technique_id, technique_level,
		 is_meditating, meditation_start, accumulated_exp,
		 pill_bonuses, artifact_bonuses,
		 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			nickname         = VALUES(nickname),
			realm_id         = VALUES(realm_id),
			realm_level      = VALUES(realm_level),
			exp              = VALUES(exp),
			spirit_root      = VALUES(spirit_root),
			base_attack      = VALUES(base_attack),
			base_defense     = VALUES(base_defense),
			base_hp          = VALUES(base_hp),
			technique_id     = VALUES(technique_id),
			technique_level  = VALUES(technique_level),
			is_meditating    = VALUES(is_meditating),
			meditation_start = VALUES(meditation_start),
			accumulated_exp  = VALUES(accumulated_exp),
			pill_bonuses     = VALUES(pill_bonuses),
			artifact_bonuses = VALUES(artifact_bonuses),
			updated_at       = NOW()`

	_, err = r.db.Exec(query,
		p.ID, p.Name, p.RealmID, p.RealmLevel, p.Experience, string(spiritRootJSON),
		p.BaseAttack, p.BaseDefense, p.BaseHP,
		p.TechniqueID, p.TechniqueLevel,
		boolToInt(p.IsMeditating), p.MeditationStart, p.AccumulatedExp,
		string(pillJSON), string(artifactJSON),
	)
	if err != nil {
		return fmt.Errorf("保存玩家%d失败: %w", p.ID, err)
	}

	return nil
}

// GetPlayer 根据ID获取玩家，返回 nil 表示不存在
func (r *PlayerRepo) GetPlayer(id uint64) (*model.Player, error) {
	query := `SELECT player_id, nickname, realm_id, realm_level, exp, spirit_root,
		base_attack, base_defense, base_hp,
		technique_id, technique_level,
		is_meditating, meditation_start, accumulated_exp,
		pill_bonuses, artifact_bonuses
		FROM cultivation_players WHERE player_id = ?`

	p := &model.Player{}
	var (
		spiritRootJSON  sql.NullString
		pillJSON        sql.NullString
		artifactJSON    sql.NullString
		isMeditatingInt int
	)

	err := r.db.QueryRow(query, id).Scan(
		&p.ID, &p.Name, &p.RealmID, &p.RealmLevel, &p.Experience, &spiritRootJSON,
		&p.BaseAttack, &p.BaseDefense, &p.BaseHP,
		&p.TechniqueID, &p.TechniqueLevel,
		&isMeditatingInt, &p.MeditationStart, &p.AccumulatedExp,
		&pillJSON, &artifactJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询玩家%d失败: %w", id, err)
	}

	p.IsMeditating = isMeditatingInt != 0

	// 反序列化灵根
	if spiritRootJSON.Valid && spiritRootJSON.String != "" {
		if err := json.Unmarshal([]byte(spiritRootJSON.String), &p.SpiritRoots); err != nil {
			r.logger.Error("反序列化灵根数据失败", "error", err)
		}
	}
	if p.SpiritRoots == nil {
		p.SpiritRoots = make(map[string]float64)
	}

	// 反序列化丹药加成
	if pillJSON.Valid && pillJSON.String != "" {
		if err := json.Unmarshal([]byte(pillJSON.String), &p.PillBonuses); err != nil {
			r.logger.Error("反序列化丹药数据失败", "error", err)
		}
	}
	if p.PillBonuses == nil {
		p.PillBonuses = make(map[string]float64)
	}

	// 反序列化法宝加成
	if artifactJSON.Valid && artifactJSON.String != "" {
		if err := json.Unmarshal([]byte(artifactJSON.String), &p.ArtifactBonuses); err != nil {
			r.logger.Error("反序列化法宝数据失败", "error", err)
		}
	}
	if p.ArtifactBonuses == nil {
		p.ArtifactBonuses = make(map[string]float64)
	}

	return p, nil
}

// CreatePlayer 创建玩家并返回自增ID
func (r *PlayerRepo) CreatePlayer(p *model.Player) (uint64, error) {
	spiritRootJSON, err := json.Marshal(p.SpiritRoots)
	if err != nil {
		return 0, fmt.Errorf("序列化灵根数据失败: %w", err)
	}

	pillJSON, err := json.Marshal(p.PillBonuses)
	if err != nil {
		return 0, fmt.Errorf("序列化丹药数据失败: %w", err)
	}

	artifactJSON, err := json.Marshal(p.ArtifactBonuses)
	if err != nil {
		return 0, fmt.Errorf("序列化法宝数据失败: %w", err)
	}

	query := `INSERT INTO cultivation_players
		(nickname, realm_id, realm_level, exp, spirit_root,
		 base_attack, base_defense, base_hp,
		 technique_id, technique_level,
		 is_meditating, meditation_start, accumulated_exp,
		 pill_bonuses, artifact_bonuses,
		 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`

	result, err := r.db.Exec(query,
		p.Name, p.RealmID, p.RealmLevel, p.Experience, string(spiritRootJSON),
		p.BaseAttack, p.BaseDefense, p.BaseHP,
		p.TechniqueID, p.TechniqueLevel,
		boolToInt(p.IsMeditating), p.MeditationStart, p.AccumulatedExp,
		string(pillJSON), string(artifactJSON),
	)
	if err != nil {
		return 0, fmt.Errorf("创建玩家失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取自增ID失败: %w", err)
	}

	return uint64(id), nil
}

// UpdatePlayerExp 更新修为值
func (r *PlayerRepo) UpdatePlayerExp(id uint64, exp int64) error {
	_, err := r.db.Exec("UPDATE cultivation_players SET exp = ? WHERE player_id = ?", exp, id)
	if err != nil {
		return fmt.Errorf("更新玩家%d修为失败: %w", id, err)
	}
	return nil
}

// UpdatePlayerRealm 更新境界
func (r *PlayerRepo) UpdatePlayerRealm(id uint64, realmID, realmLevel int) error {
	_, err := r.db.Exec(
		"UPDATE cultivation_players SET realm_id = ?, realm_level = ? WHERE player_id = ?",
		realmID, realmLevel, id,
	)
	if err != nil {
		return fmt.Errorf("更新玩家%d境界失败: %w", id, err)
	}
	return nil
}

// PlayerTechniqueRecord 带装备标志的功法记录（数据库映射用）
type PlayerTechniqueRecord struct {
	TechniqueID int  `json:"technique_id"`
	Level       int  `json:"level"`
	IsEquipped  bool `json:"is_equipped"`
}

// SaveTechnique 保存功法（INSERT ON DUPLICATE KEY UPDATE）
func (r *PlayerRepo) SaveTechnique(playerID uint64, record *PlayerTechniqueRecord) error {
	query := `INSERT INTO cultivation_techniques
		(player_id, technique_id, level, is_equipped, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			level      = VALUES(level),
			is_equipped = VALUES(is_equipped),
			updated_at  = NOW()`

	isEquippedInt := 0
	if record.IsEquipped {
		isEquippedInt = 1
	}

	_, err := r.db.Exec(query, playerID, record.TechniqueID, record.Level, isEquippedInt)
	if err != nil {
		return fmt.Errorf("保存功法失败(player=%d,tech=%d): %w", playerID, record.TechniqueID, err)
	}
	return nil
}

// GetTechniques 获取玩家所有功法
func (r *PlayerRepo) GetTechniques(playerID uint64) ([]*PlayerTechniqueRecord, error) {
	query := `SELECT technique_id, level, is_equipped
		FROM cultivation_techniques WHERE player_id = ? ORDER BY technique_id`

	rows, err := r.db.Query(query, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家%d功法失败: %w", playerID, err)
	}
	defer rows.Close()

	var records []*PlayerTechniqueRecord
	for rows.Next() {
		rec := &PlayerTechniqueRecord{}
		var equippedInt int
		if err := rows.Scan(&rec.TechniqueID, &rec.Level, &equippedInt); err != nil {
			return nil, fmt.Errorf("扫描功法数据失败: %w", err)
		}
		rec.IsEquipped = equippedInt != 0
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历功法数据失败: %w", err)
	}

	return records, nil
}

// Ping 检查数据库连接
func (r *PlayerRepo) Ping() error {
	return r.db.Ping()
}

// --- helper ---

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
