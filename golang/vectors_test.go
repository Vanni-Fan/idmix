// vectors_test.go 生成并校验跨语言测试向量（testdata/cross_language_vectors.json）。
package idmix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// 极值常量，与各语言测试共享语义。
const (
	extremeUint32Max = uint32(4294967295)
	extremeInt32Min  = int32(-2147483648)
	extremeInt64Min  = int64(-9223372036854775808)
	extremeInt64Max  = int64(9223372036854775807)
	// extremeUint64Max 为 uint64 全范围最大值。
	extremeUint64Max = uint64(18446744073709551615)
)

type crossLangValue struct {
	OType int    `json:"otype"`
	Val   string `json:"val"`
}

func formatCrossLangVal(otype uint8, v int64) string {
	if isUnsigned(otype) {
		return fmt.Sprintf("%d", uint64(v))
	}
	return fmt.Sprintf("%d", v)
}

func parseCrossLangVal(otype uint8, s string) (int64, error) {
	if isUnsigned(otype) {
		u, err := strconv.ParseUint(s, 10, 64)
		return int64(u), err
	}
	return strconv.ParseInt(s, 10, 64)
}

type crossLangCase struct {
	Name    string           `json:"name"`
	Variant int              `json:"variant"`
	Values  []crossLangValue `json:"values"`
	Encoded string           `json:"encoded"`
}

type crossLangFile struct {
	Alphabet string          `json:"alphabet"`
	Cases    []crossLangCase `json:"cases"`
}

func extremeValueCases() []struct {
	name string
	vals []any
} {
	return []struct {
		name string
		vals []any
	}{
		{"spec_example", []any{uint16(5), int64(-1), uint32(40)}},
		{"uint32_max", []any{extremeUint32Max}},
		{"int32_min", []any{extremeInt32Min}},
		{"int64_min", []any{extremeInt64Min}},
		{"int64_max", []any{extremeInt64Max}},
		{"uint64_max", []any{extremeUint64Max}},
		{
			"mixed_extremes",
			[]any{extremeUint32Max, extremeInt32Min, extremeInt64Min, extremeInt64Max},
		},
		{"embedded_small", []any{uint8(15), int8(-16), uint16(0), int16(-1)}},
		{"access_key", []any{uint32(1001), uint64(1690000000), uint8(3)}},
	}
}

func buildCrossLanguageVectors(t *testing.T) crossLangFile {
	t.Helper()
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	out := crossLangFile{Alphabet: DefaultAlphabet}
	for _, c := range extremeValueCases() {
		typed, err := normalizeAny(c.vals)
		if err != nil {
			t.Fatalf("%s: normalize: %v", c.name, err)
		}
		data, err := m.encodeBinary(typed, 0)
		if err != nil {
			t.Fatalf("%s: encodeBinary: %v", c.name, err)
		}
		enc, err := m.radix.encodeBytes(data)
		if err != nil {
			t.Fatalf("%s: radix: %v", c.name, err)
		}
		vc := crossLangCase{Name: c.name, Variant: 0, Encoded: enc}
		for _, tv := range typed {
			vc.Values = append(vc.Values, crossLangValue{OType: int(tv.otype), Val: formatCrossLangVal(tv.otype, tv.val)})
		}
		out.Cases = append(out.Cases, vc)
	}
	return out
}

func loadCrossLanguageVectors(t *testing.T) crossLangFile {
	t.Helper()
	path := filepath.Join("..", "testdata", "cross_language_vectors.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f crossLangFile
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// TestGenerateCrossLanguageVectors 在 GENERATE_VECTORS=1 时重写 testdata/cross_language_vectors.json。
func TestGenerateCrossLanguageVectors(t *testing.T) {
	if os.Getenv("GENERATE_VECTORS") != "1" {
		t.Skip("set GENERATE_VECTORS=1 to regenerate testdata/cross_language_vectors.json")
	}
	f := buildCrossLanguageVectors(t)
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "testdata", "cross_language_vectors.json")
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%d cases)", path, len(f.Cases))
}
