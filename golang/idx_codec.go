// idx_codec.go 实现 IDX 二进制层编解码（自描述整数/短字符串序列）。
//
// 二进制块结构：
//
//	[1 或 2 字节 header] + [数据对象序列]
//
// 单对象时 header 仅 1 字节；多对象时追加第 2 字节存放 count。
// 对象序列经 variant_id 派生的 XOR 掩码混淆，解码时逆操作还原。
//
// 协议细节见 arithmetic.md。
package idmix

import (
	"errors"
	"fmt"
	"math"
)

const (
	maxStringLen = 63 // 扩展模式 bit5-0 最多表示 63 字节字符串

	otypeUint8  = 0
	otypeUint16 = 1
	otypeUint32 = 2
	otypeUint64 = 3
	otypeInt8   = 4
	otypeInt16  = 5
	otypeInt32  = 6
	otypeInt64  = 7
)

var swBytes = [4]int{1, 2, 4, 8}

var embeddedOtype = [2][4]uint8{
	{otypeUint8, otypeUint16, otypeUint32, otypeUint64},
	{otypeInt8, otypeInt16, otypeInt32, otypeInt64},
}

// dataObject 是内部统一的数据对象表示：整数或短字符串。
type dataObject struct {
	isString bool
	otype    uint8
	val      int64
	str      []byte
}

// Idx 是 IDX 二进制编解码器，可独立于 idmix 文本层使用。
type Idx struct {
	maxObjects  int
	maxVariants int
	checkBits   int
	checkMask   uint8
}

// IdxOption 配置 Idx 实例。
type IdxOption func(*Idx) error

// WithMaxObjects 设置单次编码允许的最大对象个数（默认 255）。
func WithMaxObjects(n int) IdxOption {
	return func(idx *Idx) error {
		if n < 1 || n > 255 {
			return errors.New("maxObjects must be between 1 and 255")
		}
		idx.maxObjects = n
		return nil
	}
}

// WithMaxVariants 设置变体数（默认 32，variant_id 0~31）。
func WithMaxVariants(n int) IdxOption {
	return func(idx *Idx) error {
		if n < 1 || n > 32 {
			return errors.New("maxVariants must be between 1 and 32")
		}
		idx.maxVariants = n
		return nil
	}
}

// WithCheckBits 设置校验位宽度（默认 2，有效 1~2，受 1 字节 header 限制）。
func WithCheckBits(n int) IdxOption {
	return func(idx *Idx) error {
		if n < 1 || n > 2 {
			return errors.New("checkBits must be 1 or 2")
		}
		idx.checkBits = n
		idx.checkMask = uint8((1 << n) - 1)
		return nil
	}
}

// NewIdx 创建 IDX 编解码器。
func NewIdx(opts ...IdxOption) (*Idx, error) {
	idx := &Idx{
		maxObjects:  255,
		maxVariants: 32,
		checkBits:   2,
		checkMask:   0x03,
	}
	for _, opt := range opts {
		if err := opt(idx); err != nil {
			return nil, err
		}
	}
	return idx, nil
}

// Encode 将多个整数或短字符串（≤63 字节）编码为 IDX 二进制块。
func (idx *Idx) Encode(values ...any) ([]byte, error) {
	if len(values) < 1 {
		return nil, errors.New("at least one value is required")
	}
	if len(values) > idx.maxObjects {
		return nil, fmt.Errorf("too many objects: %d (max %d)", len(values), idx.maxObjects)
	}
	objects, err := normalizeObjects(values)
	if err != nil {
		return nil, err
	}
	return idx.encodeBinary(objects, 0)
}

// EncodeWithVariant 与 Encode 相同，但指定 variant_id（用于测试或确定性编码）。
func (idx *Idx) EncodeWithVariant(variantID int, values ...any) ([]byte, error) {
	if len(values) < 1 {
		return nil, errors.New("at least one value is required")
	}
	if len(values) > idx.maxObjects {
		return nil, fmt.Errorf("too many objects: %d (max %d)", len(values), idx.maxObjects)
	}
	objects, err := normalizeObjects(values)
	if err != nil {
		return nil, err
	}
	return idx.encodeBinary(objects, variantID)
}

// Decode 将 IDX 二进制块解码为 []any。
func (idx *Idx) Decode(data []byte) ([]any, error) {
	objects, err := idx.decodeBinary(data)
	if err != nil {
		return nil, err
	}
	return materializeObjects(objects)
}

func (idx *Idx) encodeBinary(objects []dataObject, variantID int) ([]byte, error) {
	if variantID < 0 || variantID >= idx.maxVariants {
		return nil, fmt.Errorf("invalid variant_id %d (max %d)", variantID, idx.maxVariants-1)
	}

	objBytes := make([]byte, 0, len(objects)*2)
	for _, obj := range objects {
		ob, err := encodeObject(obj)
		if err != nil {
			return nil, err
		}
		objBytes = append(objBytes, ob...)
	}

	mask := byte((variantID*0x9D + 0x37) & 0xFF)
	for i := range objBytes {
		objBytes[i] ^= mask
	}

	count := len(objects)
	headerLen := 1
	if count > 1 {
		headerLen = 2
	}

	data := make([]byte, headerLen+len(objBytes))
	if count == 1 {
		data[0] = byte(variantID << idx.checkBits)
	} else {
		data[0] = 0x80 | byte(variantID<<idx.checkBits)
		data[1] = byte(count)
	}
	copy(data[headerLen:], objBytes)

	xorSum := byte(0)
	for _, b := range data {
		xorSum ^= b
	}
	check := xorSum & idx.checkMask
	data[0] |= check
	return data, nil
}

func (idx *Idx) decodeBinary(data []byte) ([]dataObject, error) {
	if len(data) < 1 {
		return nil, errors.New("invalid data: too short")
	}

	byte0 := data[0]
	check := byte0 & idx.checkMask
	multi := (byte0 & 0x80) != 0
	variantID := int((byte0 & 0x7F) >> idx.checkBits)

	if variantID >= idx.maxVariants {
		return nil, fmt.Errorf("invalid variant_id %d (max %d)", variantID, idx.maxVariants-1)
	}

	headerLen := 1
	count := 1
	if multi {
		if len(data) < 2 {
			return nil, errors.New("invalid data: missing count byte")
		}
		headerLen = 2
		count = int(data[1])
		if count < 2 || count > idx.maxObjects {
			return nil, fmt.Errorf("invalid count %d", count)
		}
	}

	verify := make([]byte, len(data))
	copy(verify, data)
	verify[0] &^= idx.checkMask
	xorSum := byte(0)
	for _, b := range verify {
		xorSum ^= b
	}
	if xorSum&idx.checkMask != check {
		return nil, errors.New("checksum mismatch")
	}

	objData := make([]byte, len(data)-headerLen)
	copy(objData, data[headerLen:])
	mask := byte((variantID*0x9D + 0x37) & 0xFF)
	for i := range objData {
		objData[i] ^= mask
	}

	result := make([]dataObject, 0, count)
	pos := 0
	for i := 0; i < count; i++ {
		if pos >= len(objData) {
			return nil, errors.New("premature end of data")
		}
		obj, n, err := decodeObject(objData[pos:])
		if err != nil {
			return nil, fmt.Errorf("object[%d]: %w", i, err)
		}
		result = append(result, obj)
		pos += n
	}
	if pos != len(objData) {
		return nil, errors.New("extra bytes after data objects")
	}
	return result, nil
}

func encodeObject(obj dataObject) ([]byte, error) {
	if obj.isString {
		n := len(obj.str)
		if n < 1 || n > maxStringLen {
			return nil, fmt.Errorf("string length %d out of range [1, %d]", n, maxStringLen)
		}
		out := make([]byte, 1+n)
		out[0] = 0xC0 | byte(n) // bit7=1, bit6=1, bit5-0=len
		copy(out[1:], obj.str)
		return out, nil
	}

	if err := validateRange(obj.otype, obj.val); err != nil {
		return nil, err
	}
	if head, ok := tryEmbeddedHead(obj.otype, obj.val); ok {
		return []byte{head}, nil
	}

	sw, payload, err := payloadForNumber(obj.otype, obj.val)
	if err != nil {
		return nil, err
	}
	head := byte(0x80) | byte(sw<<4) | obj.otype // bit6=0 表示数字
	out := make([]byte, 1+len(payload))
	out[0] = head
	copy(out[1:], payload)
	return out, nil
}

func decodeObject(data []byte) (dataObject, int, error) {
	if len(data) < 1 {
		return dataObject{}, 0, errors.New("truncated object header")
	}
	head := data[0]
	if head&0x80 == 0 {
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
		return dataObject{otype: otype, val: val}, 1, nil
	}

	if head&0x40 != 0 {
		n := int(head & 0x3F)
		if n < 1 || n > maxStringLen {
			return dataObject{}, 0, fmt.Errorf("invalid string length %d", n)
		}
		if len(data) < 1+n {
			return dataObject{}, 0, errors.New("truncated string payload")
		}
		str := make([]byte, n)
		copy(str, data[1:1+n])
		return dataObject{isString: true, str: str}, 1 + n, nil
	}

	sw := (head >> 4) & 0x03
	otype := head & 0x0F
	if otype > otypeInt64 {
		return dataObject{}, 0, fmt.Errorf("invalid otype %d", otype)
	}
	numBytes := swBytes[sw]
	if len(data) < 1+numBytes {
		return dataObject{}, 0, errors.New("truncated object payload")
	}
	val, err := valueFromPayload(otype, data[1:1+numBytes])
	if err != nil {
		return dataObject{}, 0, err
	}
	if err := validateRange(otype, val); err != nil {
		return dataObject{}, 0, err
	}
	return dataObject{otype: otype, val: val}, 1 + numBytes, nil
}

func payloadForNumber(otype uint8, val int64) (sw uint8, payload []byte, err error) {
	if otype == otypeUint64 {
		mag := uint64(val)
		sw = swFromMagnitude(mag)
		return sw, uintToLEBytes(mag, swBytes[sw]), nil
	}
	if isUnsigned(otype) {
		if val < 0 {
			return 0, nil, fmt.Errorf("negative value %d for unsigned otype %d", val, otype)
		}
		mag := uint64(val)
		sw = swFromMagnitude(mag)
		return sw, uintToLEBytes(mag, swBytes[sw]), nil
	}
	sw = swFromSignedValue(val)
	payload = signedToLEBytes(val, swBytes[sw])
	return sw, payload, nil
}

func valueFromPayload(otype uint8, payload []byte) (int64, error) {
	if isUnsigned(otype) {
		mag := leBytesToUint(payload)
		if otype != otypeUint64 && mag > uint64(math.MaxInt64) {
			return 0, fmt.Errorf("value out of range for otype %d", otype)
		}
		return int64(mag), nil
	}
	return leBytesToSigned(payload), nil
}

func swFromSignedValue(val int64) uint8 {
	if val >= math.MinInt8 && val <= math.MaxInt8 {
		return 0
	}
	if val >= math.MinInt16 && val <= math.MaxInt16 {
		return 1
	}
	if val >= math.MinInt32 && val <= math.MaxInt32 {
		return 2
	}
	return 3
}

func signedToLEBytes(val int64, size int) []byte {
	buf := make([]byte, size)
	u := uint64(val)
	for i := 0; i < size; i++ {
		buf[i] = byte(u >> (8 * i))
	}
	return buf
}

func leBytesToSigned(payload []byte) int64 {
	var u uint64
	for i, b := range payload {
		u |= uint64(b) << (8 * i)
	}
	size := len(payload)
	shift := 64 - size*8
	signed := int64(u<<shift) >> shift
	return signed
}

func leBytesToUint(payload []byte) uint64 {
	var u uint64
	for i, b := range payload {
		u |= uint64(b) << (8 * i)
	}
	return u
}

func isUnsigned(otype uint8) bool { return otype <= otypeUint64 }

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

func magnitudeFromTyped(otype uint8, val int64) (mag uint64, neg bool) {
	if isUnsigned(otype) {
		return uint64(val), false
	}
	if val < 0 {
		return uint64(-val), true
	}
	return uint64(val), false
}

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

func tryEmbeddedHead(otype uint8, val int64) (byte, bool) {
	mag, neg := magnitudeFromTyped(otype, val)
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

func uintToLEBytes(v uint64, size int) []byte {
	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = byte(v >> (8 * i))
	}
	return buf
}

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
	default:
		return fmt.Errorf("invalid otype %d", otype)
	}
	return nil
}

func materializeObjects(objects []dataObject) ([]any, error) {
	out := make([]any, len(objects))
	for i, obj := range objects {
		if obj.isString {
			out[i] = string(obj.str)
			continue
		}
		v, err := materializeValue(dataObject{otype: obj.otype, val: obj.val})
		if err != nil {
			return nil, fmt.Errorf("value[%d]: %w", i, err)
		}
		out[i] = v
	}
	return out, nil
}

func materializeValue(obj dataObject) (any, error) {
	switch obj.otype {
	case otypeUint8:
		return uint8(obj.val), nil
	case otypeUint16:
		return uint16(obj.val), nil
	case otypeUint32:
		return uint32(obj.val), nil
	case otypeUint64:
		return uint64(obj.val), nil
	case otypeInt8:
		return int8(obj.val), nil
	case otypeInt16:
		return int16(obj.val), nil
	case otypeInt32:
		return int32(obj.val), nil
	case otypeInt64:
		return obj.val, nil
	default:
		return nil, fmt.Errorf("invalid otype %d", obj.otype)
	}
}

func formatCrossLangVal(otype uint8, v int64) string {
	if isUnsigned(otype) {
		return fmt.Sprintf("%d", uint64(v))
	}
	return fmt.Sprintf("%d", v)
}
