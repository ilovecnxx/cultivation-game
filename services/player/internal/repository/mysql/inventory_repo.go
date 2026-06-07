package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"cultivation-game/services/player/internal/model"

	"go.uber.org/zap"
)

// InventoryRepo 背包/装备数据访问
type InventoryRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewInventoryRepo 创建 InventoryRepo
func NewInventoryRepo(db *sql.DB, log *zap.Logger) *InventoryRepo {
	return &InventoryRepo{db: db, log: log}
}

// -------- 背包物品操作 --------

// InsertItem 向背包插入物品
func (r *InventoryRepo) InsertItem(inv *model.InventoryItem) error {
	query := `INSERT INTO inventory_items (player_id, item_id, quantity, slot_index, is_equipped, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	inv.CreatedAt = now
	inv.UpdatedAt = now

	result, err := r.db.Exec(query,
		inv.PlayerID, inv.ItemID, inv.Quantity, inv.SlotIndex, inv.IsEquipped,
		inv.CreatedAt, inv.UpdatedAt,
	)
	if err != nil {
		r.log.Error("插入背包物品失败", zap.Error(err))
		return fmt.Errorf("插入背包物品失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取插入ID失败: %w", err)
	}
	inv.ID = id
	return nil
}

// GetInventoryByPlayer 查询玩家背包所有物品
func (r *InventoryRepo) GetInventoryByPlayer(playerID int64) ([]*model.InventoryItem, error) {
	query := `SELECT ii.id, ii.player_id, ii.item_id, ii.quantity, ii.slot_index, ii.is_equipped,
		ii.created_at, ii.updated_at,
		i.id, i.name, i.type, i.quality, i.required_level, i.required_realm,
		i.description, i.max_stack, i.base_attack, i.base_defense, i.base_hp, i.base_mp,
		i.use_effect, i.sell_price, i.sell_price_bound, i.created_at
		FROM inventory_items ii
		JOIN items i ON i.id = ii.item_id
		WHERE ii.player_id = ?
		ORDER BY ii.slot_index ASC`

	rows, err := r.db.Query(query, playerID)
	if err != nil {
		r.log.Error("查询背包失败", zap.Error(err))
		return nil, fmt.Errorf("查询背包失败: %w", err)
	}
	defer rows.Close()

	return r.scanInventoryItems(rows)
}

// GetInventoryItem 查询单个背包物品（含关联物品模板）
func (r *InventoryRepo) GetInventoryItem(inventoryItemID int64) (*model.InventoryItem, error) {
	query := `SELECT ii.id, ii.player_id, ii.item_id, ii.quantity, ii.slot_index, ii.is_equipped,
		ii.created_at, ii.updated_at,
		i.id, i.name, i.type, i.quality, i.required_level, i.required_realm,
		i.description, i.max_stack, i.base_attack, i.base_defense, i.base_hp, i.base_mp,
		i.use_effect, i.sell_price, i.sell_price_bound, i.created_at
		FROM inventory_items ii
		JOIN items i ON i.id = ii.item_id
		WHERE ii.id = ?`

	items, err := r.scanInventoryItems(r.db.QueryRow(query, inventoryItemID))
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

// FindStackableItem 查找背包中可堆叠的同种物品
func (r *InventoryRepo) FindStackableItem(playerID, itemID int64) (*model.InventoryItem, error) {
	query := `SELECT ii.id, ii.player_id, ii.item_id, ii.quantity, ii.slot_index, ii.is_equipped,
		ii.created_at, ii.updated_at,
		i.id, i.name, i.type, i.quality, i.required_level, i.required_realm,
		i.description, i.max_stack, i.base_attack, i.base_defense, i.base_hp, i.base_mp,
		i.use_effect, i.sell_price, i.sell_price_bound, i.created_at
		FROM inventory_items ii
		JOIN items i ON i.id = ii.item_id
		WHERE ii.player_id = ? AND ii.item_id = ? AND ii.quantity < i.max_stack AND ii.is_equipped = false
		LIMIT 1`

	rows, qErr := r.db.Query(query, playerID, itemID)
	if qErr != nil {
		return nil, fmt.Errorf("查询堆叠物品失败: %w", qErr)
	}
	defer rows.Close()

	items, err := r.scanInventoryItems(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

// UpdateItemQuantity 更新物品数量
func (r *InventoryRepo) UpdateItemQuantity(id int64, quantity int32) error {
	query := `UPDATE inventory_items SET quantity=?, updated_at=? WHERE id=?`
	_, err := r.db.Exec(query, quantity, time.Now(), id)
	if err != nil {
		r.log.Error("更新物品数量失败", zap.Error(err))
		return fmt.Errorf("更新物品数量失败: %w", err)
	}
	return nil
}

// UpdateItemSlot 更新物品格子索引
func (r *InventoryRepo) UpdateItemSlot(id int64, slotIndex int32) error {
	query := `UPDATE inventory_items SET slot_index=?, updated_at=? WHERE id=?`
	_, err := r.db.Exec(query, slotIndex, time.Now(), id)
	if err != nil {
		r.log.Error("更新物品格子失败", zap.Error(err))
		return fmt.Errorf("更新物品格子失败: %w", err)
	}
	return nil
}

// UpdateItemEquipStatus 更新穿戴状态
func (r *InventoryRepo) UpdateItemEquipStatus(id int64, isEquipped bool) error {
	query := `UPDATE inventory_items SET is_equipped=?, updated_at=? WHERE id=?`
	_, err := r.db.Exec(query, isEquipped, time.Now(), id)
	if err != nil {
		r.log.Error("更新物品装备状态失败", zap.Error(err))
		return fmt.Errorf("更新物品装备状态失败: %w", err)
	}
	return nil
}

// DeleteItem 删除物品（数量归零或移除）
func (r *InventoryRepo) DeleteItem(id int64) error {
	_, err := r.db.Exec("DELETE FROM inventory_items WHERE id = ?", id)
	if err != nil {
		r.log.Error("删除物品失败", zap.Error(err))
		return fmt.Errorf("删除物品失败: %w", err)
	}
	return nil
}

// GetInventoryCount 查询背包中的物品数
func (r *InventoryRepo) GetInventoryCount(playerID int64) (int, error) {
	query := `SELECT COUNT(*) FROM inventory_items WHERE player_id = ? AND is_equipped = false`
	var count int
	err := r.db.QueryRow(query, playerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("查询背包数量失败: %w", err)
	}
	return count, nil
}

// -------- 装备操作 --------

// InsertEquipment 插入装备记录
func (r *InventoryRepo) InsertEquipment(eq *model.Equipment) error {
	query := `INSERT INTO equipments (player_id, slot, inventory_item_id, item_id, level, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	eq.CreatedAt = now
	eq.UpdatedAt = now

	result, err := r.db.Exec(query,
		eq.PlayerID, eq.Slot, eq.InventoryItemID, eq.ItemID, eq.Level,
		eq.CreatedAt, eq.UpdatedAt,
	)
	if err != nil {
		r.log.Error("插入装备记录失败", zap.Error(err))
		return fmt.Errorf("插入装备记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取装备ID失败: %w", err)
	}
	eq.ID = id
	return nil
}

// GetEquipmentByPlayer 查询玩家所有装备
func (r *InventoryRepo) GetEquipmentByPlayer(playerID int64) ([]*model.Equipment, error) {
	query := `SELECT e.id, e.player_id, e.slot, e.inventory_item_id, e.item_id, e.level,
		e.created_at, e.updated_at,
		i.id, i.name, i.type, i.quality, i.required_level, i.required_realm,
		i.description, i.max_stack, i.base_attack, i.base_defense, i.base_hp, i.base_mp,
		i.use_effect, i.sell_price, i.sell_price_bound, i.created_at
		FROM equipments e
		JOIN items i ON i.id = e.item_id
		WHERE e.player_id = ?
		ORDER BY e.slot ASC`

	rows, err := r.db.Query(query, playerID)
	if err != nil {
		r.log.Error("查询玩家装备失败", zap.Error(err))
		return nil, fmt.Errorf("查询装备失败: %w", err)
	}
	defer rows.Close()

	return r.scanEquipments(rows)
}

// GetEquipmentBySlot 查询指定槽位的装备
func (r *InventoryRepo) GetEquipmentBySlot(playerID, slot int64) (*model.Equipment, error) {
	query := `SELECT e.id, e.player_id, e.slot, e.inventory_item_id, e.item_id, e.level,
		e.created_at, e.updated_at,
		i.id, i.name, i.type, i.quality, i.required_level, i.required_realm,
		i.description, i.max_stack, i.base_attack, i.base_defense, i.base_hp, i.base_mp,
		i.use_effect, i.sell_price, i.sell_price_bound, i.created_at
		FROM equipments e
		JOIN items i ON i.id = e.item_id
		WHERE e.player_id = ? AND e.slot = ?`

	rows, qErr := r.db.Query(query, playerID, slot)
	if qErr != nil {
		return nil, fmt.Errorf("查询槽位装备失败: %w", qErr)
	}
	defer rows.Close()

	items, err := r.scanEquipments(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

// DeleteEquipment 删除装备记录
func (r *InventoryRepo) DeleteEquipment(id int64) error {
	_, err := r.db.Exec("DELETE FROM equipments WHERE id = ?", id)
	if err != nil {
		r.log.Error("删除装备记录失败", zap.Error(err))
		return fmt.Errorf("删除装备记录失败: %w", err)
	}
	return nil
}

// DeleteEquipmentBySlot 按槽位删除装备记录
func (r *InventoryRepo) DeleteEquipmentBySlot(playerID, slot int64) error {
	_, err := r.db.Exec("DELETE FROM equipments WHERE player_id = ? AND slot = ?", playerID, slot)
	if err != nil {
		r.log.Error("按槽位删除装备失败", zap.Error(err))
		return fmt.Errorf("删除装备失败: %w", err)
	}
	return nil
}

// UpdateEquipmentLevel 更新装备强化等级
func (r *InventoryRepo) UpdateEquipmentLevel(id int64, level int32) error {
	query := `UPDATE equipments SET level=?, updated_at=? WHERE id=?`
	_, err := r.db.Exec(query, level, time.Now(), id)
	if err != nil {
		r.log.Error("更新装备等级失败", zap.Error(err))
		return fmt.Errorf("更新装备等级失败: %w", err)
	}
	return nil
}

// -------- 物品模板查询 --------

// GetItem 根据ID查询物品模板
func (r *InventoryRepo) GetItem(itemID int64) (*model.Item, error) {
	query := `SELECT id, name, type, quality, required_level, required_realm,
		description, max_stack, base_attack, base_defense, base_hp, base_mp,
		use_effect, sell_price, sell_price_bound, created_at
		FROM items WHERE id = ?`

	item := &model.Item{}
	err := r.db.QueryRow(query, itemID).Scan(
		&item.ID, &item.Name, &item.Type, &item.Quality,
		&item.RequiredLevel, &item.RequiredRealm,
		&item.Description, &item.MaxStack,
		&item.BaseAttack, &item.BaseDefense, &item.BaseHP, &item.BaseMP,
		&item.UseEffect, &item.SellPrice, &item.SellPriceBound,
		&item.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询物品模板失败", zap.Error(err))
		return nil, fmt.Errorf("查询物品失败: %w", err)
	}
	return item, nil
}

// ListItems 查询多个物品模板（按ID列表）
func (r *InventoryRepo) ListItems(ids []int64) (map[int64]*model.Item, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `SELECT id, name, type, quality, required_level, required_realm,
		description, max_stack, base_attack, base_defense, base_hp, base_mp,
		use_effect, sell_price, sell_price_bound, created_at
		FROM items WHERE id IN (` + placeholders(len(ids)) + `)`

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		r.log.Error("批量查询物品失败", zap.Error(err))
		return nil, fmt.Errorf("批量查询物品失败: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]*model.Item)
	for rows.Next() {
		item := &model.Item{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.Type, &item.Quality,
			&item.RequiredLevel, &item.RequiredRealm,
			&item.Description, &item.MaxStack,
			&item.BaseAttack, &item.BaseDefense, &item.BaseHP, &item.BaseMP,
			&item.UseEffect, &item.SellPrice, &item.SellPriceBound,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描物品行失败: %w", err)
		}
		result[item.ID] = item
	}
	return result, nil
}

// GetEquipmentByID 根据ID查询装备
func (r *InventoryRepo) GetEquipmentByID(equipmentID int64) (*model.Equipment, error) {
	query := `SELECT e.id, e.player_id, e.slot, e.inventory_item_id, e.item_id, e.level,
		e.created_at, e.updated_at,
		i.id, i.name, i.type, i.quality, i.required_level, i.required_realm,
		i.description, i.max_stack, i.base_attack, i.base_defense, i.base_hp, i.base_mp,
		i.use_effect, i.sell_price, i.sell_price_bound, i.created_at
		FROM equipments e
		JOIN items i ON i.id = e.item_id
		WHERE e.id = ?`

	rows, err := r.db.Query(query, equipmentID)
	if err != nil {
		return nil, fmt.Errorf("查询装备失败: %w", err)
	}
	defer rows.Close()

	items, err := r.scanEquipments(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

// UpdateEquipmentMaxLevel 更新装备最大强化等级
func (r *InventoryRepo) UpdateEquipmentMaxLevel(id int64, maxLevel int32) error {
	query := `UPDATE equipments SET max_level=?, updated_at=? WHERE id=?`
	_, err := r.db.Exec(query, maxLevel, time.Now(), id)
	if err != nil {
		r.log.Error("更新装备最大等级失败", zap.Error(err))
		return fmt.Errorf("更新装备最大等级失败: %w", err)
	}
	return nil
}

// -------- 附魔操作 --------

// GetPlayerEnchantments 获取玩家所有附魔
func (r *InventoryRepo) GetPlayerEnchantments(playerID int64) ([]*model.PlayerEnchantment, error) {
	query := `SELECT id, player_id, equipment_id, enchant_id, level, slot_index, created_at, updated_at
		FROM player_enchantments WHERE player_id = ? ORDER BY equipment_id, slot_index ASC`

	rows, err := r.db.Query(query, playerID)
	if err != nil {
		r.log.Error("查询玩家附魔失败", zap.Error(err))
		return nil, fmt.Errorf("查询玩家附魔失败: %w", err)
	}
	defer rows.Close()

	var enchants []*model.PlayerEnchantment
	for rows.Next() {
		e := &model.PlayerEnchantment{}
		err := rows.Scan(&e.ID, &e.PlayerID, &e.EquipmentID, &e.EnchantID,
			&e.Level, &e.SlotIndex, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描附魔行失败: %w", err)
		}
		enchants = append(enchants, e)
	}
	return enchants, rows.Err()
}

// GetEquipmentEnchantments 获取装备的所有附魔
func (r *InventoryRepo) GetEquipmentEnchantments(equipmentID int64) ([]*model.PlayerEnchantment, error) {
	query := `SELECT id, player_id, equipment_id, enchant_id, level, slot_index, created_at, updated_at
		FROM player_enchantments WHERE equipment_id = ? ORDER BY slot_index ASC`

	rows, err := r.db.Query(query, equipmentID)
	if err != nil {
		r.log.Error("查询装备附魔失败", zap.Error(err))
		return nil, fmt.Errorf("查询装备附魔失败: %w", err)
	}
	defer rows.Close()

	var enchants []*model.PlayerEnchantment
	for rows.Next() {
		e := &model.PlayerEnchantment{}
		err := rows.Scan(&e.ID, &e.PlayerID, &e.EquipmentID, &e.EnchantID,
			&e.Level, &e.SlotIndex, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描附魔行失败: %w", err)
		}
		enchants = append(enchants, e)
	}
	return enchants, rows.Err()
}

// SaveEnchantment 保存附魔记录
func (r *InventoryRepo) SaveEnchantment(enchant *model.PlayerEnchantment) error {
	query := `INSERT INTO player_enchantments (player_id, equipment_id, enchant_id, level, slot_index, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	enchant.CreatedAt = now
	enchant.UpdatedAt = now

	result, err := r.db.Exec(query,
		enchant.PlayerID, enchant.EquipmentID, enchant.EnchantID,
		enchant.Level, enchant.SlotIndex, enchant.CreatedAt, enchant.UpdatedAt)
	if err != nil {
		r.log.Error("保存附魔记录失败", zap.Error(err))
		return fmt.Errorf("保存附魔失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取附魔ID失败: %w", err)
	}
	enchant.ID = id
	return nil
}

// RemoveEnchantment 删除附魔记录
func (r *InventoryRepo) RemoveEnchantment(id int64) error {
	_, err := r.db.Exec("DELETE FROM player_enchantments WHERE id = ?", id)
	if err != nil {
		r.log.Error("删除附魔记录失败", zap.Error(err))
		return fmt.Errorf("删除附魔失败: %w", err)
	}
	return nil
}

// -------- 觉醒操作 --------

// GetEquipmentAwakening 获取装备觉醒记录
func (r *InventoryRepo) GetEquipmentAwakening(equipmentID int64) (*model.EquipmentAwakening, error) {
	query := `SELECT id, player_id, equipment_id, awaken_level, created_at, updated_at
		FROM equipment_awakenings WHERE equipment_id = ?`

	awaken := &model.EquipmentAwakening{}
	err := r.db.QueryRow(query, equipmentID).Scan(
		&awaken.ID, &awaken.PlayerID, &awaken.EquipmentID,
		&awaken.AwakenLevel, &awaken.CreatedAt, &awaken.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("查询觉醒记录失败", zap.Error(err))
		return nil, fmt.Errorf("查询觉醒记录失败: %w", err)
	}
	return awaken, nil
}

// SaveAwakening 保存觉醒记录
func (r *InventoryRepo) SaveAwakening(awaken *model.EquipmentAwakening) error {
	query := `INSERT INTO equipment_awakenings (player_id, equipment_id, awaken_level, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`

	now := time.Now()
	awaken.CreatedAt = now
	awaken.UpdatedAt = now

	result, err := r.db.Exec(query,
		awaken.PlayerID, awaken.EquipmentID, awaken.AwakenLevel,
		awaken.CreatedAt, awaken.UpdatedAt)
	if err != nil {
		r.log.Error("保存觉醒记录失败", zap.Error(err))
		return fmt.Errorf("保存觉醒记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取觉醒ID失败: %w", err)
	}
	awaken.ID = id
	return nil
}

// UpdateAwakening 更新觉醒记录
func (r *InventoryRepo) UpdateAwakening(awaken *model.EquipmentAwakening) error {
	query := `UPDATE equipment_awakenings SET awaken_level=?, updated_at=? WHERE id=?`
	_, err := r.db.Exec(query, awaken.AwakenLevel, time.Now(), awaken.ID)
	if err != nil {
		r.log.Error("更新觉醒记录失败", zap.Error(err))
		return fmt.Errorf("更新觉醒记录失败: %w", err)
	}
	return nil
}

// -------- 辅助方法 --------

type scannable interface {
	Scan(dest ...any) error
}

func (r *InventoryRepo) scanInventoryItems(rows interface{}) ([]*model.InventoryItem, error) {
	var items []*model.InventoryItem

	switch v := rows.(type) {
	case *sql.Rows:
		for v.Next() {
			item, err := r.scanInventoryItem(v)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, v.Err()
	case *sql.Row:
		item, err := r.scanInventoryItem(v)
		if err != nil {
			return nil, err
		}
		return []*model.InventoryItem{item}, nil
	}
	return items, nil
}

func (r *InventoryRepo) scanInventoryItem(row scannable) (*model.InventoryItem, error) {
	inv := &model.InventoryItem{Item: &model.Item{}}
	err := row.Scan(
		&inv.ID, &inv.PlayerID, &inv.ItemID, &inv.Quantity, &inv.SlotIndex, &inv.IsEquipped,
		&inv.CreatedAt, &inv.UpdatedAt,
		&inv.Item.ID, &inv.Item.Name, &inv.Item.Type, &inv.Item.Quality,
		&inv.Item.RequiredLevel, &inv.Item.RequiredRealm,
		&inv.Item.Description, &inv.Item.MaxStack,
		&inv.Item.BaseAttack, &inv.Item.BaseDefense, &inv.Item.BaseHP, &inv.Item.BaseMP,
		&inv.Item.UseEffect, &inv.Item.SellPrice, &inv.Item.SellPriceBound,
		&inv.Item.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("扫描背包物品行失败: %w", err)
	}
	return inv, nil
}

func (r *InventoryRepo) scanEquipments(rows *sql.Rows) ([]*model.Equipment, error) {
	var equipments []*model.Equipment
	for rows.Next() {
		eq := &model.Equipment{Item: &model.Item{}}
		err := rows.Scan(
			&eq.ID, &eq.PlayerID, &eq.Slot, &eq.InventoryItemID, &eq.ItemID, &eq.Level,
			&eq.CreatedAt, &eq.UpdatedAt,
			&eq.Item.ID, &eq.Item.Name, &eq.Item.Type, &eq.Item.Quality,
			&eq.Item.RequiredLevel, &eq.Item.RequiredRealm,
			&eq.Item.Description, &eq.Item.MaxStack,
			&eq.Item.BaseAttack, &eq.Item.BaseDefense, &eq.Item.BaseHP, &eq.Item.BaseMP,
			&eq.Item.UseEffect, &eq.Item.SellPrice, &eq.Item.SellPriceBound,
			&eq.Item.CreatedAt,
		)
		if err != nil {
			r.log.Error("扫描装备行失败", zap.Error(err))
			return nil, fmt.Errorf("扫描装备行失败: %w", err)
		}
		equipments = append(equipments, eq)
	}
	return equipments, rows.Err()
}

// placeholders 生成 SQL 占位符 (?,?,?)
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, 0, n*2-1)
	for i := 0; i < n; i++ {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '?')
	}
	return string(b)
}
