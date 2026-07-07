// Package idmix 实现 IDX 二进制编码与 idmix 可插拔文本编解码层。
//
// 两层可独立使用：
//  1. Idx（idx_codec.go）：整数/短字符串 → 自描述二进制块
//  2. Codec 接口 + EncodeBytes / DecodeString：任意二进制 ↔ 文本
//
// IdMix 组合 Idx 与 Codec；Encode 内部为 Idx 编码后调用 Codec.Encode。
//
// 完整协议说明见项目根目录 arithmetic.md。
package idmix

import (
	"errors"
	"math/rand"
)

// DefaultAlphabet 为默认 RadixCodec 使用的 62 进制字符表。
const DefaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// IdMix 组合 IDX 二进制编解码与文本 Codec。
type IdMix struct {
	idx   *Idx
	codec Codec
}

// Option 配置 IdMix 实例（Codec、Idx）。
type Option func(*IdMix) error

// WithCodec 设置二进制↔文本编解码器（RadixCodec、Base64Codec、自定义 Codec 等）。
func WithCodec(codec Codec) Option {
	return func(m *IdMix) error {
		if codec == nil {
			return errors.New("codec cannot be nil")
		}
		m.codec = codec
		return nil
	}
}

// WithAlphabet 使用指定字符表创建 RadixCodec 并设为 IdMix 的 Codec（便捷方法）。
func WithAlphabet(alphabet string) Option {
	return func(m *IdMix) error {
		rc, err := NewRadixCodec(alphabet)
		if err != nil {
			return err
		}
		m.codec = rc
		return nil
	}
}

// WithIdx 设置 IDX 编解码器（maxObjects、maxVariants、checkBits 等在 Idx 上配置）。
func WithIdx(idx *Idx) Option {
	return func(m *IdMix) error {
		if idx == nil {
			return errors.New("idx cannot be nil")
		}
		m.idx = idx
		return nil
	}
}

// New 创建 IdMix 实例（默认 Idx + 默认 RadixCodec）。
func New(opts ...Option) (*IdMix, error) {
	idx, err := NewIdx()
	if err != nil {
		return nil, err
	}
	m := &IdMix{
		idx:   idx,
		codec: defaultCodecInstance(),
	}
	for _, opt := range opts {
		if err := opt(m); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Idx 返回内嵌的 IDX 编解码器。
func (m *IdMix) Idx() *Idx {
	return m.idx
}

// Codec 返回当前实例使用的文本编解码器。
func (m *IdMix) Codec() Codec {
	return m.codec
}

// Encode 将多个整数或短字符串编码为文本（Idx 二进制 + Codec.Encode）。
func (m *IdMix) Encode(values ...any) (string, error) {
	if len(values) < 1 {
		return "", errors.New("at least one value is required")
	}
	variantID := rand.Intn(m.idx.maxVariants)
	data, err := m.encodeBinary(values, variantID)
	if err != nil {
		return "", err
	}
	return m.codec.Encode(data)
}

func (m *IdMix) encodeBinary(values []any, variantID int) ([]byte, error) {
	objects, err := normalizeObjects(values)
	if err != nil {
		return nil, err
	}
	return m.idx.encodeBinary(objects, variantID)
}

// Decode 将文本解码为 []any。
func (m *IdMix) Decode(s string) ([]any, error) {
	data, err := m.codec.Decode(s)
	if err != nil {
		return nil, err
	}
	return m.idx.Decode(data)
}

// EncodeWithVariant 确定性编码（指定 variant_id），主要用于测试。
func (m *IdMix) EncodeWithVariant(variantID int, values ...any) (string, error) {
	data, err := m.encodeBinary(values, variantID)
	if err != nil {
		return "", err
	}
	return m.codec.Encode(data)
}
