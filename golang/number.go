// number.go 负责 Encode/Decode 的公共类型转换层。
//
// 将调用方传入的 any 整数规范化为内部 typedValue 表示，
// 解码后再还原为与编码时相同的 Go 具体类型。
package idmix

import (
	"fmt"
	"math"
)

// normalizeAny 批量将 []any 转为内部 typedValue 切片。
// 任一元素类型不支持或越界时，返回带索引的错误信息。
func normalizeAny(values []any) ([]typedValue, error) {
	out := make([]typedValue, len(values))
	for i, v := range values {
		tv, err := valueFromAny(v)
		if err != nil {
			return nil, fmt.Errorf("value[%d]: %w", i, err)
		}
		out[i] = tv
	}
	return out, nil
}

// valueFromAny 将单个 any 值转为 typedValue。
//
// 支持类型及映射规则：
//   - uint8/16/32 → 保留对应无符号 otype
//   - uint64、uint → otypeUint64（值不得超过 math.MaxInt64）
//   - int8/16/32/64、int → 保留对应有符号 otype（int 统一为 otypeInt64）
//
// 不支持的类型（如 string、float）将返回错误。
func valueFromAny(v any) (typedValue, error) {
	switch x := v.(type) {
	case uint8:
		return typedValue{otypeUint8, int64(x)}, nil
	case uint16:
		return typedValue{otypeUint16, int64(x)}, nil
	case uint32:
		return typedValue{otypeUint32, int64(x)}, nil
	case uint64:
		if x > math.MaxInt64 {
			return typedValue{}, fmt.Errorf("uint64 value %d overflows int64", x)
		}
		return typedValue{otypeUint64, int64(x)}, nil
	case uint:
		if uint64(x) > math.MaxInt64 {
			return typedValue{}, fmt.Errorf("uint value %d overflows int64", x)
		}
		return typedValue{otypeUint64, int64(x)}, nil
	case int8:
		return typedValue{otypeInt8, int64(x)}, nil
	case int16:
		return typedValue{otypeInt16, int64(x)}, nil
	case int32:
		return typedValue{otypeInt32, int64(x)}, nil
	case int64:
		return typedValue{otypeInt64, x}, nil
	case int:
		return typedValue{otypeInt64, int64(x)}, nil
	default:
		return typedValue{}, fmt.Errorf("unsupported type %T (only integer types are allowed)", v)
	}
}
