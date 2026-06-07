package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"cultivation-game/services/player/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FriendHandler struct {
	db            *sql.DB
	playerService *service.PlayerService
	log           *zap.Logger
}

func NewFriendHandler(db *sql.DB, ps *service.PlayerService, log *zap.Logger) *FriendHandler {
	return &FriendHandler{db: db, playerService: ps, log: log}
}

// AddFriend 添加好友 / 发送申请
func (h *FriendHandler) AddFriend(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	var req struct {
		FriendID int64  `json:"friend_id"`
		Message  string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 检查是否已有关系
	var status int
	h.db.QueryRow("SELECT status FROM friends WHERE player_id=? AND friend_id=?", playerID, req.FriendID).Scan(&status)
	if status == 1 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已是好友"})
		return
	}
	// 插入/更新申请
	h.db.Exec("INSERT INTO friends (player_id,friend_id,status) VALUES (?,?,0) ON DUPLICATE KEY UPDATE status=0", playerID, req.FriendID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "好友申请已发送"})
}

// AcceptFriend 接受好友申请
func (h *FriendHandler) AcceptFriend(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	var req struct{ FriendID int64 `json:"friend_id"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 双向添加
	h.db.Exec("UPDATE friends SET status=1 WHERE player_id=? AND friend_id=?", req.FriendID, playerID)
	h.db.Exec("INSERT INTO friends (player_id,friend_id,status) VALUES (?,?,1) ON DUPLICATE KEY UPDATE status=1", playerID, req.FriendID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已添加好友"})
}

// RemoveFriend 删除好友
func (h *FriendHandler) RemoveFriend(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	var req struct{ FriendID int64 `json:"friend_id"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	h.db.Exec("DELETE FROM friends WHERE (player_id=? AND friend_id=?) OR (player_id=? AND friend_id=?)", playerID, req.FriendID, req.FriendID, playerID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已删除好友"})
}

// ListFriends 好友列表（含在线状态）
func (h *FriendHandler) ListFriends(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	rows, _ := h.db.Query(`SELECT p.id,p.nickname,p.realm_id,p.realm_stage,p.spirit_root,
		CASE WHEN p.updated_at > DATE_SUB(NOW(), INTERVAL 5 MINUTE) THEN 1 ELSE 0 END as online
		FROM friends f JOIN players p ON f.friend_id=p.id WHERE f.player_id=? AND f.status=1`, playerID)
	defer rows.Close()
	type Friend struct {
		ID         int64  `json:"id"`
		Nickname   string `json:"nickname"`
		RealmID    int    `json:"realm_id"`
		RealmStage int    `json:"realm_stage"`
		SpiritRoot int    `json:"spirit_root"`
		Online     int    `json:"online"`
	}
	var list []Friend
	for rows.Next() {
		var f Friend; rows.Scan(&f.ID, &f.Nickname, &f.RealmID, &f.RealmStage, &f.SpiritRoot, &f.Online)
		list = append(list, f)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// PendingRequests 待处理的好友申请
func (h *FriendHandler) PendingRequests(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	rows, _ := h.db.Query(`SELECT p.id,p.nickname,p.realm_id FROM friends f JOIN players p ON f.player_id=p.id WHERE f.friend_id=? AND f.status=0`, playerID)
	defer rows.Close()
	type Request struct {
		ID       int64  `json:"id"`
		Nickname string `json:"nickname"`
		RealmID  int    `json:"realm_id"`
	}
	var list []Request
	for rows.Next() {
		var r Request; rows.Scan(&r.ID, &r.Nickname, &r.RealmID); list = append(list, r)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// SearchPlayers 搜索玩家
func (h *FriendHandler) SearchPlayers(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	kw := c.Query("q")
	if kw == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": []struct{}{}})
		return
	}
	rows, _ := h.db.Query("SELECT id,nickname,realm_id,realm_stage FROM players WHERE nickname LIKE ? AND id!=? LIMIT 10", "%"+kw+"%", playerID)
	defer rows.Close()
	type Player struct {
		ID         int64  `json:"id"`
		Nickname   string `json:"nickname"`
		RealmID    int    `json:"realm_id"`
		RealmStage int    `json:"realm_stage"`
	}
	var list []Player
	for rows.Next() {
		var p Player; rows.Scan(&p.ID, &p.Nickname, &p.RealmID, &p.RealmStage)
		list = append(list, p)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// GetPrivateMessages 获取私聊记录
func (h *FriendHandler) GetPrivateMessages(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	peerID, _ := strconv.ParseInt(c.Query("peer_id"), 10, 64)
	rows, _ := h.db.Query(`SELECT id,from_id,to_id,text,is_read,created_at FROM private_messages
		WHERE (from_id=? AND to_id=?) OR (from_id=? AND to_id=?) ORDER BY id DESC LIMIT 100`, playerID, peerID, peerID, playerID)
	defer rows.Close()
	type Msg struct {
		ID        int64  `json:"id"`
		FromID    int64  `json:"from_id"`
		ToID      int64  `json:"to_id"`
		Text      string `json:"text"`
		IsRead    int    `json:"is_read"`
		CreatedAt string `json:"created_at"`
	}
	var list []Msg
	for rows.Next() {
		var m Msg; rows.Scan(&m.ID, &m.FromID, &m.ToID, &m.Text, &m.IsRead, &m.CreatedAt)
		list = append(list, m)
	}
	h.db.Exec("UPDATE private_messages SET is_read=1 WHERE from_id=? AND to_id=? AND is_read=0", peerID, playerID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// SendPrivateMessage 发送私聊消息
func (h *FriendHandler) SendPrivateMessage(c *gin.Context) {
	playerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	jwtPID, _ := c.Get("player_id")
	if jwtPID.(int64) != playerID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权操作"})
		return
	}
	var req struct {
		ToID int64  `json:"to_id"`
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	h.db.Exec("INSERT INTO private_messages (from_id,to_id,text) VALUES (?,?,?)", playerID, req.ToID, req.Text)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "发送成功"})
}
