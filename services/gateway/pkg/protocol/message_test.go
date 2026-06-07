// Package protocol 消息协议单元测试。
package protocol

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Error code constants
// ---------------------------------------------------------------------------

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"ErrSuccess", ErrSuccess, 0},
		{"ErrUnknown", ErrUnknown, 1000},
		{"ErrInvalidPacket", ErrInvalidPacket, 1001},
		{"ErrRateLimited", ErrRateLimited, 1002},
		{"ErrUnauthorized", ErrUnauthorized, 1003},
		{"ErrInternal", ErrInternal, 1004},
		{"ErrTimeout", ErrTimeout, 1005},
		{"ErrMsgIDNotFound", ErrMsgIDNotFound, 1006},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewPacket
// ---------------------------------------------------------------------------

func TestNewPacket(t *testing.T) {
	body := []byte(`{"action":"login"}`)
	packet := NewPacket(100, 10001, body)

	if packet.MsgID != 100 {
		t.Errorf("MsgID = %d, want %d", packet.MsgID, 100)
	}
	if packet.PlayerID != 10001 {
		t.Errorf("PlayerID = %d, want %d", packet.PlayerID, 10001)
	}
	if string(packet.Body) != string(body) {
		t.Errorf("Body = %q, want %q", string(packet.Body), string(body))
	}
	if packet.Timestamp == 0 {
		t.Error("Timestamp should be non-zero")
	}
	if packet.TraceID != "" {
		t.Errorf("TraceID = %q, want empty", packet.TraceID)
	}
}

func TestNewPacket_NilBody(t *testing.T) {
	packet := NewPacket(1, 0, nil)
	if packet.Body != nil {
		t.Errorf("Body = %v, want nil", packet.Body)
	}
}

// ---------------------------------------------------------------------------
// Encode / Decode roundtrip
// ---------------------------------------------------------------------------

func TestEncodeDecode_Roundtrip(t *testing.T) {
	tests := []struct {
		name     string
		packet   *Packet
	}{
		{
			"simple text body",
			&Packet{
				MsgID:     100,
				PlayerID:  10001,
				Body:      []byte(`{"action":"login"}`),
				Timestamp: time.Now().UnixMilli(),
				TraceID:   "trace-001",
			},
		},
		{
			"binary body",
			&Packet{
				MsgID:     200,
				PlayerID:  20002,
				Body:      []byte{0x00, 0x01, 0xFF, 0xFE},
				Timestamp: 1700000000000,
				TraceID:   "",
			},
		},
		{
			"empty body",
			&Packet{
				MsgID:     300,
				PlayerID:  30003,
				Body:      []byte{},
				Timestamp: 0,
				TraceID:   "",
			},
		},
		{
			"zero player ID",
			&Packet{
				MsgID:     1,
				PlayerID:  0,
				Body:      []byte(`{"system":"init"}`),
				Timestamp: 1700000000000,
				TraceID:   "trace-002",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Encode(tt.packet)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if len(data) == 0 {
				t.Fatal("Encode returned empty data")
			}

			decoded, err := Decode(data)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if decoded.MsgID != tt.packet.MsgID {
				t.Errorf("MsgID = %d, want %d", decoded.MsgID, tt.packet.MsgID)
			}
			if decoded.PlayerID != tt.packet.PlayerID {
				t.Errorf("PlayerID = %d, want %d", decoded.PlayerID, tt.packet.PlayerID)
			}
			if string(decoded.Body) != string(tt.packet.Body) {
				t.Errorf("Body = %v, want %v", decoded.Body, tt.packet.Body)
			}
			if decoded.Timestamp != tt.packet.Timestamp {
				t.Errorf("Timestamp = %d, want %d", decoded.Timestamp, tt.packet.Timestamp)
			}
			if decoded.TraceID != tt.packet.TraceID {
				t.Errorf("TraceID = %q, want %q", decoded.TraceID, tt.packet.TraceID)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Large payload handling
// ---------------------------------------------------------------------------

func TestEncodeDecode_LargePayload(t *testing.T) {
	body := make([]byte, 100*1024) // 100 KB
	for i := range body {
		body[i] = byte(i % 256)
	}

	packet := &Packet{
		MsgID:     700,
		PlayerID:  70007,
		Body:      body,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := Encode(packet)
	if err != nil {
		t.Fatalf("Encode failed for large payload: %v", err)
	}

	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode failed for large payload: %v", err)
	}

	if len(decoded.Body) != len(body) {
		t.Fatalf("decoded body length = %d, want %d", len(decoded.Body), len(body))
	}
	for i := range body {
		if decoded.Body[i] != body[i] {
			t.Fatalf("body byte %d: got %02x, want %02x", i, decoded.Body[i], body[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Special characters in body
// ---------------------------------------------------------------------------

func TestEncodeDecode_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name string
		body []byte
	}{
		{"unicode CJK", []byte(`{"msg":"玩家登录"}`)},
		{"emoji", []byte(`{"icon":"🔥🎮"}`)},
		{"null bytes", []byte("data\x00with\x00nulls")},
		{"all printable ASCII", []byte(strings.Repeat("!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~", 10))},
		{"JSON special chars", []byte(`{"data":"line1\nline2\t tab \"quoted\""}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := &Packet{
				MsgID:     400,
				PlayerID:  40004,
				Body:      tt.body,
				Timestamp: time.Now().UnixMilli(),
			}

			data, err := Encode(packet)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			decoded, err := Decode(data)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if string(decoded.Body) != string(tt.body) {
				t.Errorf("Body mismatch: got %q, want %q", string(decoded.Body), string(tt.body))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Invalid JSON handling
// ---------------------------------------------------------------------------

func TestDecode_InvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"just whitespace", []byte("   ")},
		{"truncated JSON", []byte(`{"msg_id":123`)},
		{"random bytes", []byte("\x00\xFF\xFE\xFD")},
		{"plain text", []byte("not json at all")},
		{"array instead of object", []byte(`[1,2,3]`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.data)
			if err == nil {
				t.Error("expected error for invalid JSON, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Decode — missing msg_id
// ---------------------------------------------------------------------------

func TestDecode_MissingMsgID(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"empty object", `{}`},
		{"no msg_id field", `{"player_id":10001}`},
		{"msg_id is null", `{"msg_id":null}`},
		{"msg_id is string", `{"msg_id":"abc"}`},
		{"msg_id is zero", `{"msg_id":0}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode([]byte(tt.json))
			if err == nil {
				t.Error("expected error for missing msg_id, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ErrorPacket
// ---------------------------------------------------------------------------

func TestErrorPacket(t *testing.T) {
	tests := []struct {
		name  string
		code  int
		msg   string
	}{
		{"rate limited", ErrRateLimited, "too many requests"},
		{"unauthorized", ErrUnauthorized, "invalid token"},
		{"internal error", ErrInternal, "internal server error"},
		{"empty message", ErrUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ErrorPacket(tt.code, tt.msg)

			if p.MsgID != uint32(tt.code) {
				t.Errorf("MsgID = %d, want %d", p.MsgID, tt.code)
			}
			if p.PlayerID != 0 {
				t.Errorf("PlayerID = %d, want 0", p.PlayerID)
			}
			if string(p.Body) != tt.msg {
				t.Errorf("Body = %q, want %q", string(p.Body), tt.msg)
			}
			if p.Timestamp == 0 {
				t.Error("Timestamp should be non-zero")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Encode nil packet safety
// ---------------------------------------------------------------------------

func TestEncode_NilPacket(t *testing.T) {
	_, err := Encode(nil)
	// json.Marshal(nil) returns "null", which should be valid JSON.
	if err != nil {
		t.Fatalf("Encode(nil) failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Decode of valid JSON with extra fields
// ---------------------------------------------------------------------------

func TestDecode_ExtraFieldsIgnored(t *testing.T) {
	jsonData := []byte(`{"msg_id":100,"player_id":20002,"body":"eHl6","timestamp":1234567890,"trace_id":"t1","extra_field":"should_be_ignored"}`)

	p, err := Decode(jsonData)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if p.MsgID != 100 {
		t.Errorf("MsgID = %d, want %d", p.MsgID, 100)
	}
	if p.PlayerID != 20002 {
		t.Errorf("PlayerID = %d, want %d", p.PlayerID, 20002)
	}
	if p.TraceID != "t1" {
		t.Errorf("TraceID = %q, want %q", p.TraceID, "t1")
	}
}
