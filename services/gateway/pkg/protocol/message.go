// Package protocol 定义消息协议。
//
// 生产环境应使用 protobuf 编解码，对应的 proto 定义见 api/proto/gateway.proto。
// 本文件使用 JSON 编码作为临时方案，确保无 protoc 依赖时也可编译。
// 替换为 protobuf 时，将 Encode/Decode 替换为 proto.Marshal/proto.Unmarshal 即可。
package protocol

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// packetPool 复用 Packet 对象以减少 GC 压力，适用于高频消息场景。
var packetPool = sync.Pool{
	New: func() interface{} {
		return &Packet{}
	},
}

// AcquirePacket 从池中获取一个 Packet 实例。
func AcquirePacket() *Packet {
	return packetPool.Get().(*Packet)
}

// ReleasePacket 将 Packet 归还到池中。
func ReleasePacket(p *Packet) {
	p.MsgID = 0
	p.PlayerID = 0
	p.Body = nil
	p.Timestamp = 0
	p.TraceID = ""
	packetPool.Put(p)
}

// 系统错误码
const (
	ErrSuccess       = 0
	ErrUnknown       = 1000
	ErrInvalidPacket = 1001
	ErrRateLimited   = 1002
	ErrUnauthorized  = 1003
	ErrInternal      = 1004
	ErrTimeout       = 1005
	ErrMsgIDNotFound = 1006
)

// Packet 网关消息包。
// 对应 proto: message Packet { uint32 msg_id = 1; uint64 player_id = 2; bytes body = 3; int64 timestamp = 4; string trace_id = 5; }
type Packet struct {
	MsgID     uint32 `json:"msg_id"`     // 消息 ID
	PlayerID  uint64 `json:"player_id"`  // 玩家 ID
	Body      []byte `json:"body"`       // 消息体（protobuf 编码的业务消息）
	Timestamp int64  `json:"timestamp"`  // 客户端时间戳（毫秒）
	TraceID   string `json:"trace_id"`   // 追踪 ID
}

// NewPacket 创建新的消息包。
func NewPacket(msgID uint32, playerID uint64, body []byte) *Packet {
	return &Packet{
		MsgID:     msgID,
		PlayerID:  playerID,
		Body:      body,
		Timestamp: time.Now().UnixMilli(),
	}
}

// Encode 编码消息包为 JSON。
// 生产环境可替换为 proto.Marshal(p)，当前兼容无 protoc 环境。
func Encode(p *Packet) ([]byte, error) {
	return json.Marshal(p)
}

// Decode 从 JSON 解码消息包。
// 生产环境可替换为 proto.Unmarshal(data, p)，当前兼容无 protoc 环境。
func Decode(data []byte) (*Packet, error) {
	var p Packet
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if p.MsgID == 0 {
		return nil, errors.New("msg_id is required")
	}
	return &p, nil
}

// ErrorPacket 创建错误响应包。
func ErrorPacket(code int, msg string) *Packet {
	return NewPacket(uint32(code), 0, []byte(msg))
}
