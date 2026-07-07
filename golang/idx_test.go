// idx_test.go 覆盖 Idx 配置项与字符串长度边界。
package idmix

import (
	"strings"
	"testing"
)

func TestStringLengthBoundaries(t *testing.T) {
	idx, err := NewIdx()
	if err != nil {
		t.Fatal(err)
	}

	ok63 := strings.Repeat("a", maxStringLen)
	tooLong := strings.Repeat("b", maxStringLen+1)

	t.Run("empty_string_rejected", func(t *testing.T) {
		_, err := idx.Encode("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
		t.Logf("空字符串 => %v", err)
	})

	t.Run("empty_bytes_rejected", func(t *testing.T) {
		_, err := idx.Encode([]byte{})
		if err == nil {
			t.Fatal("expected error for empty []byte")
		}
	})

	t.Run("len_1_ok", func(t *testing.T) {
		data, err := idx.EncodeWithVariant(0, "x")
		if err != nil {
			t.Fatal(err)
		}
		out, err := idx.Decode(data)
		if err != nil || out[0].(string) != "x" {
			t.Fatalf("round-trip: %v %v", out, err)
		}
	})

	t.Run("len_63_ok", func(t *testing.T) {
		data, err := idx.EncodeWithVariant(0, ok63)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != 1+1+maxStringLen {
			t.Fatalf("binary len=%d, want %d", len(data), 1+1+maxStringLen)
		}
		out, err := idx.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		if out[0].(string) != ok63 {
			t.Fatal("decoded string mismatch")
		}
		t.Logf("63 字节字符串编码 OK, 总长 %d 字节", len(data))
	})

	t.Run("len_64_rejected", func(t *testing.T) {
		_, err := idx.Encode(tooLong)
		if err == nil {
			t.Fatal("expected error for 64-byte string")
		}
		t.Logf("64 字节 => %v", err)
	})

	t.Run("len_64_bytes_rejected", func(t *testing.T) {
		_, err := idx.Encode([]byte(tooLong))
		if err == nil {
			t.Fatal("expected error for 64-byte []byte")
		}
	})

	t.Run("idmix_end_to_end_63", func(t *testing.T) {
		m, err := New()
		if err != nil {
			t.Fatal(err)
		}
		str, err := m.EncodeWithVariant(0, ok63)
		if err != nil {
			t.Fatal(err)
		}
		list, err := m.Decode(str)
		if err != nil {
			t.Fatal(err)
		}
		if list[0].(string) != ok63 {
			t.Fatal("IdMix 63-byte round-trip failed")
		}
	})

	t.Run("idmix_len_64_rejected", func(t *testing.T) {
		m, err := New()
		if err != nil {
			t.Fatal(err)
		}
		_, err = m.Encode(tooLong)
		if err == nil {
			t.Fatal("expected IdMix encode error for 64-byte string")
		}
	})
}

func TestIdxMaxObjects(t *testing.T) {
	t.Run("new_invalid", func(t *testing.T) {
		for _, n := range []int{0, 256} {
			_, err := NewIdx(WithMaxObjects(n))
			if err == nil {
				t.Fatalf("maxObjects=%d should be rejected", n)
			}
			t.Logf("maxObjects=%d => %v", n, err)
		}
	})

	t.Run("encode_over_limit", func(t *testing.T) {
		idx, err := NewIdx(WithMaxObjects(2))
		if err != nil {
			t.Fatal(err)
		}
		_, err = idx.Encode(uint8(1), uint8(2), uint8(3))
		if err == nil {
			t.Fatal("expected too many objects error")
		}
		t.Logf("3 objects max=2 => %v", err)
	})

	t.Run("encode_at_limit", func(t *testing.T) {
		idx, err := NewIdx(WithMaxObjects(2))
		if err != nil {
			t.Fatal(err)
		}
		data, err := idx.EncodeWithVariant(0, uint8(1), uint8(2))
		if err != nil {
			t.Fatal(err)
		}
		if data[0]&0x80 == 0 {
			t.Fatal("two objects should use 2-byte header")
		}
		if data[1] != 2 {
			t.Fatalf("count=%d, want 2", data[1])
		}
		out, err := idx.Decode(data)
		if err != nil || len(out) != 2 {
			t.Fatalf("decode: %v len=%d", err, len(out))
		}
	})

	t.Run("max_255_new_ok", func(t *testing.T) {
		idx, err := NewIdx(WithMaxObjects(255))
		if err != nil {
			t.Fatal(err)
		}
		if idx.maxObjects != 255 {
			t.Fatalf("got %d", idx.maxObjects)
		}
	})
}

func TestIdxMaxVariants(t *testing.T) {
	t.Run("new_invalid", func(t *testing.T) {
		for _, n := range []int{0, 33} {
			_, err := NewIdx(WithMaxVariants(n))
			if err == nil {
				t.Fatalf("maxVariants=%d should be rejected", n)
			}
			t.Logf("maxVariants=%d => %v", n, err)
		}
	})

	t.Run("encode_variant_out_of_range", func(t *testing.T) {
		idx, err := NewIdx(WithMaxVariants(4))
		if err != nil {
			t.Fatal(err)
		}
		_, err = idx.EncodeWithVariant(4, uint8(1))
		if err == nil {
			t.Fatal("variant_id=4 should fail when maxVariants=4")
		}
		t.Logf("variant=4 max=4 => %v", err)
	})

	t.Run("encode_variant_at_limit", func(t *testing.T) {
		idx, err := NewIdx(WithMaxVariants(4))
		if err != nil {
			t.Fatal(err)
		}
		data, err := idx.EncodeWithVariant(3, uint8(42))
		if err != nil {
			t.Fatal(err)
		}
		out, err := idx.Decode(data)
		if err != nil || out[0].(uint8) != 42 {
			t.Fatalf("variant=3 round-trip: %v %v", out, err)
		}
	})

	t.Run("decode_rejects_high_variant", func(t *testing.T) {
		large, err := NewIdx(WithMaxVariants(32))
		if err != nil {
			t.Fatal(err)
		}
		data, err := large.EncodeWithVariant(20, uint8(7))
		if err != nil {
			t.Fatal(err)
		}
		small, err := NewIdx(WithMaxVariants(16))
		if err != nil {
			t.Fatal(err)
		}
		_, err = small.Decode(data)
		if err == nil {
			t.Fatal("expected invalid variant_id when maxVariants=16")
		}
		t.Logf("variant=20 max=16 => %v", err)
	})

	t.Run("idmix_respects_max_variants", func(t *testing.T) {
		idx, err := NewIdx(WithMaxVariants(8))
		if err != nil {
			t.Fatal(err)
		}
		m, err := New(WithIdx(idx))
		if err != nil {
			t.Fatal(err)
		}
		_, err = m.EncodeWithVariant(8, uint32(1))
		if err == nil {
			t.Fatal("EncodeWithVariant(8) should fail when maxVariants=8")
		}
	})
}

func TestIdxCheckBits(t *testing.T) {
	t.Run("new_invalid", func(t *testing.T) {
		for _, n := range []int{0, 3} {
			_, err := NewIdx(WithCheckBits(n))
			if err == nil {
				t.Fatalf("checkBits=%d should be rejected", n)
			}
			t.Logf("checkBits=%d => %v", n, err)
		}
	})

	t.Run("checkBits_1_roundtrip", func(t *testing.T) {
		idx, err := NewIdx(WithCheckBits(1))
		if err != nil {
			t.Fatal(err)
		}
		if idx.checkBits != 1 || idx.checkMask != 0x01 {
			t.Fatalf("checkMask=%02X", idx.checkMask)
		}
		data, err := idx.EncodeWithVariant(0, uint16(5), int64(-1))
		if err != nil {
			t.Fatal(err)
		}
		out, err := idx.Decode(data)
		if err != nil || out[0].(uint16) != 5 {
			t.Fatalf("round-trip: %v", err)
		}
	})

	t.Run("checkBits_1_rejects_tamper", func(t *testing.T) {
		idx, err := NewIdx(WithCheckBits(1))
		if err != nil {
			t.Fatal(err)
		}
		data, err := idx.EncodeWithVariant(0, uint32(1))
		if err != nil {
			t.Fatal(err)
		}
		data[len(data)-1] ^= 0x01
		_, err = idx.Decode(data)
		if err == nil {
			t.Fatal("expected checksum mismatch with checkBits=1")
		}
	})

	t.Run("checkBits_2_default", func(t *testing.T) {
		idx, err := NewIdx()
		if err != nil {
			t.Fatal(err)
		}
		if idx.checkBits != 2 || idx.checkMask != 0x03 {
			t.Fatalf("default checkBits=%d mask=%02X", idx.checkBits, idx.checkMask)
		}
	})

	t.Run("idmix_with_checkBits_1", func(t *testing.T) {
		idx, err := NewIdx(WithCheckBits(1))
		if err != nil {
			t.Fatal(err)
		}
		m, err := New(WithIdx(idx))
		if err != nil {
			t.Fatal(err)
		}
		str, err := m.EncodeWithVariant(0, uint32(99))
		if err != nil {
			t.Fatal(err)
		}
		list, err := m.Decode(str)
		if err != nil || list[0].(uint32) != 99 {
			t.Fatalf("IdMix checkBits=1: %v", err)
		}
	})
}
