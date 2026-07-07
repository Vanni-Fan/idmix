// codec.go 定义 idmix 文本层的可插拔编解码接口及内置实现。
package idmix

import (
	"encoding/base64"
	"sync"
)

// Codec 二进制与文本之间的编解码器（idmix 文本层插拔点）。
//
// 可实现自定义字符表进制、Base64、AES+Base64、异或+Base64 等任意方案。
type Codec interface {
	Encode(data []byte) (string, error)
	Decode(s string) ([]byte, error)
}

// FuncCodec 由函数实现的 Codec，便于包装 AES/XOR 等自定义逻辑。
type FuncCodec struct {
	EncodeFn func(data []byte) (string, error)
	DecodeFn func(s string) ([]byte, error)
}

func (f FuncCodec) Encode(data []byte) (string, error) {
	if f.EncodeFn == nil {
		return "", errNilCodecFunc
	}
	return f.EncodeFn(data)
}

func (f FuncCodec) Decode(s string) ([]byte, error) {
	if f.DecodeFn == nil {
		return nil, errNilCodecFunc
	}
	return f.DecodeFn(s)
}

// Base64Codec 使用标准 Base64 的二进制↔文本编解码器。
type Base64Codec struct{}

func NewBase64Codec() Base64Codec { return Base64Codec{} }

func (Base64Codec) Encode(data []byte) (string, error) {
	return base64.StdEncoding.EncodeToString(data), nil
}

func (Base64Codec) Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

var (
	defaultCodec     Codec
	defaultCodecOnce sync.Once
)

func defaultCodecInstance() Codec {
	defaultCodecOnce.Do(func() {
		c, err := NewRadixCodec(DefaultAlphabet)
		if err != nil {
			panic(err)
		}
		defaultCodec = c
	})
	return defaultCodec
}

func resolveCodec(codec ...Codec) Codec {
	if len(codec) > 0 && codec[0] != nil {
		return codec[0]
	}
	return defaultCodecInstance()
}

// EncodeBytes 将任意二进制编码为文本；codec 不传时使用默认 RadixCodec。
func EncodeBytes(data []byte, codec ...Codec) (string, error) {
	return resolveCodec(codec...).Encode(data)
}

// DecodeString 将文本还原为二进制；codec 不传时使用默认 RadixCodec。
func DecodeString(s string, codec ...Codec) ([]byte, error) {
	return resolveCodec(codec...).Decode(s)
}
