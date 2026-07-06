// xid_codec.go 实现 XID v1.1 二进制层编解码。
//
// 二进制块结构（小端序）：
//
//	[2 字节 header] + [数据对象序列]
//
// header 位域（从低到高）：check | count | variant_id
// 每个数据对象编码一个带类型的整数，支持内嵌模式（1 字节）和扩展模式（1+负载字节）。
// 对象序列在写入 header 前会经 variant_id 派生的 XOR 掩码混淆，解码时逆操作还原。
//
// 协议细节见 arithmetic.md 第 2~6 节。
package idmix

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// 原始类型索引（otype），写入扩展模式对象头的低 4 位。
const (
	otypeUint8  = 0
	otypeUint16 = 1
	otypeUint32 = 2
	otypeUint64 = 3
	otypeInt8   = 4
	otypeInt16  = 5
	otypeInt32  = 6
	otypeInt64  = 7
)

// swBytes 将扩展模式中的存储宽度索引 sw 映射为实际字节数：1/2/4/8。
var swBytes = [4]int{1, 2, 4, 8}

// embeddedOtype[sign][widthBits] 将内嵌模式对象头的 sign 与 width_bits 映射为 otype。
//
// sign=0 为无符号类型，sign=1 为有符号类型；
// widthBits：00=8位, 01=16位, 10=32位, 11=64位。
var embeddedOtype = [2][4]uint8{
	{otypeUint8, otypeUint16, otypeUint32, otypeUint64},
	{otypeInt8, otypeInt16, otypeInt32, otypeInt64},
}

// typedValue 是内部统一的整数表示：保留原始类型索引与 int64 数值。
// 所有编解码逻辑均在此类型上操作，与 Go 具体类型解耦。
type typedValue struct {
	otype uint8 // 原始类型索引 0~7
	val   int64 // 数值（无符号类型也以非负 int64 存储）
}

// materializeValues 将 typedValue 切片还原为 []any，恢复编码时的具体 Go 类型。
func materializeValues(typed []typedValue) ([]any, error) {
	out := make([]any, len(typed))
	for i, tv := range typed {
		v, err := materializeValue(tv)
		if err != nil {
			return nil, fmt.Errorf("value[%d]: %w", i, err)
		}
		out[i] = v
	}
	return out, nil
}

// materializeValue 将单个 typedValue 转为对应的 Go 具体类型。
func materializeValue(tv typedValue) (any, error) {
	switch tv.otype {
	case otypeUint8:
		return uint8(tv.val), nil
	case otypeUint16:
		return uint16(tv.val), nil
	case otypeUint32:
		return uint32(tv.val), nil
	case otypeUint64:
		return uint64(tv.val), nil
	case otypeInt8:
		return int8(tv.val), nil
	case otypeInt16:
		return int16(tv.val), nil
	case otypeInt32:
		return int32(tv.val), nil
	case otypeInt64:
		return tv.val, nil
	default:
		return nil, fmt.Errorf("invalid otype %d", tv.otype)
	}
}

// encodeBinary 将 typedValue 序列编码为 XID 二进制块。
//
// 流程：
//  1. 逐个 encodeObject 得到对象字节序列
//  2. 用 variant_id 派生 XOR 掩码混淆对象区（mask = (variant_id*0x9D + 0x37) & 0xFF）
//  3. 组装 2 字节小端 header（先填 count 和 variant_id，check 位暂为 0）
//  4. 对整个 data 做逐字节 XOR，取低 checkBits 位写入 header
func (m *IdMix) encodeBinary(typed []typedValue, variantID int) ([]byte, error) {
	objects := make([]byte, 0, len(typed)*2)
	for _, tv := range typed {
		obj, err := encodeObject(tv)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj...)
	}
	mask := byte((variantID*0x9D + 0x37) & 0xFF)
	for i := range objects {
		objects[i] ^= mask
	}

	count := len(typed)
	header := uint16(variantID)<<m.variantShift | uint16(count)<<m.countShift
	data := make([]byte, 2+len(objects))
	binary.LittleEndian.PutUint16(data, header)
	copy(data[2:], objects)

	xorSum := byte(0)
	for _, b := range data {
		xorSum ^= b
	}
	check := uint16(xorSum) & m.checkMask
	header |= check
	binary.LittleEndian.PutUint16(data, header)
	return data, nil
}

// decodeBinary 将 XID 二进制块解码为 typedValue 序列。
//
// 校验顺序：长度 → 解析 header → 变体/计数合法性 → XOR 校验和 → 解混淆 → 逐对象解码。
func (m *IdMix) decodeBinary(data []byte) ([]typedValue, error) {
	if len(data) < 2 {
		return nil, errors.New("invalid data: too short")
	}
	header := binary.LittleEndian.Uint16(data)
	check := header & m.checkMask
	count := int((header & m.countMask) >> m.countShift)
	variantID := int((header & m.variantMask) >> m.variantShift)

	if variantID >= m.maxVariants {
		return nil, fmt.Errorf("invalid variant_id %d (max %d)", variantID, m.maxVariants-1)
	}
	if count > m.maxObjects {
		return nil, fmt.Errorf("invalid count %d (max %d)", count, m.maxObjects)
	}

	// 校验：将 check 位清零后重算 XOR，与 header 中记录的 check 比较
	verify := make([]byte, len(data))
	copy(verify, data)
	verify[0] &= ^byte(m.checkMask)
	xorSum := byte(0)
	for _, b := range verify {
		xorSum ^= b
	}
	if uint16(xorSum)&m.checkMask != check {
		return nil, errors.New("checksum mismatch")
	}

	objects := make([]byte, len(data)-2)
	copy(objects, data[2:])
	mask := byte((variantID*0x9D + 0x37) & 0xFF)
	for i := range objects {
		objects[i] ^= mask
	}

	result := make([]typedValue, 0, count)
	pos := 0
	for i := 0; i < count; i++ {
		if pos >= len(objects) {
			return nil, errors.New("premature end of data")
		}
		tv, n, err := decodeObject(objects[pos:])
		if err != nil {
			return nil, fmt.Errorf("object[%d]: %w", i, err)
		}
		result = append(result, tv)
		pos += n
	}
	if pos != len(objects) {
		return nil, errors.New("extra bytes after data objects")
	}
	return result, nil
}

// encodeObject 将单个 typedValue 编码为一个数据对象字节序列。
//
// 编码优先级（见 arithmetic.md 第 2.2 节）：
//  1. 内嵌模式（1 字节）：正数表示 P < 17（[-16,16]，+16 除外）
//  2. 扩展模式（1+负载）：正数表示 P 按数值大小选最小 sw，bit6 存符号
func encodeObject(tv typedValue) ([]byte, error) {
	if err := validateRange(tv.otype, tv.val); err != nil {
		return nil, err
	}
	if head, ok := tryEmbeddedHead(tv.otype, tv.val); ok {
		return []byte{head}, nil
	}
	mag, neg := magnitudeFromTyped(tv)
	sw := swFromMagnitude(mag)
	payload := uintToLEBytes(mag, swBytes[sw])
	head := byte(0x80) | byte(sw<<4) | tv.otype
	if neg {
		head |= 1 << 6
	}
	out := make([]byte, 1+len(payload))
	out[0] = head
	copy(out[1:], payload)
	return out, nil
}

// decodeObject 从字节流中解码一个数据对象。
//
// 返回：typedValue、消耗的字节数、错误。
// 根据对象头 bit7 区分内嵌模式（bit7=0）与扩展模式（bit7=1）。
func decodeObject(data []byte) (typedValue, int, error) {
	if len(data) < 1 {
		return typedValue{}, 0, errors.New("truncated object header")
	}
	head := data[0]
	if head&0x80 == 0 {
		// 内嵌模式：bit6=sign, bit5-4=width_bits, bit3-0=value
		sign := (head >> 6) & 1
		wb := (head >> 4) & 0x03
		v := head & 0x0F
		otype := embeddedOtype[sign][wb]
		var val int64
		if sign == 0 {
			val = int64(v)
		} else {
			val = -int64(v) - 1
		}
		return typedValue{otype, val}, 1, nil
	}
	// 扩展模式：bit7=1，bit6=sign，bit5-4=sw，bit3-0=otype
	sw := (head >> 4) & 0x03
	otype := head & 0x0F
	if otype > otypeInt64 {
		return typedValue{}, 0, fmt.Errorf("invalid otype %d", otype)
	}
	numBytes := swBytes[sw]
	if len(data) < 1+numBytes {
		return typedValue{}, 0, errors.New("truncated object payload")
	}
	mag := uint64(0)
	for i := 0; i < numBytes; i++ {
		mag |= uint64(data[1+i]) << (8 * i)
	}
	neg := (head>>6)&1 != 0
	val := valueFromMagnitude(otype, mag, neg)
	if err := validateRange(otype, val); err != nil {
		return typedValue{}, 0, err
	}
	return typedValue{otype, val}, 1 + numBytes, nil
}

// isUnsigned 判断 otype 是否为无符号类型（0~3）。
func isUnsigned(otype uint8) bool { return otype <= otypeUint64 }

// isSigned 判断 otype 是否为有符号类型（4~7）。
func isSigned(otype uint8) bool { return otype >= otypeInt8 }

// widthBits 将 otype 映射为内嵌模式中的 width_bits 字段（00/01/10/11）。
func widthBits(otype uint8) uint8 {
	switch otype {
	case otypeUint8, otypeInt8:
		return 0
	case otypeUint16, otypeInt16:
		return 1
	case otypeUint32, otypeInt32:
		return 2
	default:
		return 3
	}
}

// magnitudeFromTyped 返回正数表示 P 与原值是否为负。
func magnitudeFromTyped(tv typedValue) (mag uint64, neg bool) {
	if isUnsigned(tv.otype) {
		return uint64(tv.val), false
	}
	if tv.val < 0 {
		return uint64(-tv.val), true
	}
	return uint64(tv.val), false
}

// swFromMagnitude 按 P 的数值大小选取最小 sw（与 otype 位宽无关）。
func swFromMagnitude(mag uint64) uint8 {
	if mag < 256 {
		return 0
	}
	if mag < 65536 {
		return 1
	}
	if mag < 4294967296 {
		return 2
	}
	return 3
}

// tryEmbeddedHead 在 P < 17 时尝试 1 字节内嵌编码。
func tryEmbeddedHead(otype uint8, val int64) (byte, bool) {
	mag, neg := magnitudeFromTyped(typedValue{otype, val})
	if mag >= 17 {
		return 0, false
	}
	wb := widthBits(otype)
	if mag == 16 {
		if neg {
			return byte(1<<6) | byte(wb<<4) | 15, true
		}
		return 0, false
	}
	if neg {
		return byte(1<<6) | byte(wb<<4) | byte(mag-1), true
	}
	return byte(wb<<4) | byte(mag), true
}

// valueFromMagnitude 从正数表示 P 与 sign 还原原值。
func valueFromMagnitude(otype uint8, mag uint64, neg bool) int64 {
	if !neg {
		return int64(mag)
	}
	if mag == 1<<63 {
		return math.MinInt64
	}
	return -int64(mag)
}

// uintToLEBytes 将无符号整数按小端序写入指定长度的字节切片。
func uintToLEBytes(v uint64, size int) []byte {
	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = byte(v >> (8 * i))
	}
	return buf
}

// validateRange 校验数值是否在 otype 对应类型的合法范围内。
func validateRange(otype uint8, val int64) error {
	switch otype {
	case otypeUint8:
		if val < 0 || val > math.MaxUint8 {
			return fmt.Errorf("value %d out of uint8 range", val)
		}
	case otypeUint16:
		if val < 0 || val > math.MaxUint16 {
			return fmt.Errorf("value %d out of uint16 range", val)
		}
	case otypeUint32:
		if val < 0 || val > math.MaxUint32 {
			return fmt.Errorf("value %d out of uint32 range", val)
		}
	case otypeUint64:
		// uint64 全范围：内部以 int64 位模式存储
	case otypeInt8:
		if val < math.MinInt8 || val > math.MaxInt8 {
			return fmt.Errorf("value %d out of int8 range", val)
		}
	case otypeInt16:
		if val < math.MinInt16 || val > math.MaxInt16 {
			return fmt.Errorf("value %d out of int16 range", val)
		}
	case otypeInt32:
		if val < math.MinInt32 || val > math.MaxInt32 {
			return fmt.Errorf("value %d out of int32 range", val)
		}
	case otypeInt64:
		// int64 全范围合法
	default:
		return fmt.Errorf("invalid otype %d", otype)
	}
	return nil
}
