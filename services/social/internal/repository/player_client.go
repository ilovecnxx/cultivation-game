// Package repository 提供数据访问层实现，包括跨服务 HTTP 客户端
package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PlayerClient 通过 HTTP 调用 player 服务的客户端
type PlayerClient struct {
	baseURL string
	client  *http.Client
}

// NewPlayerClient 创建 PlayerClient
// playerAddr: player 服务地址, 例如 "http://player:8080"
func NewPlayerClient(playerAddr string) *PlayerClient {
	return &PlayerClient{
		baseURL: playerAddr,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetPlayerRealm 获取玩家境界
func (c *PlayerClient) GetPlayerRealm(ctx context.Context, userID string) (string, error) {
	// 调用 player 服务获取境界
	url := fmt.Sprintf("%s/api/v1/player/user/%s", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "qi_refining", nil // 降级返回炼气期
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Player *struct {
			Realm int32 `json:"realm"`
		} `json:"player"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "qi_refining", nil
	}

	if result.Player == nil {
		return "qi_refining", nil
	}

	// 境界 int32 → realm string 映射
	realmMap := map[int32]string{
		1: "mortal",
		2: "qi_refining",
		3: "foundation_build",
		4: "golden_core",
		5: "nascent_soul",
	}
	realm, ok := realmMap[result.Player.Realm]
	if !ok {
		return "qi_refining", nil
	}
	return realm, nil
}

// AddPlayerExp 给玩家增加修为
func (c *PlayerClient) AddPlayerExp(ctx context.Context, userID string, exp int64) error {
	if exp == 0 {
		return nil
	}

	// 先查询玩家ID
	playerID, err := c.getPlayerID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取玩家ID失败: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/player/%d/add-exp", c.baseURL, playerID)
	payload := map[string]interface{}{"exp": exp}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("调用玩家服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("玩家服务返回错误: %d", resp.StatusCode)
	}
	return nil
}

// GetPlayerExp 获取玩家修为
func (c *PlayerClient) GetPlayerExp(ctx context.Context, userID string) (int64, error) {
	playerID, err := c.getPlayerID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("获取玩家ID失败: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/player/%d", c.baseURL, playerID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("调用玩家服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Player *struct {
			SpiritPower int64 `json:"spirit_power"`
		} `json:"player"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, nil
	}
	if result.Player == nil {
		return 0, nil
	}
	return result.Player.SpiritPower, nil
}

// GetPlayerAttrs 获取玩家战斗属性
func (c *PlayerClient) GetPlayerAttrs(ctx context.Context, userID string) (map[string]int64, error) {
	playerID, err := c.getPlayerID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取玩家ID失败: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/player/%d/attributes", c.baseURL, playerID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		// 降级返回默认属性
		return map[string]int64{"hp": 1000, "attack": 100, "defense": 50}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data map[string]int64 `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]int64{"hp": 1000, "attack": 100, "defense": 50}, nil
	}
	if result.Data == nil {
		return map[string]int64{"hp": 1000, "attack": 100, "defense": 50}, nil
	}
	return result.Data, nil
}

// getPlayerID 根据 userID 获取 playerID
func (c *PlayerClient) getPlayerID(ctx context.Context, userID string) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/player/user/%s", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("查询玩家失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Player *struct {
			ID int64 `json:"id"`
		} `json:"player"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("解析玩家数据失败: %w", err)
	}
	if result.Player == nil {
		return 0, fmt.Errorf("玩家不存在: %s", userID)
	}
	return result.Player.ID, nil
}
