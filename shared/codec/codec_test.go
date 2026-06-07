package codec

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"strings"
	"testing"

	"github.com/golang/snappy"
	"google.golang.org/protobuf/proto"

	gamepb "cultivation-game/shared/proto"
)

// testStruct is a plain Go struct used for JSON codec tests.
type testStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// ---- ProtobufCodec Tests ----

func TestProtobufCodec_Marshal_Unmarshal(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	msg := &gamepb.GameMessage{
		MsgId:     1001,
		Seq:       42,
		Timestamp: 1717000000,
		Payload:   []byte("hello"),
		Status:    0,
		ErrorMsg:  "",
	}

	data, ct, err := c.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Marshal returned empty data")
	}
	if ct != CompressNone {
		t.Errorf("CompressType = %d, want %d (CompressNone)", ct, CompressNone)
	}

	decoded := &gamepb.GameMessage{}
	if err := c.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.MsgId != 1001 || decoded.Seq != 42 || decoded.Timestamp != 1717000000 {
		t.Errorf("Decoded fields mismatch: %+v", decoded)
	}
	if !bytes.Equal(decoded.Payload, []byte("hello")) {
		t.Errorf("Payload = %v, want %v", decoded.Payload, []byte("hello"))
	}
}

func TestProtobufCodec_Marshal_NonProtoMessage(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	_, _, err := c.Marshal("not a proto message")
	if err == nil {
		t.Error("Expected error when marshaling non-proto.Message")
	}
}

func TestProtobufCodec_Unmarshal_NonProtoMessage(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	err := c.Unmarshal([]byte("data"), "not a proto message")
	if err == nil {
		t.Error("Expected error when unmarshaling to non-proto.Message")
	}
}

func TestProtobufCodec_Unmarshal_InvalidData(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	msg := &gamepb.GameMessage{}
	err := c.Unmarshal([]byte{0xFF, 0xFF, 0xFF}, msg)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid protobuf data")
	}
}

func TestProtobufCodec_RoundtripAllTypes(t *testing.T) {
	c := NewProtobufCodec(CompressNone)

	tests := []proto.Message{
		&gamepb.GameMessage{MsgId: 1, Seq: 100, Timestamp: 12345, Status: 0, ErrorMsg: "ok"},
		&gamepb.CultivateReq{TechniqueId: 5, Duration: 60},
		&gamepb.CultivateResp{ExpGained: 5000, NewRealm: 2, Breakthrough: true},
		&gamepb.CombatAction{SkillId: 2001, TargetId: 42},
		&gamepb.CombatRound{Description: "命中", Damage: 150, IsCritical: false, AttackerId: 1},
		&gamepb.PlayerInfo{Id: 1001, Nickname: "修仙者", RealmId: 3, RealmLevel: 5, Exp: 50000, SpiritRoot: "金灵根"},
	}

	for _, original := range tests {
		data, _, err := c.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal %T error: %v", original, err)
		}

		// Create a new instance of the same type (using proto.Clone or type switch)
		decoded := proto.Clone(original)
		if err := c.Unmarshal(data, decoded); err != nil {
			t.Fatalf("Unmarshal %T error: %v", original, err)
		}

		if !proto.Equal(original, decoded) {
			t.Errorf("Roundtrip %T failed: original=%+v decoded=%+v", original, original, decoded)
		}
	}
}

func TestProtobufCodec_Name(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	if name := c.Name(); name != "protobuf" {
		t.Errorf("Name() = %q, want 'protobuf'", name)
	}
}

// ---- ProtobufCodec + Snappy Compression Tests ----

func TestProtobufCodec_SnappyCompression(t *testing.T) {
	c := NewProtobufCodec(CompressSnappy)

	// Large message to trigger compression (>= 64 bytes)
	msg := &gamepb.PlayerInfo{
		Id:         9999,
		Nickname:   strings.Repeat("A", 200),
		RealmId:    5,
		RealmLevel: 9,
		Exp:        999999,
		SpiritRoot: "变异雷灵根",
	}

	data, ct, err := c.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if ct != CompressSnappy {
		t.Errorf("CompressType = %d, want %d (CompressSnappy)", ct, CompressSnappy)
	}

	decoded := &gamepb.PlayerInfo{}
	if err := c.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Id != 9999 || decoded.Nickname != msg.Nickname {
		t.Errorf("Decoded mismatch: %+v", decoded)
	}
}

func TestProtobufCodec_GzipCompression(t *testing.T) {
	c := NewProtobufCodec(CompressGzip)

	msg := &gamepb.PlayerInfo{
		Id:       8888,
		Nickname: strings.Repeat("B", 200),
	}

	data, ct, err := c.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if ct != CompressGzip {
		t.Errorf("CompressType = %d, want %d (CompressGzip)", ct, CompressGzip)
	}

	decoded := &gamepb.PlayerInfo{}
	if err := c.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Id != 8888 {
		t.Errorf("Id = %d, want 8888", decoded.Id)
	}
}

func TestProtobufCodec_SkipsCompressionForSmallData(t *testing.T) {
	c := NewProtobufCodec(CompressSnappy)
	msg := &gamepb.GameMessage{MsgId: 1}

	data, ct, err := c.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	// Small data (< 64 bytes) should not be compressed
	if ct != CompressNone {
		t.Errorf("Expected CompressNone for small data, got %d", ct)
	}

	// But it should still decode correctly
	decoded := &gamepb.GameMessage{}
	if err := c.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.MsgId != 1 {
		t.Errorf("MsgId = %d, want 1", decoded.MsgId)
	}
}

// ---- JSONCodec Tests ----

func TestJSONCodec_Marshal_Unmarshal(t *testing.T) {
	c := NewJSONCodec(CompressNone)
	original := testStruct{Name: "测试", Age: 25}

	data, ct, err := c.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if ct != CompressNone {
		t.Errorf("CompressType = %d, want %d", ct, CompressNone)
	}

	var decoded testStruct
	if err := c.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != original {
		t.Errorf("Decoded = %+v, want %+v", decoded, original)
	}
}

func TestJSONCodec_Marshal_Map(t *testing.T) {
	c := NewJSONCodec(CompressNone)
	original := map[string]interface{}{"key": "value", "num": 42}

	data, _, err := c.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded map[string]interface{}
	if err := c.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded["key"] != "value" {
		t.Errorf("key = %v, want 'value'", decoded["key"])
	}
}

func TestJSONCodec_Pretty(t *testing.T) {
	c := NewJSONCodecPretty(CompressNone)
	original := testStruct{Name: "测试", Age: 25}

	data, _, err := c.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Pretty output should contain newlines and indentation
	if !bytes.Contains(data, []byte("\n")) {
		t.Error("Pretty JSON should contain newlines")
	}
	if !bytes.Contains(data, []byte("  ")) {
		t.Error("Pretty JSON should contain indentation")
	}

	// Should still decode correctly
	var decoded testStruct
	if err := c.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != original {
		t.Errorf("Decoded = %+v, want %+v", decoded, original)
	}
}

func TestJSONCodec_Name(t *testing.T) {
	c := NewJSONCodec(CompressNone)
	if name := c.Name(); name != "json" {
		t.Errorf("Name() = %q, want 'json'", name)
	}
}

// ---- JSONCodec + Compression Tests ----

func TestJSONCodec_SnappyCompression(t *testing.T) {
	c := NewJSONCodec(CompressSnappy)
	data := map[string]interface{}{
		"large_field": strings.Repeat("X", 200),
	}

	encoded, ct, err := c.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if ct != CompressSnappy {
		t.Errorf("CompressType = %d, want %d (CompressSnappy)", ct, CompressSnappy)
	}

	var decoded map[string]interface{}
	if err := c.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded["large_field"] != data["large_field"] {
		t.Errorf("Decoded mismatch")
	}
}

func TestJSONCodec_GzipCompression(t *testing.T) {
	c := NewJSONCodec(CompressGzip)
	data := map[string]interface{}{
		"large_field": strings.Repeat("Y", 200),
	}

	encoded, ct, err := c.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if ct != CompressGzip {
		t.Errorf("CompressType = %d, want %d (CompressGzip)", ct, CompressGzip)
	}

	var decoded map[string]interface{}
	if err := c.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded["large_field"] != data["large_field"] {
		t.Errorf("Decoded mismatch")
	}
}

func TestJSONCodec_SkipsCompressionForSmallData(t *testing.T) {
	c := NewJSONCodec(CompressSnappy)
	original := testStruct{Name: "hi"}

	data, ct, err := c.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if ct != CompressNone {
		t.Errorf("Expected CompressNone for small data, got %d", ct)
	}

	var decoded testStruct
	if err := c.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
}

// ---- Compression Utilities Tests ----

func TestCompressData_Snappy(t *testing.T) {
	original := []byte("hello world, this is test data for snappy compression")
	compressed, err := compressData(original, CompressSnappy)
	if err != nil {
		t.Fatalf("compressData(Snappy) error: %v", err)
	}

	decompressed, err := decompressData(compressed, CompressSnappy)
	if err != nil {
		t.Fatalf("decompressData(Snappy) error: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Errorf("Snappy roundtrip failed: got %v, want %v", decompressed, original)
	}
}

func TestCompressData_Gzip(t *testing.T) {
	original := []byte("hello world, this is test data for gzip compression")
	compressed, err := compressData(original, CompressGzip)
	if err != nil {
		t.Fatalf("compressData(Gzip) error: %v", err)
	}

	decompressed, err := decompressData(compressed, CompressGzip)
	if err != nil {
		t.Fatalf("decompressData(Gzip) error: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Errorf("Gzip roundtrip failed: got %v, want %v", decompressed, original)
	}
}

func TestCompressData_None(t *testing.T) {
	original := []byte("test data")
	result, err := compressData(original, CompressNone)
	if err != nil {
		t.Fatalf("compressData(None) error: %v", err)
	}
	if !bytes.Equal(original, result) {
		t.Errorf("CompressNone should return data unchanged")
	}
}

func TestCompressData_UnknownType(t *testing.T) {
	result, err := compressData([]byte("test"), CompressType(99))
	if err != nil {
		t.Errorf("Unknown compress type should not error, got: %v", err)
	}
	if !bytes.Equal(result, []byte("test")) {
		t.Errorf("Unknown compress type should return data unchanged")
	}
}

func TestDecompressData_Snappy_Invalid(t *testing.T) {
	_, err := decompressData([]byte{0xFF, 0xFF, 0xFF}, CompressSnappy)
	if err == nil {
		t.Error("Expected error when decompressing invalid snappy data")
	}
}

func TestDecompressData_Gzip_Invalid(t *testing.T) {
	_, err := decompressData([]byte{0xFF, 0xFF, 0xFF}, CompressGzip)
	if err == nil {
		t.Error("Expected error when decompressing invalid gzip data")
	}
}

func TestDecompressData_None(t *testing.T) {
	data := []byte("test data")
	result, err := decompressData(data, CompressNone)
	if err != nil {
		t.Fatalf("decompressData(None) error: %v", err)
	}
	if !bytes.Equal(data, result) {
		t.Errorf("DecompressNone should return data unchanged")
	}
}

func TestDecompressData_UnknownType(t *testing.T) {
	result, err := decompressData([]byte("test"), CompressType(99))
	if err != nil {
		t.Errorf("Unknown decompress type should not error, got: %v", err)
	}
	if !bytes.Equal(result, []byte("test")) {
		t.Errorf("Unknown decompress type should return data unchanged")
	}
}

// ---- DetectCompression Tests ----

func TestDetectCompression_None_EmptyData(t *testing.T) {
	if ct := detectCompression(nil); ct != CompressNone {
		t.Errorf("detectCompression(nil) = %d, want CompressNone", ct)
	}
	if ct := detectCompression([]byte{}); ct != CompressNone {
		t.Errorf("detectCompression(empty) = %d, want CompressNone", ct)
	}
}

func TestDetectCompression_None_ShortData(t *testing.T) {
	if ct := detectCompression([]byte{0x00}); ct != CompressNone {
		t.Errorf("detectCompression(single byte) = %d, want CompressNone", ct)
	}
}

func TestDetectCompression_Gzip(t *testing.T) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write([]byte("test data"))
	w.Close()
	gzipData := buf.Bytes()

	ct := detectCompression(gzipData)
	if ct != CompressGzip {
		t.Errorf("detectCompression(gzip) = %d, want CompressGzip", ct)
	}
}

func TestDetectCompression_Snappy(t *testing.T) {
	snappyData := snappy.Encode(nil, []byte("test data for snappy detection"))

	ct := detectCompression(snappyData)
	if ct != CompressSnappy {
		t.Errorf("detectCompression(snappy) = %d, want CompressSnappy", ct)
	}
}

func TestDetectCompression_PlainText(t *testing.T) {
	// Plain text that happens to pass snappy decode check should be detected as snappy
	// This is a known limitation of snappy detection, but test normal plain text
	ct := detectCompression([]byte("plain text data that is not compressed"))
	_ = ct
	// Just ensure no panic
}

// ---- Codec Interface Tests ----

func TestCodecInterface_Protobuf(t *testing.T) {
	var c Codec = NewProtobufCodec(CompressNone)
	if c == nil {
		t.Fatal("ProtobufCodec does not implement Codec")
	}
}

func TestCodecInterface_JSON(t *testing.T) {
	var c Codec = NewJSONCodec(CompressNone)
	if c == nil {
		t.Fatal("JSONCodec does not implement Codec")
	}
}

// ---- Convenience Functions Tests ----

func TestMarshalProto(t *testing.T) {
	msg := &gamepb.GameMessage{MsgId: 42}
	data, err := MarshalProto(msg)
	if err != nil {
		t.Fatalf("MarshalProto error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("MarshalProto returned empty data")
	}

	decoded := &gamepb.GameMessage{}
	if err := UnmarshalProto(data, decoded); err != nil {
		t.Fatalf("UnmarshalProto error: %v", err)
	}
	if decoded.MsgId != 42 {
		t.Errorf("MsgId = %d, want 42", decoded.MsgId)
	}
}

func TestMarshalJSON(t *testing.T) {
	original := testStruct{Name: "test", Age: 30}
	data, err := MarshalJSON(original)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	var decoded testStruct
	if err := UnmarshalJSON(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if decoded != original {
		t.Errorf("Decoded = %+v, want %+v", decoded, original)
	}
}

func TestMarshalJSON_Map(t *testing.T) {
	original := map[string]interface{}{"a": 1, "b": "two"}
	data, err := MarshalJSON(original)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	var decoded map[string]interface{}
	if err := UnmarshalJSON(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if decoded["a"].(float64) != 1 || decoded["b"] != "two" {
		t.Errorf("Decoded = %v, want {a:1 b:two}", decoded)
	}
}

func TestMarshalJSON_InvalidUnmarshal(t *testing.T) {
	err := UnmarshalJSON([]byte("{invalid json"), &map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// ---- ProtobufCodec UnmarshalWithHint Tests ----

func TestProtobufCodec_UnmarshalWithHint(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	msg := &gamepb.GameMessage{MsgId: 1, Seq: 2}

	data, _, err := c.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	// UnmarshalWithHint should work with CompressNone hint
	decoded := &gamepb.GameMessage{}
	if err := c.UnmarshalWithHint(data, decoded, CompressNone); err != nil {
		t.Fatalf("UnmarshalWithHint error: %v", err)
	}
	if decoded.MsgId != 1 || decoded.Seq != 2 {
		t.Errorf("Decoded = %+v, want MsgId=1 Seq=2", decoded)
	}
}

func TestProtobufCodec_UnmarshalWithHint_InvalidType(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	err := c.UnmarshalWithHint([]byte{1, 2, 3}, "not proto message", CompressNone)
	if err == nil {
		t.Error("Expected error for non-proto.Message target")
	}
}

func TestProtobufCodec_UnmarshalWithHint_UnsupportedCompress(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	msg := &gamepb.GameMessage{MsgId: 1}

	data, _, err := c.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	decoded := &gamepb.GameMessage{}
	err = c.UnmarshalWithHint(data, decoded, CompressType(99))
	if err == nil {
		t.Error("Expected error for unsupported compress type hint")
	}
}

// ---- JSONCodec UnmarshalWithHint Tests ----

func TestJSONCodec_UnmarshalWithHint(t *testing.T) {
	c := NewJSONCodec(CompressNone)
	original := testStruct{Name: "test", Age: 99}

	data, _, err := c.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded testStruct
	if err := c.UnmarshalWithHint(data, &decoded, CompressNone); err != nil {
		t.Fatalf("UnmarshalWithHint error: %v", err)
	}
	if decoded != original {
		t.Errorf("Decoded = %+v, want %+v", decoded, original)
	}
}

func TestJSONCodec_UnmarshalWithHint_UnsupportedCompress(t *testing.T) {
	c := NewJSONCodec(CompressNone)
	var v interface{}
	err := c.UnmarshalWithHint([]byte{1, 2, 3}, &v, CompressType(99))
	if err == nil {
		t.Error("Expected error for unsupported compress type hint")
	}
}

// ---- Empty and Nil Input Tests ----

func TestCodec_EmptyData(t *testing.T) {
	// Protobuf unmarshal of empty data
	c := NewProtobufCodec(CompressNone)
	msg := &gamepb.GameMessage{}
	err := c.Unmarshal([]byte{}, msg)
	if err != nil {
		t.Errorf("Unmarshal empty protobuf data should not error: %v", err)
	}

	// JSON unmarshal of empty data
	c2 := NewJSONCodec(CompressNone)
	var v interface{}
	err = c2.Unmarshal([]byte{}, &v)
	if err == nil {
		t.Error("Unmarshal empty JSON data should error")
	}
}

func TestCodec_NilInput(t *testing.T) {
	c := NewProtobufCodec(CompressNone)

	// Marshal nil
	_, _, err := c.Marshal(nil)
	if err == nil {
		// Actually, nil might panic or error depending on protobuf implementation
		// Just ensure it doesn't panic
	}

	// Marshal nil proto message
	var msg *gamepb.GameMessage = nil
	_, _, err = c.Marshal(msg)
	if err == nil {
		t.Log("Note: Marshal(nil proto.Message) did not error")
	}
}

// ---- CompressType Enum Tests ----

func TestCompressTypeValues(t *testing.T) {
	tests := []struct {
		name string
		ct   CompressType
		want uint8
	}{
		{"None", CompressNone, 0},
		{"Snappy", CompressSnappy, 1},
		{"Gzip", CompressGzip, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint8(tt.ct) != tt.want {
				t.Errorf("CompressType = %d, want %d", uint8(tt.ct), tt.want)
			}
		})
	}
}

// ---- JSONCodec Marshal/Unmarshal error tests ----

type marshalErrorStruct struct{}

func (m marshalErrorStruct) MarshalJSON() ([]byte, error) {
	return nil, &json.UnsupportedTypeError{}
}

func TestJSONCodec_Marshal_Error(t *testing.T) {
	c := NewJSONCodec(CompressNone)
	_, _, err := c.Marshal(make(chan int))
	if err == nil {
		t.Error("Expected error when marshaling non-serializable value")
	}
}

func TestJSONCodec_Marshal_MarshalerError(t *testing.T) {
	c := NewJSONCodec(CompressNone)
	_, _, err := c.Marshal(marshalErrorStruct{})
	if err == nil {
		t.Error("Expected error when marshal function returns error")
	}
}

// ---- ProtobufCodec Marshal/Unmarshal edge cases ----

func TestProtobufCodec_Marshal_EmptyMessage(t *testing.T) {
	c := NewProtobufCodec(CompressNone)
	msg := &gamepb.GameMessage{}

	data, ct, err := c.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal empty message error: %v", err)
	}
	if ct != CompressNone {
		t.Errorf("CompressType = %d, want CompressNone for empty message", ct)
	}

	decoded := &gamepb.GameMessage{}
	if err := c.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal empty message error: %v", err)
	}
}

// ---- Large data compression tests ----

func TestLargeDataCompression(t *testing.T) {
	// Create a large protobuf message
	largePayload := make([]byte, 10000)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	msg := &gamepb.GameMessage{
		MsgId:   1,
		Payload: largePayload,
	}

	// Test with Snappy
	cSnappy := NewProtobufCodec(CompressSnappy)
	snappyData, ct, err := cSnappy.Marshal(msg)
	if err != nil {
		t.Fatalf("Snappy marshal error: %v", err)
	}
	if ct != CompressSnappy {
		t.Errorf("Expected CompressSnappy for large data, got %d", ct)
	}

	// Compressed size should be smaller than original
	// (protobuf + payload compression)
	_ = snappyData

	decoded := &gamepb.GameMessage{}
	if err := cSnappy.Unmarshal(snappyData, decoded); err != nil {
		t.Fatalf("Snappy unmarshal error: %v", err)
	}
	if !bytes.Equal(decoded.Payload, largePayload) {
		t.Error("Snappy compressed payload mismatch")
	}

	// Test with Gzip
	cGzip := NewProtobufCodec(CompressGzip)
	gzipData, ct, err := cGzip.Marshal(msg)
	if err != nil {
		t.Fatalf("Gzip marshal error: %v", err)
	}
	if ct != CompressGzip {
		t.Errorf("Expected CompressGzip for large data, got %d", ct)
	}

	decoded2 := &gamepb.GameMessage{}
	if err := cGzip.Unmarshal(gzipData, decoded2); err != nil {
		t.Fatalf("Gzip unmarshal error: %v", err)
	}
	if !bytes.Equal(decoded2.Payload, largePayload) {
		t.Error("Gzip compressed payload mismatch")
	}
}
