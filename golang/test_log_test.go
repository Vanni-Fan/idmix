// test_log_test.go 提供测试辅助工具：结构化日志输出与十六进制格式化。
//
// 配合 go test -v 使用，便于在终端直观查看编码/解码的中间结果。
package idmix

import (
	"fmt"
	"strings"
	"testing"
)

// logValues 打印一组值的索引、Go 类型与内容。
func logValues(t *testing.T, prefix string, values []any) {
	t.Helper()
	for i, v := range values {
		t.Logf("%s[%d] %T = %v", prefix, i, v, v)
	}
}

// logRoundTrip 执行完整的编码→解码往返，并输出各阶段详情。
//
// 日志内容包括：字符表、编码输入、中间二进制块、最终字符串、解码输出及逐项校验结果。
// 返回解码后的 []any，供调用方做进一步断言。
func logRoundTrip(t *testing.T, m *IdMix, title string, inputs ...any) []any {
	t.Helper()
	t.Logf("────────────────────────────────────────")
	t.Logf("▶ %s", title)
	t.Logf("  字符表: %q (进制=%d)", m.radix.charsString(), m.radix.base)
	logValues(t, "  编码输入", inputs)

	str, err := m.Encode(inputs...)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}
	raw, err := m.radix.decodeBytes(str)
	if err != nil {
		t.Fatalf("文本转二进制失败: %v", err)
	}
	t.Logf("  二进制: %s (%d bytes)", formatHex(raw), len(raw))
	t.Logf("  字符串: %q (len=%d)", str, len(str))

	list, err := m.Decode(str)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}
	logValues(t, "  解码输出", list)

	for i := range inputs {
		got := fmt.Sprintf("%v(%T)", list[i], list[i])
		// int 在内部编码为 int64，对比时统一为 int64 避免误报
		wantVal := inputs[i]
		if _, ok := wantVal.(int); ok {
			wantVal = int64(wantVal.(int))
		}
		want := fmt.Sprintf("%v(%T)", wantVal, wantVal)
		okMark := "✓"
		if got != want {
			okMark = "✗"
		}
		t.Logf("  校验[%d]: %s  %s => %s", i, okMark, want, got)
	}
	return list
}

// formatHex 将字节切片格式化为空格分隔的大写十六进制字符串，便于日志阅读。
func formatHex(b []byte) string {
	if len(b) == 0 {
		return "(empty)"
	}
	var sb strings.Builder
	for i, x := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%02X", x)
	}
	return sb.String()
}

// charsString 返回字符表的字符串形式，仅供测试日志使用。
func (rc *radixCodec) charsString() string {
	return string(rc.chars)
}
