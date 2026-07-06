// uint64_limits_test.go 验证 uint64 全范围往返。
package idmix

import (
	"testing"
)

func TestUint64MaxRoundTrip(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	str, err := m.Encode(extremeUint64Max)
	if err != nil {
		t.Fatal(err)
	}
	list, err := m.Decode(str)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].(uint64) != extremeUint64Max {
		t.Fatalf("got %v", list[0])
	}
}

func TestUint64AboveInt64MaxRoundTrip(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	v := uint64(9223372036854775808) // 0x8000000000000000
	str, err := m.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	list, err := m.Decode(str)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].(uint64) != v {
		t.Fatalf("got %v", list[0])
	}
}
