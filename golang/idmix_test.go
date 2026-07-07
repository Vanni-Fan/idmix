// idmix_test.go 覆盖 XID v1.1 编解码的核心行为与边界条件。
//
// 测试分组：
//   - 演示用例（Demo*）：带 -v 日志的往返示例，便于人工阅读
//   - 规范一致性（TestSpecExampleBinary）：与 arithmetic.md 第 7 节二进制样例对齐
//   - 进制层（TestRadixRoundTrip）：自定义字符表往返
//   - 类型与模式（TestRoundTrip*、TestEmbeddedModes）：全类型、内嵌/扩展模式
//   - 安全与校验（TestChecksumRejects、TestRejectRateApprox）：篡改拒绝、随机串误通过率
//   - 配置校验（TestNewValidation、TestEncodeErrors）：非法参数与输入
//   - 大 uint32 单值（TestRoundTripUint32SingleLarge）：百万~24亿量级与 sqids 对比场景
package idmix

import (
	"fmt"
	"strings"
	"testing"
)

// TestRoundTripUint32SingleLarge 单值 uint32 大整数往返（典型时间戳/ID 量级，见 testUint32LargeSingle）。
func TestRoundTripUint32SingleLarge(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	list := logRoundTrip(t, m, fmt.Sprintf("单值 uint32(%d)", testUint32LargeSingle), testUint32LargeSingle)
	if list[0].(uint32) != testUint32LargeSingle {
		t.Fatalf("got %v, want %d", list[0], testUint32LargeSingle)
	}
}

// TestDemoEncodeDecode 规范文档中的经典示例：uint16(5), int64(-1), uint32(40)。
func TestDemoEncodeDecode(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	logRoundTrip(t, m, "规范示例: uint16(5), int64(-1), uint32(40)",
		uint16(5), int64(-1), uint32(40))
}

// TestDemoCustomChinese 验证 Unicode 字符表可作为自定义进制使用。
func TestDemoCustomChinese(t *testing.T) {
	m, err := New(WithAlphabet("一二三四五六七八九十"))
	if err != nil {
		t.Fatal(err)
	}
	logRoundTrip(t, m, "中文字符表 一二三四五六七八九十",
		uint16(100), int32(-10), uint8(3))
}

// TestDemoCustomAlphabet 四进制字符表往返示例。
func TestDemoCustomAlphabet(t *testing.T) {
	m, err := New(WithAlphabet("abcd"))
	if err != nil {
		t.Fatal(err)
	}
	logRoundTrip(t, m, "四进制字符表 abcd",
		uint16(100), int32(-10), uint8(3))
}

// TestDemoMixedSmallInts 混合内嵌模式（小整数 1 字节）与扩展模式（较大整数）。
func TestDemoMixedSmallInts(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	logRoundTrip(t, m, "小整数(内嵌模式) + 普通整数",
		uint8(10), int16(-5), uint32(1000), int64(-999))
}

// TestSpecExampleBinary 验证 encodeBinary 输出与 arithmetic.md 固定样例一致。
// variant_id=0 时，[uint16(5), int64(-1), uint32(40)] 为 3 对象块（2 字节 header）。
func TestSpecExampleBinary(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.encodeBinary([]any{uint16(5), int64(-1), uint32(40)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F}
	t.Logf("规范二进制块 (variant=0): %s", formatHex(data))
	for i := range want {
		if data[i] != want[i] {
			t.Fatalf("byte[%d] = %02X, want %02X (full: %s)", i, data[i], want[i], formatHex(data))
		}
	}
	t.Log("二进制块与 arithmetic.md 示例一致")
}

// TestRadixRoundTrip 测试进制层的字符表校验与字节往返。
func TestRadixRoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		alphabet  string
		wantError bool
	}{
		{"default62", DefaultAlphabet, false},
		{"quaternary", "abcd", false},
		{"base11", "0123456789abc", false},
		{"duplicate", "abca", true},
		{"tooShort", "a", true},
	}
	raw := []byte{0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := New(WithAlphabet(c.alphabet))
			if c.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				t.Logf("字符表 %q => 预期错误: %v", c.alphabet, err)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			rc, err := NewRadixCodec(c.alphabet)
			if err != nil {
				t.Fatal(err)
			}
			enc, err := EncodeBytes(raw, rc)
			if err != nil {
				t.Fatal(err)
			}
			dec, err := DecodeString(enc, rc)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("进制=%d  原始=%s", len([]rune(c.alphabet)), formatHex(raw))
			t.Logf("  编码串: %q (len=%d)", enc, len(enc))
			t.Logf("  还原:   %s", formatHex(dec))
			if string(dec) != string(raw) {
				t.Fatalf("round-trip 失败")
			}
		})
	}
}

// TestRoundTripBasic 基础端到端往返，并断言解码后的具体类型与值。
func TestRoundTripBasic(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	list := logRoundTrip(t, m, "基础往返", uint16(5), int64(-1), uint32(40))
	if list[0].(uint16) != 5 || list[1].(int64) != -1 || list[2].(uint32) != 40 {
		t.Fatal("值不匹配")
	}
}

// TestRoundTripAllTypes 覆盖全部支持的整数类型及内嵌/扩展边界值。
func TestRoundTripAllTypes(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	inputs := []any{
		uint8(0), uint8(15), uint16(128), uint32(0x7FFFFFFF), uint64(1 << 40),
		int8(-16), int8(-1), int8(127), int16(-128), int32(0), int64(-1), 42,
	}
	str, err := m.Encode(inputs...)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := DecodeString(str, m.Codec())
	t.Logf("全类型测试: 输入 %d 个值", len(inputs))
	t.Logf("  二进制: %s (%d bytes)", formatHex(raw), len(raw))
	t.Logf("  字符串: %q (len=%d)", str, len(str))

	list, err := m.Decode(str)
	if err != nil {
		t.Fatal(err)
	}
	want := []any{
		uint8(0), uint8(15), uint16(128), uint32(0x7FFFFFFF), uint64(1 << 40),
		int8(-16), int8(-1), int8(127), int16(-128), int32(0), int64(-1), int64(42),
	}
	for i := range want {
		got := fmt.Sprintf("%v", list[i])
		wantStr := fmt.Sprintf("%v", want[i])
		mark := "✓"
		if got != wantStr {
			mark = "✗"
			t.Fatalf("[%d] got %v (%T), want %v", i, list[i], list[i], want[i])
		}
		t.Logf("  [%02d] %s %T(%v)", i, mark, list[i], list[i])
	}
}

// TestRoundTripCustomBase 自定义四进制字符表下的端到端往返。
func TestRoundTripCustomBase(t *testing.T) {
	m, err := New(WithAlphabet("abcd"))
	if err != nil {
		t.Fatal(err)
	}
	list := logRoundTrip(t, m, "自定义四进制", uint16(5), int32(-3), uint8(9))
	if list[0].(uint16) != 5 || list[1].(int32) != -3 || list[2].(uint8) != 9 {
		t.Fatal()
	}
}

// TestEmbeddedModes 验证内嵌模式（1 字节）与扩展模式（2+ 字节）的对象长度选择。
func TestEmbeddedModes(t *testing.T) {
	t.Log("内嵌模式 vs 扩展模式（对象字节长度）:")
	cases := []struct {
		v    any
		len  int
		mode string
	}{
		{uint8(10), 1, "内嵌"},
		{int16(-5), 1, "内嵌"},
		{uint32(16), 2, "扩展"},
		{int32(0), 1, "内嵌"},
	}
	for _, c := range cases {
		tv, err := objectFromAny(c.v)
		if err != nil {
			t.Fatal(err)
		}
		obj, err := encodeObject(tv)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("  %T(%v) => %s, %d byte(s), hex=%s",
			c.v, c.v, c.mode, len(obj), formatHex(obj))
		if len(obj) != c.len {
			t.Fatalf("len = %d, want %d", len(obj), c.len)
		}
	}
}

// TestSingleObjectOneByteHeader 单元素编码时 IDX 整体头仅 1 字节（无 count 字节）。
func TestSingleObjectOneByteHeader(t *testing.T) {
	idx, err := NewIdx()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		val   any
		objLen int // 对象区字节数（不含 header）
	}{
		{"embedded_uint8", uint8(10), 1},
		{"embedded_int16_neg", int16(-5), 1},
		{"extended_uint32", uint32(1000), 3}, // 1 对象头 + 2 负载
		{"extended_string", "hi", 1 + 2}, // 1 头 + 2 负载
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := idx.EncodeWithVariant(0, c.val)
			if err != nil {
				t.Fatal(err)
			}
			wantTotal := 1 + c.objLen
			if len(data) != wantTotal {
				t.Fatalf("len=%d, want %d (1-byte header + object)", len(data), wantTotal)
			}
			if data[0]&0x80 != 0 {
				t.Fatalf("header byte0=%02X, bit7 should be 0 (single object)", data[0])
			}
			t.Logf("单元素 %v => %s (header 1 字节, 总长 %d)", c.val, formatHex(data), len(data))

			out, err := idx.Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(out) != 1 {
				t.Fatalf("decoded count=%d, want 1", len(out))
			}
			got := fmt.Sprintf("%v", out[0])
			want := fmt.Sprintf("%v", c.val)
			if got != want {
				t.Fatalf("decoded %v, want %v", out[0], c.val)
			}
		})
	}

	// 对比：两元素时 header 为 2 字节
	multi, err := idx.EncodeWithVariant(0, uint8(1), uint8(2))
	if err != nil {
		t.Fatal(err)
	}
	if len(multi) < 2 {
		t.Fatal("multi-object block too short")
	}
	if multi[0]&0x80 == 0 {
		t.Fatalf("multi header byte0=%02X, bit7 should be 1", multi[0])
	}
	if multi[1] != 2 {
		t.Fatalf("count byte=%d, want 2", multi[1])
	}
	t.Logf("双元素对比 => %s (header 2 字节)", formatHex(multi))
}

// TestSingleObjectHeaderIdMix 单元素经 IdMix 端到端往返，二进制层仍为 1 字节头。
func TestSingleObjectHeaderIdMix(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	str, err := m.EncodeWithVariant(0, uint32(42))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := DecodeString(str, m.Codec())
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 1 {
		t.Fatal("empty binary")
	}
	if raw[0]&0x80 != 0 {
		t.Fatalf("IdMix 单元素 header bit7=1, got %02X", raw[0])
	}
	list, err := m.Decode(str)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].(uint32) != 42 {
		t.Fatalf("got %v, want 42", list[0])
	}
	t.Logf("IdMix 单元素: binary=%s str=%q", formatHex(raw), str)
}

// TestStringRoundTrip 扩展模式字符串（≤63 字节）往返。
func TestStringRoundTrip(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	list := logRoundTrip(t, m, "字符串 + 整数", "hello", uint16(5), "世界")
	if list[0].(string) != "hello" || list[1].(uint16) != 5 || list[2].(string) != "世界" {
		t.Fatal("值不匹配")
	}
}

// TestCustomCodec 验证可插拔 Codec（Base64、异或包装等）。
func TestCustomCodec(t *testing.T) {
	raw := []byte{0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F}

	t.Run("base64", func(t *testing.T) {
		c := NewBase64Codec()
		s, err := EncodeBytes(raw, c)
		if err != nil {
			t.Fatal(err)
		}
		out, err := DecodeString(s, c)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != string(raw) {
			t.Fatal("base64 round-trip failed")
		}
		m, err := New(WithCodec(c))
		if err != nil {
			t.Fatal(err)
		}
		str, err := m.EncodeWithVariant(0, uint16(5), int64(-1), uint32(40))
		if err != nil {
			t.Fatal(err)
		}
		list, err := m.Decode(str)
		if err != nil {
			t.Fatal(err)
		}
		if list[0].(uint16) != 5 {
			t.Fatal("IdMix with Base64Codec failed")
		}
	})

	t.Run("xor_wrap_radix", func(t *testing.T) {
		inner := mustRadix(t, DefaultAlphabet)
		const key byte = 0x5A
		xor := FuncCodec{
			EncodeFn: func(data []byte) (string, error) {
				buf := make([]byte, len(data))
				for i, b := range data {
					buf[i] = b ^ key
				}
				return inner.Encode(buf)
			},
			DecodeFn: func(s string) ([]byte, error) {
				buf, err := inner.Decode(s)
				if err != nil {
					return nil, err
				}
				for i := range buf {
					buf[i] ^= key
				}
				return buf, nil
			},
		}
		s, err := EncodeBytes(raw, xor)
		if err != nil {
			t.Fatal(err)
		}
		out, err := DecodeString(s, xor)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != string(raw) {
			t.Fatal("xor+radix round-trip failed")
		}
	})
}

func mustRadix(t *testing.T, alphabet string) *RadixCodec {
	t.Helper()
	rc, err := NewRadixCodec(alphabet)
	if err != nil {
		t.Fatal(err)
	}
	return rc
}

// TestEncodeBytesStandalone 包级 EncodeBytes/DecodeString 可独立包装任意二进制。
func TestEncodeBytesStandalone(t *testing.T) {
	raw := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	s, err := EncodeBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(raw) {
		t.Fatalf("got %x, want %x", out, raw)
	}
	t.Logf("EncodeBytes: %q => %q", raw, s)

	s2, err := EncodeBytes(raw, mustRadix(t, "abcd"))
	if err != nil {
		t.Fatal(err)
	}
	out2, err := DecodeString(s2, mustRadix(t, "abcd"))
	if err != nil {
		t.Fatal(err)
	}
	if string(out2) != string(raw) {
		t.Fatal("custom alphabet round-trip failed")
	}
}

// TestChecksumRejects 篡改对象区任意字节后，解码应被校验和拒绝。
func TestChecksumRejects(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.encodeBinary([]any{uint32(1)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("原始二进制: %s", formatHex(data))
	data[1] ^= 0x01
	t.Logf("篡改后:     %s (对象区首字节 XOR 0x01)", formatHex(data))
	str, err := EncodeBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("篡改字符串: %q", str)
	_, err = m.Decode(str)
	if err == nil {
		t.Fatal("expected checksum error")
	}
	t.Logf("解码结果: 拒绝 ✓  (%v)", err)
}

// TestNewValidation 验证 IdMix / Idx 非法配置能被正确拒绝。
func TestNewValidation(t *testing.T) {
	_, err := NewIdx(WithMaxObjects(256))
	if err == nil {
		t.Fatal("expected maxObjects overflow")
	}
	t.Logf("maxObjects=256 => 预期错误: %v", err)

	_, err = New(WithAlphabet("abca"))
	if err == nil {
		t.Fatal("expected duplicate alphabet error")
	}
	t.Logf("重复字符 abca => 预期错误: %v", err)

	idx, err := NewIdx(WithMaxObjects(100))
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(WithIdx(idx), WithCodec(mustRadix(t, "abcd")))
	if err != nil {
		t.Fatal(err)
	}
	if m.Idx().maxObjects != 100 {
		t.Fatalf("idx maxObjects = %d, want 100", m.Idx().maxObjects)
	}
}

// TestEncodeErrors 空参数与非整数类型应返回明确错误。
func TestEncodeErrors(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Encode()
	if err == nil {
		t.Fatal("expected empty encode error")
	}
	t.Logf("空参数 Encode => %v", err)

	_, err = m.Encode(3.14)
	if err == nil {
		t.Fatal("expected non-integer error")
	}
	t.Logf("非整数 Encode(3.14) => %v", err)
}

// TestDecodeInvalidChar 字符不在自定义字符表中时应解码失败。
func TestDecodeInvalidChar(t *testing.T) {
	m, err := New(WithAlphabet("abcd"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Decode("axy")
	if err == nil {
		t.Fatal("expected decode error")
	}
	t.Logf("非法字符 axy (四进制) => %v", err)
}

// TestMultipleEncodingsDiffer 同一输入多次编码应产生多种不同字符串（变体多态性）。
func TestMultipleEncodingsDiffer(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]struct{})
	samples := make([]string, 0, 8)
	for i := 0; i < 50; i++ {
		s, err := m.Encode(uint32(42))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := seen[s]; !ok && len(samples) < 8 {
			samples = append(samples, s)
		}
		seen[s] = struct{}{}
	}
	t.Logf("同一数据 uint32(42) 编码 50 次，产生 %d 种不同字符串（变体混淆）:", len(seen))
	for i, s := range samples {
		t.Logf("  样例[%d]: %q", i, s)
	}
	if len(seen) > len(samples) {
		t.Logf("  ... 另有 %d 种未列出", len(seen)-len(samples))
	}
	if len(seen) < 2 {
		t.Fatal("expected multiple variants")
	}
}

// TestRejectRateApprox 随机字符串的误通过率应低于 30%（2-bit 校验的统计特性）。
func TestRejectRateApprox(t *testing.T) {
	m, err := New(WithAlphabet("abcd"))
	if err != nil {
		t.Fatal(err)
	}
	pass := 0
	chars := []rune("abcd")
	const n = 5000
	for i := 0; i < n; i++ {
		var b strings.Builder
		for j := 0; j < 8; j++ {
			b.WriteRune(chars[(i*3+j*7)%4])
		}
		if _, err := m.Decode(b.String()); err == nil {
			pass++
		}
	}
	rate := pass * 100 / n
	t.Logf("随机 8 字符串解码测试: %d 次, 误通过率 %d%% (期望 <30%%)", n, rate)
	if rate > 30 {
		t.Fatalf("pass rate %d%% too high", rate)
	}
}
