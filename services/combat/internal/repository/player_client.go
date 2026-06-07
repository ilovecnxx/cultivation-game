// Package repository 提供跨服务 HTTP 客户端
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
	Protected                bool  `json:"protected"`
	PvpProtected             bool  `json:"pvp_protected"`
	ProtectionRemainingSec   int64 `json:"protection_remaining_sec"`
	PvpProtectionRemainingSec int64 `json:"pvp_protection_remaining_sec"`
	BreakthroughGraceRemaining int32 `json:"breakthrough_grace_remaining"`
	FreeResurrectionRemaining int32 `json:"free_resurrection_remaining"`
}

// GetProtectionStatus 获取玩家保护状态
func (c *PlayerClient) GetProtectionStatus(playerID string) (*ProtectionStatus, error) {
	url := fmt.Sprintf("%s/api/v1/protection/status?player_id=%s", c.baseURL, playerID)
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

// IsPvpProtected 检查玩家是否处于 PVP 保护
func (c *PlayerClient) IsPvpProtected(playerID string) (bool, error) {
	status, err := c.GetProtectionStatus(playerID)
	if err != nil {
		return false, err
	}
	if status == nil {
		return false, nil
	}
	return status.PvpProtected, nil
}

// UseFreeResurrection 使用一次免费复活次数，返回是否成功
func (c *PlayerClient) UseFreeResurrection(playerID string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/protection/free-resurrection/use", c.baseURL)
	payload := []byte(fmt.Sprintf(`{"player_id": %s}`, playerID))
	resp, err := c.client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return false, fmt.Errorf("调用免费复活失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("解析免费复活响应失败: %w", err)
	}
	if result.Code != 0 {
		return false, nil
	}
	return true, nil
}

// GetFreeResurrectionRemaining 获取玩家剩余免费复活次数
func (c *PlayerClient) GetFreeResurrectionRemaining(playerID string) (int32, error) {
	status, err := c.GetProtectionStatus(playerID)
	if err != nil {
		return 0, err
	}
	if status == nil {
		return 0, nil
	}
	return status.FreeResurrectionRemaining, nil
}
