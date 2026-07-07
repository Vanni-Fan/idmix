// number.go 负责 Encode/Decode 的公共类型转换层。
//
// 将调用方传入的 any 规范化为内部 dataObject 表示，
// 解码后再还原为与编码时相同的 Go 具体类型。
package idmix

import (
	"fmt"
)

func normalizeObjects(values []any) ([]dataObject, error) {
	out := make([]dataObject, len(values))
	for i, v := range values {
		obj, err := objectFromAny(v)
		if err != nil {
			return nil, fmt.Errorf("value[%d]: %w", i, err)
		}
		out[i] = obj
	}
	return out, nil
}

func objectFromAny(v any) (dataObject, error) {
	switch x := v.(type) {
	case string:
		if len(x) == 0 {
			return dataObject{}, fmt.Errorf("empty string is not allowed (max %d bytes)", maxStringLen)
		}
		if len(x) > maxStringLen {
			return dataObject{}, fmt.Errorf("string length %d exceeds max %d", len(x), maxStringLen)
		}
		return dataObject{isString: true, str: []byte(x)}, nil
	case []byte:
		if len(x) == 0 {
			return dataObject{}, fmt.Errorf("empty byte slice is not allowed (max %d bytes)", maxStringLen)
		}
		if len(x) > maxStringLen {
			return dataObject{}, fmt.Errorf("byte slice length %d exceeds max %d", len(x), maxStringLen)
		}
		str := make([]byte, len(x))
		copy(str, x)
		return dataObject{isString: true, str: str}, nil
	case uint8:
		return dataObject{otype: otypeUint8, val: int64(x)}, nil
	case uint16:
		return dataObject{otype: otypeUint16, val: int64(x)}, nil
	case uint32:
		return dataObject{otype: otypeUint32, val: int64(x)}, nil
	case uint64:
		return dataObject{otype: otypeUint64, val: int64(x)}, nil
	case uint:
		return dataObject{otype: otypeUint64, val: int64(x)}, nil
	case int8:
		return dataObject{otype: otypeInt8, val: int64(x)}, nil
	case int16:
		return dataObject{otype: otypeInt16, val: int64(x)}, nil
	case int32:
		return dataObject{otype: otypeInt32, val: int64(x)}, nil
	case int64:
		return dataObject{otype: otypeInt64, val: x}, nil
	case int:
		return dataObject{otype: otypeInt64, val: int64(x)}, nil
	default:
		return dataObject{}, fmt.Errorf("unsupported type %T (integer or string up to %d bytes)", v, maxStringLen)
	}
}

func objectFromAnyValue(v any) (dataObject, error) {
	return objectFromAny(v)
}
