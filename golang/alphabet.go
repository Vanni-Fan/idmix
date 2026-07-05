// alphabet.go 实现 XID 的文本层：将二进制块编码为自定义进制字符串，反之亦然。
//
// 编码策略：在原始数据前附加 2 字节大端长度前缀，整体视为一个大整数，
// 再按自定义字符表做进制转换（类似无填充的 Base-N）。
// 解码时通过尝试 0/1 字节前导填充来恢复长度前缀，消除 big.Int 去前导零的影响。
package idmix

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)

// radixCodec 封装自定义进制（Base-N）编解码逻辑。
type radixCodec struct {
	base       int            // 进制基数，等于字符表长度
	chars      []rune         // 字符表，索引 i 对应数字 i
	fromCustom map[rune]int   // 字符 → 数字索引，用于解码查表
}

// newRadixCodec 根据字符表创建进制编解码器。
//
// 校验规则：
//   - 至少 2 个字符（进制 ≥ 2）
//   - 字符不可重复
func newRadixCodec(alphabet string) (*radixCodec, error) {
	runes := []rune(alphabet)
	if len(runes) < 2 {
		return nil, errors.New("alphabet must have at least 2 unique characters")
	}
	rc := &radixCodec{
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

// encodeBytes 将二进制块编码为自定义进制字符串。
//
// 步骤：
//  1. 构造 [2字节大端长度 | 原始数据] 的包装缓冲区
//  2. 将包装缓冲区视为无符号大整数
//  3. 反复除基取余，映射到 chars 表，得到最终字符串
//
// 空输入编码为字符表第一个字符（表示数值 0）。
func (rc *radixCodec) encodeBytes(data []byte) (string, error) {
	if len(data) == 0 {
		return string(rc.chars[0]), nil
	}
	wrapped := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(wrapped, uint16(len(data)))
	copy(wrapped[2:], data)
	n := new(big.Int).SetBytes(wrapped)
	return rc.intToString(n), nil
}

// decodeBytes 将自定义进制字符串还原为原始二进制块。
//
// 由于 big.Int 会去掉前导零，解码时需尝试 0 或 1 字节的前导填充，
// 使得 [长度前缀(2B) | 数据] 的总长度与长度字段一致。
func (rc *radixCodec) decodeBytes(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("empty string")
	}
	n, err := rc.stringToInt(s)
	if err != nil {
		return nil, err
	}
	raw := n.Bytes()
	// pad=0：无前导零；pad=1：补 1 字节前导零（应对长度前缀高字节为 0 的情况）
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

// intToString 将大整数按自定义进制转为字符串（高位在前）。
func (rc *radixCodec) intToString(n *big.Int) string {
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
	// 余数序列是低位在前，需反转为高位在前
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

// stringToInt 将自定义进制字符串解析为大整数。
func (rc *radixCodec) stringToInt(s string) (*big.Int, error) {
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
