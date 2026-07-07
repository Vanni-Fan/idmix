// alphabet.go 实现基于自定义字符表的 RadixCodec（默认 idmix 文本层）。
//
// 编码策略：在原始数据前附加 2 字节大端长度前缀，整体视为一个大整数，
// 再按自定义字符表做进制转换（类似无填充的 Base-N）。
package idmix

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)

var errNilCodecFunc = errors.New("codec function is nil")

// RadixCodec 使用自定义字符表（Base-N）的二进制↔文本编解码器。
type RadixCodec struct {
	base       int
	chars      []rune
	fromCustom map[rune]int
}

// NewRadixCodec 根据字符表创建 RadixCodec。
func NewRadixCodec(alphabet string) (*RadixCodec, error) {
	runes := []rune(alphabet)
	if len(runes) < 2 {
		return nil, errors.New("alphabet must have at least 2 unique characters")
	}
	rc := &RadixCodec{
		base:       len(runes),
		chars:      runes,
		fromCustom: make(map[rune]int, len(runes)),
	}
	for i, r := range runes {
		if _, ok := rc.fromCustom[r]; ok {
			return nil, fmt.Errorf("alphabet contains duplicate character %q", r)
		}
		rc.fromCustom[r] = i
	}
	return rc, nil
}

// Alphabet 返回字符表字符串。
func (rc *RadixCodec) Alphabet() string {
	return string(rc.chars)
}

// Base 返回进制基数。
func (rc *RadixCodec) Base() int {
	return rc.base
}

func (rc *RadixCodec) Encode(data []byte) (string, error) {
	if len(data) == 0 {
		return string(rc.chars[0]), nil
	}
	wrapped := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(wrapped, uint16(len(data)))
	copy(wrapped[2:], data)
	n := new(big.Int).SetBytes(wrapped)
	return rc.intToString(n), nil
}

func (rc *RadixCodec) Decode(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("empty string")
	}
	n, err := rc.stringToInt(s)
	if err != nil {
		return nil, err
	}
	raw := n.Bytes()
	for pad := 0; pad <= 1; pad++ {
		buf := make([]byte, pad+len(raw))
		copy(buf[pad:], raw)
		if len(buf) < 2 {
			continue
		}
		dataLen := int(binary.BigEndian.Uint16(buf[:2]))
		if len(buf) != 2+dataLen {
			continue
		}
		return buf[2:], nil
	}
	return nil, errors.New("invalid encoded data length")
}

func (rc *RadixCodec) intToString(n *big.Int) string {
	if n.Sign() == 0 {
		return string(rc.chars[0])
	}
	base := big.NewInt(int64(rc.base))
	zero := big.NewInt(0)
	rem := new(big.Int)
	chars := make([]rune, 0, 32)
	for n.Cmp(zero) > 0 {
		n.DivMod(n, base, rem)
		chars = append(chars, rc.chars[rem.Int64()])
	}
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

func (rc *RadixCodec) stringToInt(s string) (*big.Int, error) {
	n := big.NewInt(0)
	base := big.NewInt(int64(rc.base))
	for _, r := range s {
		idx, ok := rc.fromCustom[r]
		if !ok {
			return nil, fmt.Errorf("invalid character %q", r)
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(idx)))
	}
	return n, nil
}
