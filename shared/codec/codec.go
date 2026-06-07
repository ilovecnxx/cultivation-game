// Package codec 提供消息编解码器，支持 Protobuf 主编码和 JSON 降级编码，
// 并支持 Snappy/Gzip 压缩。
package codec

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/snappy"
	"google.golang.org/protobuf/proto"
)

// CompressType 压缩算法类型。
type CompressType uint8

const (
	CompressNone CompressType = 0 // 不压缩
	CompressSnappy CompressType = 1 // Snappy 压缩（速度快）
	CompressGzip   CompressType = 2 // Gzip 压缩（压缩率高）
)

// Codec 编解码器接口，定义序列化和反序列化方法。
type Codec interface {
	// Marshal 将消息编码为字节切片，返回 (编码后数据, 使用的压缩类型, 错误)
	Marshal(v interface{}) ([]byte, CompressType, error)
	// Unmarshal 将字节切片解码为消息
	Unmarshal(data []byte, v interface{}) error
	// UnmarshalWithHint 用指定的压缩类型解码（跳过检测步骤）
	UnmarshalWithHint(data []byte, v interface{}, hint CompressType) error
	Name() string
}

// ---- ProtobufCodec ----

// ProtobufCodec 基于 Protobuf 的主编解码器。
type ProtobufCodec struct {
	compress CompressType // 使用的压缩算法
}

// NewProtobufCodec 创建 Protobuf 编解码器，指定压缩方式。
func NewProtobufCodec(ct CompressType) *ProtobufCodec {
	return &ProtobufCodec{compress: ct}
}

// Marshal 编码并压缩。
func (c *ProtobufCodec) Marshal(v interface{}) ([]byte, CompressType, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, CompressNone, fmt.Errorf("codec: ProtobufCodec 要求 proto.Message 类型, 得到 %T", v)
	}
	raw, err := proto.Marshal(msg)
	if err != nil {
		return nil, CompressNone, fmt.Errorf("codec: proto.Marshal 失败: %w", err)
	}
	if c.compress == CompressNone || len(raw) < 64 {
		// 数据太小时不压缩
		return raw, CompressNone, nil
	}
	compressed, err := compressData(raw, c.compress)
	if err != nil {
		return nil, CompressNone, err
	}
	return compressed, c.compress, nil
}

// Unmarshal 解码（自动检测压缩类型）。
func (c *ProtobufCodec) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("codec: ProtobufCodec 要求 proto.Message 类型, 得到 %T", v)
	}
	ct := detectCompression(data)
	return c.unmarshal(data, msg, ct)
}

// UnmarshalWithHint 使用指定的压缩类型解码。
func (c *ProtobufCodec) UnmarshalWithHint(data []byte, v interface{}, hint CompressType) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("codec: ProtobufCodec 要求 proto.Message 类型, 得到 %T", v)
	}
	return c.unmarshal(data, msg, hint)
}

func (c *ProtobufCodec) unmarshal(data []byte, msg proto.Message, ct CompressType) error {
	var raw []byte
	var err error
	switch ct {
	case CompressNone:
		raw = data
	case CompressSnappy, CompressGzip:
		raw, err = decompressData(data, ct)
		if err != nil {
			return fmt.Errorf("codec: 解压失败: %w", err)
		}
	default:
		return fmt.Errorf("codec: 不支持的压缩类型 %d", ct)
	}
	if err := proto.Unmarshal(raw, msg); err != nil {
		return fmt.Errorf("codec: proto.Unmarshal 失败: %w", err)
	}
	return nil
}

func (c *ProtobufCodec) Name() string { return "protobuf" }

// ---- JSONCodec ----

// JSONCodec 基于 JSON 的降级编解码器，消息需实现 json.Marshaler/Unmarshaler。
type JSONCodec struct {
	compress   CompressType
	pretty     bool // 是否格式化输出（仅调试用）
}

// NewJSONCodec 创建 JSON 编解码器。
func NewJSONCodec(ct CompressType) *JSONCodec {
	return &JSONCodec{compress: ct}
}

// NewJSONCodecPretty 创建带格式化的 JSON 编解码器（调试用途）。
func NewJSONCodecPretty(ct CompressType) *JSONCodec {
	return &JSONCodec{compress: ct, pretty: true}
}

// Marshal 将任意对象编码为 JSON，可选压缩。
func (c *JSONCodec) Marshal(v interface{}) ([]byte, CompressType, error) {
	var raw []byte
	var err error
	if c.pretty {
		raw, err = json.MarshalIndent(v, "", "  ")
	} else {
		raw, err = json.Marshal(v)
	}
	if err != nil {
		return nil, CompressNone, fmt.Errorf("codec: json.Marshal 失败: %w", err)
	}
	if c.compress == CompressNone || len(raw) < 64 {
		return raw, CompressNone, nil
	}
	compressed, err := compressData(raw, c.compress)
	if err != nil {
		return nil, CompressNone, err
	}
	return compressed, c.compress, nil
}

// Unmarshal 自动检测压缩并解码。
func (c *JSONCodec) Unmarshal(data []byte, v interface{}) error {
	ct := detectCompression(data)
	return c.unmarshal(data, v, ct)
}

// UnmarshalWithHint 使用指定压缩类型解码。
func (c *JSONCodec) UnmarshalWithHint(data []byte, v interface{}, hint CompressType) error {
	return c.unmarshal(data, v, hint)
}

func (c *JSONCodec) unmarshal(data []byte, v interface{}, ct CompressType) error {
	var raw []byte
	var err error
	switch ct {
	case CompressNone:
		raw = data
	case CompressSnappy, CompressGzip:
		raw, err = decompressData(data, ct)
		if err != nil {
			return fmt.Errorf("codec: 解压失败: %w", err)
		}
	default:
		return fmt.Errorf("codec: 不支持的压缩类型 %d", ct)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("codec: json.Unmarshal 失败: %w", err)
	}
	return nil
}

func (c *JSONCodec) Name() string { return "json" }

// ---- 压缩工具函数 ----

// compressData 根据压缩类型压缩数据。
func compressData(data []byte, ct CompressType) ([]byte, error) {
	switch ct {
	case CompressSnappy:
		return snappy.Encode(nil, data), nil
	case CompressGzip:
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("codec: gzip 写入失败: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("codec: gzip 关闭失败: %w", err)
		}
		return buf.Bytes(), nil
	default:
		return data, nil
	}
}

// decompressData 根据压缩类型解压数据。
func decompressData(data []byte, ct CompressType) ([]byte, error) {
	switch ct {
	case CompressSnappy:
		decoded, err := snappy.Decode(nil, data)
		if err != nil {
			return nil, fmt.Errorf("codec: snappy 解码失败: %w", err)
		}
		return decoded, nil
	case CompressGzip:
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("codec: gzip 读取器创建失败: %w", err)
		}
		defer r.Close()
		decoded, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("codec: gzip 读取失败: %w", err)
		}
		return decoded, nil
	default:
		return data, nil
	}
}

// detectCompression 通过 Magic Bytes 检测数据是否被压缩。
// Snappy 格式没有固定魔数，这里通过尝试解码来判断。
// Gzip 的魔数是 0x1F, 0x8B。
func detectCompression(data []byte) CompressType {
	if len(data) < 2 {
		return CompressNone
	}
	// Gzip 魔数
	if data[0] == 0x1F && data[1] == 0x8B {
		return CompressGzip
	}
	// Snappy 块格式通常以非标准字节开头，尝试解码看是否合法
	if len(data) > 4 {
		_, err := snappy.Decode(nil, data)
		if err == nil {
			return CompressSnappy
		}
	}
	return CompressNone
}

// ---- 便捷函数 ----

// MarshalProto 便捷函数：用 ProtobufCodec 编码（不压缩）。
func MarshalProto(v proto.Message) ([]byte, error) {
	c := NewProtobufCodec(CompressNone)
	data, _, err := c.Marshal(v)
	return data, err
}

// UnmarshalProto 便捷函数：用 ProtobufCodec 解码（自动检测压缩）。
func UnmarshalProto(data []byte, v proto.Message) error {
	c := NewProtobufCodec(CompressNone)
	return c.Unmarshal(data, v)
}

// MarshalJSON 便捷函数：用 JSONCodec 编码（不压缩）。
func MarshalJSON(v interface{}) ([]byte, error) {
	c := NewJSONCodec(CompressNone)
	data, _, err := c.Marshal(v)
	return data, err
}

// UnmarshalJSON 便捷函数：用 JSONCodec 解码（自动检测压缩）。
func UnmarshalJSON(data []byte, v interface{}) error {
	c := NewJSONCodec(CompressNone)
	return c.Unmarshal(data, v)
}
