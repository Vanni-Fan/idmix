// extreme_values_test.go 极值整数往返测试。
package idmix

import (
	"fmt"
	"testing"
)

func TestExtremeValuesRoundTrip(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		val  any
	}{
		{"uint32_max", extremeUint32Max},
		{"int32_min", extremeInt32Min},
		{"int64_min", extremeInt64Min},
		{"int64_max", extremeInt64Max},
		{"uint64_max", extremeUint64Max},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			list := logRoundTrip(t, m, c.name, c.val)
			if fmt.Sprintf("%v", list[0]) != fmt.Sprintf("%v", c.val) {
				t.Fatalf("got %v (%T), want %v (%T)", list[0], list[0], c.val, c.val)
			}
		})
	}
}

func TestExtremeValuesMixedRoundTrip(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	inputs := []any{extremeUint32Max, extremeInt32Min, extremeInt64Min, extremeInt64Max}
	list := logRoundTrip(t, m, "mixed_extremes", inputs...)
	want := []any{uint32(extremeUint32Max), int32(extremeInt32Min), extremeInt64Min, extremeInt64Max}
	for i := range want {
		if fmt.Sprintf("%v", list[i]) != fmt.Sprintf("%v", want[i]) {
			t.Fatalf("[%d] got %v, want %v", i, list[i], want[i])
		}
	}
}
