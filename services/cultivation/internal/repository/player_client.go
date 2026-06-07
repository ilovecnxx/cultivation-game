// Package repository 提供数据访问层实现，包括跨服务 HTTP 客户端
package repository

import (
	"bytes"
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
// playerAddr: player 服务地址, 例如 "http://player:8083"
func NewPlayerClient(playerAddr string) *PlayerClient {
	return &PlayerClient{
		baseURL: playerAddr,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// ProtectionStatus 保护状态响应
type ProtectionStatus struct {
	Protected                   bool  `json:"protected"`
	PvpProtected                bool  `json:"pvp_protected"`
	ProtectionRemainingSec      int64 `json:"protection_remaining_sec"`
	PvpProtectionRemainingSec   int64 `json:"pvp_protection_remaining_sec"`
	BreakthroughGraceRemaining  int32 `json:"breakthrough_grace_remaining"`
	FreeResurrectionRemaining   int32 `json:"free_resurrection_remaining"`
}

// GetProtectionStatus 获取玩家保护状态
func (c *PlayerClient) GetProtectionStatus(playerID int64) (*ProtectionStatus, error) {
	url := fmt.Sprintf("%s/api/v1/protection/status?player_id=%d", c.baseURL, playerID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("调用保护服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int               `json:"code"`
		Msg  string            `json:"msg"`
		Data *ProtectionStatus `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析保护状态失败: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("保护服务返回错误: %s", result.Msg)
	}
	return result.Data, nil
}

// GetBreakthroughGraceRemaining 获取玩家剩余突破免罚次数
func (c *PlayerClient) GetBreakthroughGraceRemaining(playerID int64) (int32, error) {
	status, err := c.GetProtectionStatus(playerID)
	if err != nil {
		return 0, err
	}
	if status == nil {
		return 0, nil
	}
	return status.BreakthroughGraceRemaining, nil
}

// GetFreeResurrectionRemaining 获取玩家剩余免费复活次数
func (c *PlayerClient) GetFreeResurrectionRemaining(playerID int64) (int32, error) {
	status, err := c.GetProtectionStatus(playerID)
	if err != nil {
		return 0, err
	}
	if status == nil {
		return 0, nil
	}
	return status.FreeResurrectionRemaining, nil
}

// UseBreakthroughGrace 使用一次突破免罚次数，返回减免比例
// 实现 service.ProtectionChecker 接口
func (c *PlayerClient) UseBreakthroughGrace(playerID uint64) (float64, error) {
	// 先查询减免比例
	url := fmt.Sprintf("%s/api/v1/protection/breakthrough-grace?player_id=%d", c.baseURL, playerID)
	resp, err := c.client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("调用突破免罚查询失败: %w", err)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data *struct {
			Reduction         float64 `json:"reduction"`
			HasGrace          bool    `json:"has_grace"`
			PenaltyMultiplier float64 `json:"penalty_multiplier"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("解析突破免罚响应失败: %w", err)
	}
	if result.Code != 0 {
		return 0, nil
	}
	if result.Data == nil || !result.Data.HasGrace {
		return 0, nil
	}

	// 有免罚次数，请求消耗一次
	useURL := fmt.Sprintf("%s/api/v1/protection/breakthrough-grace/use", c.baseURL)
	usePayload := []byte(fmt.Sprintf(`{"player_id": %d}`, playerID))
	useResp, err := c.client.Post(useURL, "application/json", bytes.NewReader(usePayload))
	if err != nil {
		return 0, fmt.Errorf("调用突破免罚消耗失败: %w", err)
	}
	defer useResp.Body.Close()

	if useResp.StatusCode >= 400 {
		return 0, nil
	}

	return result.Data.Reduction, nil
}
