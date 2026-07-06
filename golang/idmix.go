// Package idmix 实现 XID v1.1 规范：将多个带类型的整数序列编码为短字符串。
//
// 编码流程分两层：
//  1. 二进制层（xid_codec.go）：整数 → 自描述二进制块（含校验位、变体混淆）
//  2. 文本层（alphabet.go）：二进制块 → 自定义进制字符串
//
// 典型用法：
//
//	m, err := idmix.New()
//	str, err := m.Encode(uint16(5), int64(-1), uint32(40))
//	list, err := m.Decode(str)
//	v := list[0].(uint16)
//
// 完整协议说明见项目根目录 arithmetic.md。
package idmix

import (
	"errors"
	"fmt"
	"math/bits"
	"math/rand"
)

// DefaultAlphabet 为默认 62 进制字符表（a-z, A-Z, 0-9）。
// 字符表长度即进制基数；顺序可自定义，相当于一种轻量"密钥"。
const DefaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// 默认布局参数，对应 arithmetic.md 第 2.1 节推荐配置。
const (
	defaultMaxObjects  = 511 // count 域最多 9 位，上限 511 个对象
	defaultMaxVariants = 32  // variant_id 取值 0~31，产生 32 种不同字符串
	defaultCheckBits   = 2   // 全局 XOR 校验的低 2 位写入 header
)

// IdMix 是 XID v1.1 编解码器的主入口类型。
//
// 16 位 header 的位域在 New 时由 finalizeLayout 根据 maxObjects、maxVariants、checkBits 动态计算，
// 三者之和不得超过 16 位。
type IdMix struct {
	radix        *radixCodec // 自定义进制编解码器
	maxObjects   int         // 单次 Encode 允许的最大整数个数
	maxVariants  int         // 变体数，控制多态混淆（同一输入可产生多种字符串）
	checkBits    int         // 校验位宽度
	countBits    int         // header 中对象计数字段的位宽（由 maxObjects 推导）
	variantBits  int         // header 中变体 ID 字段的位宽（由 maxVariants 推导）
	checkMask    uint16      // 校验位掩码，位于 header 最低位
	countMask    uint16      // 计数字段掩码，紧随校验位之后
	variantMask  uint16      // 变体字段掩码，位于 header 最高位
	countShift   uint8       // 计数字段在 header 中的左移量
	variantShift uint8       // 变体字段在 header 中的左移量
}

// Option 是函数式配置项，在 New 时依次应用到 IdMix 实例。
type Option func(*IdMix) error

// WithAlphabet 设置自定义进制字符表。
//
// 要求：长度 ≥ 2（即进制 ≥ 2），字符不可重复。
// 例如 "abcd" 为四进制，"0123456789abc" 为十一进制，"一二三四五六七八九十" 为十进制。
func WithAlphabet(alphabet string) Option {
	return func(m *IdMix) error {
		rc, err := newRadixCodec(alphabet)
		if err != nil {
			return err
		}
		m.radix = rc
		return nil
	}
}

// WithMaxObjects 设置单次编码允许的最大对象个数（默认 511）。
// 该值决定 header 中 count 域的位宽；越大则 count 占用的位数越多。
func WithMaxObjects(n int) Option {
	return func(m *IdMix) error {
		if n < 1 {
			return errors.New("maxObjects must be at least 1")
		}
		m.maxObjects = n
		return nil
	}
}

// WithMaxVariants 设置变体数（默认 32，即 variant_id 取值 0~31）。
// 每次 Encode 会随机选取一个变体，使相同输入产生不同字符串，用于防猜或规避敏感词。
func WithMaxVariants(n int) Option {
	return func(m *IdMix) error {
		if n < 1 {
			return errors.New("maxVariants must be at least 1")
		}
		m.maxVariants = n
		return nil
	}
}

// WithCheckBits 设置校验位宽度（默认 2，有效范围 1~8）。
// 校验值为整个二进制块（含 header）逐字节 XOR 后的低 checkBits 位。
func WithCheckBits(n int) Option {
	return func(m *IdMix) error {
		if n < 1 || n > 8 {
			return errors.New("checkBits must be between 1 and 8")
		}
		m.checkBits = n
		return nil
	}
}

// New 创建并初始化 IdMix 实例。
//
// 应用所有 Option 后调用 finalizeLayout 计算 header 位域布局；
// 若 checkBits + countBits + variantBits > 16，返回错误。
func New(opts ...Option) (*IdMix, error) {
	rc, err := newRadixCodec(DefaultAlphabet)
	if err != nil {
		return nil, err
	}
	m := &IdMix{
		radix:       rc,
		maxObjects:  defaultMaxObjects,
		maxVariants: defaultMaxVariants,
		checkBits:   defaultCheckBits,
	}
	for _, opt := range opts {
		if err := opt(m); err != nil {
			return nil, err
		}
	}
	if err := m.finalizeLayout(); err != nil {
		return nil, err
	}
	return m, nil
}

// finalizeLayout 根据 maxObjects、maxVariants、checkBits 计算 16 位 header 的位域分配。
//
// 布局（从低位到高位）：check | count | variant_id
// 使用 bits.Len 推导 countBits 和 variantBits，确保能表示 [0, max-1] 的全部取值。
func (m *IdMix) finalizeLayout() error {
	variantBits := bits.Len(uint(m.maxVariants - 1))
	if m.maxVariants <= 1 {
		variantBits = 1
	}
	countBits := bits.Len(uint(m.maxObjects))
	if m.maxObjects <= 1 {
		countBits = 1
	}
	total := m.checkBits + countBits + variantBits
	if total > 16 {
		return fmt.Errorf(
			"checkBits(%d) + countBits(%d) + variantBits(%d) = %d exceeds 16-bit header",
			m.checkBits, countBits, variantBits, total,
		)
	}
	m.countBits = countBits
	m.variantBits = variantBits
	m.checkMask = uint16((1 << m.checkBits) - 1)
	m.countMask = uint16((1<<countBits)-1) << m.checkBits
	m.variantMask = uint16((1<<variantBits)-1) << (m.checkBits + countBits)
	m.countShift = uint8(m.checkBits)
	m.variantShift = uint8(m.checkBits + countBits)
	return nil
}

// Encode 将多个整数编码为 XID 字符串，并保留各值的原始 Go 类型信息。
//
// 支持的类型：uint8/16/32/64、int8/16/32/64、int、uint（uint64 全范围，> MaxInt64 时以 int64 位模式存储）。
// 编码时随机选取 variant_id，因此同一组输入多次调用可能产生不同字符串，但均可正确解码。
//
// 示例：
//
//	str, err := idObj.Encode(uint16(5), int64(-1), uint32(40), 123)
func (m *IdMix) Encode(values ...any) (string, error) {
	if len(values) < 1 {
		return "", errors.New("at least one value is required")
	}
	if len(values) > m.maxObjects {
		return "", fmt.Errorf("too many objects: %d (max %d)", len(values), m.maxObjects)
	}
	typed, err := normalizeAny(values)
	if err != nil {
		return "", err
	}
	variantID := rand.Intn(m.maxVariants)
	data, err := m.encodeBinary(typed, variantID)
	if err != nil {
		return "", err
	}
	return m.radix.encodeBytes(data)
}

// Decode 将 XID 字符串解码为 []any，元素类型与编码时一致。
//
// 调用方需自行做类型断言，例如 list[0].(uint16)。
// 解码失败时可能因：非法字符、校验和不匹配、变体越界、对象格式错误等。
func (m *IdMix) Decode(s string) ([]any, error) {
	data, err := m.radix.decodeBytes(s)
	if err != nil {
		return nil, err
	}
	typed, err := m.decodeBinary(data)
	if err != nil {
		return nil, err
	}
	return materializeValues(typed)
}
