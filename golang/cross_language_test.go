// cross_language_test.go 校验各语言共享的固定编码串能否被 Go 正确解码。
package idmix

import (
	"fmt"
	"testing"
)

func TestCrossLanguageVectors(t *testing.T) {
	f := loadCrossLanguageVectors(t)
	m, err := New(WithAlphabet(f.Alphabet))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			list, err := m.Decode(c.Encoded)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(list) != len(c.Values) {
				t.Fatalf("count %d, want %d", len(list), len(c.Values))
			}
			for i, want := range c.Values {
				wantVal, err := parseCrossLangVal(uint8(want.OType), want.Val)
				if err != nil {
					t.Fatalf("[%d] parse val: %v", i, err)
				}
				gotTV, err := valueFromAny(list[i])
				if err != nil {
					t.Fatalf("[%d] normalize decoded: %v", i, err)
				}
				wantTV := typedValue{otype: uint8(want.OType), val: wantVal}
				if gotTV != wantTV {
					t.Fatalf("[%d] got otype=%d val=%d, want otype=%d val=%d",
						i, gotTV.otype, gotTV.val, wantTV.otype, wantTV.val)
				}
				t.Logf("  [%d] otype=%d val=%d OK", i, gotTV.otype, gotTV.val)
			}
			t.Logf("encoded=%q len=%d", c.Encoded, len(c.Encoded))
		})
	}
}

func TestCrossLanguageEncodeDeterministic(t *testing.T) {
	f := loadCrossLanguageVectors(t)
	m, err := New(WithAlphabet(f.Alphabet))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			var inputs []any
			for _, v := range c.Values {
				val, err := parseCrossLangVal(uint8(v.OType), v.Val)
				if err != nil {
					t.Fatal(err)
				}
				inputs = append(inputs, materializeFromOtypeVal(uint8(v.OType), val))
			}
			typed, err := normalizeAny(inputs)
			if err != nil {
				t.Fatal(err)
			}
			data, err := m.encodeBinary(typed, c.Variant)
			if err != nil {
				t.Fatal(err)
			}
			enc, err := m.radix.encodeBytes(data)
			if err != nil {
				t.Fatal(err)
			}
			if enc != c.Encoded {
				t.Fatalf("re-encode mismatch:\n  got  %q\n  want %q", enc, c.Encoded)
			}
		})
	}
}

func materializeFromOtypeVal(otype uint8, val int64) any {
	switch otype {
	case otypeUint8:
		return uint8(val)
	case otypeUint16:
		return uint16(val)
	case otypeUint32:
		return uint32(val)
	case otypeUint64:
		return uint64(val)
	case otypeInt8:
		return int8(val)
	case otypeInt16:
		return int16(val)
	case otypeInt32:
		return int32(val)
	case otypeInt64:
		return val
	default:
		panic(fmt.Sprintf("invalid otype %d", otype))
	}
}
