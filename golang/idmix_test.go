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

// TestSpecExampleBinary 验证 encodeBinary 输出与 arithmetic.md 第 7 节固定样例一致。
// variant_id=0 时，[uint16(5), int64(-1), uint32(40)] 的二进制块应为固定 6 字节。
func TestSpecExampleBinary(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	typed := []typedValue{
		{otypeUint16, 5},
		{otypeInt64, -1},
		{otypeUint32, 40},
	}
	data, err := m.encodeBinary(typed, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x0F, 0x00, 0x22, 0x47, 0xB5, 0x1F}
	t.Logf("规范二进制块 (variant=0): %s", formatHex(data))
	for i := range want {
		if data[i] != want[i] {
			t.Fatalf("byte[%d] = %02X, want %02X", i, data[i], want[i])
		}
	}
	t.Log("二进制块与 arithmetic.md 第7节示例一致")
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
	raw := []byte{0x0F, 0x00, 0x22, 0x47, 0xB5, 0x1F}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := New(WithAlphabet(c.alphabet))
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
			enc, err := m.radix.encodeBytes(raw)
			if err != nil {
				t.Fatal(err)
			}
			dec, err := m.radix.decodeBytes(enc)
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
	raw, _ := m.radix.decodeBytes(str)
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
		{int32(0), 2, "扩展"},
	}
	for _, c := range cases {
		tv, err := valueFromAny(c.v)
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

// TestChecksumRejects 篡改对象区任意字节后，解码应被校验和拒绝。
func TestChecksumRejects(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	data, err := m.encodeBinary([]typedValue{{otypeUint32, 1}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("原始二进制: %s", formatHex(data))
	data[2] ^= 0x01
	t.Logf("篡改后:     %s (对象区首字节 XOR 0x01)", formatHex(data))
	str, err := m.radix.encodeBytes(data)
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

// TestNewValidation 验证 header 位域溢出、重复字符表等配置错误能被正确拒绝。
func TestNewValidation(t *testing.T) {
	_, err := New(WithMaxObjects(512), WithMaxVariants(32), WithCheckBits(2))
	if err == nil {
		t.Fatal("expected layout overflow")
	}
	t.Logf("maxObjects=512 + variant=32 => 预期错误: %v", err)

	_, err = New(WithAlphabet("abca"))
	if err == nil {
		t.Fatal("expected duplicate alphabet error")
	}
	t.Logf("重复字符 abca => 预期错误: %v", err)
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

	_, err = m.Encode("x")
	if err == nil {
		t.Fatal("expected non-integer error")
	}
	t.Logf("非整数 Encode(\"x\") => %v", err)
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
