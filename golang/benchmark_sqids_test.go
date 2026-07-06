// benchmark_sqids_test.go 将 idmix 与 sqids-go 进行编码长度与性能对比。
//
// 对比说明：
//   - sqids 仅支持非负整数序列；idmix 支持带类型整数（含负数）
//   - idmix 含 32 态变体，同一输入编码长度有波动，报告 min/max/avg
//   - 字母表：sqids 默认 64 字符，idmix 默认 62 字符；另附双方均用 62 字符的公平对比
//
// 运行方式：
//   - go test -v -run TestCompareSqids          # 编码长度对比
//   - go test -v -run TestCompareSqidsPerformance # 吞吐对比
//   - go test -bench=BenchmarkCompare            # 标准 benchmark
package idmix

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sqids/sqids-go"
)

// testUint32LargeSingle 大 uint32 单值对比用例（范围 1_000_000 ~ 2_400_000_000）。
const testUint32LargeSingle = uint32(2_000_000_000)

// compareAlphabet62 用于公平对比的 62 字符字母表（与 idmix 默认一致）。
// sqids 默认字母表为 64 字符，对比时额外使用此表配置 sqids。
const compareAlphabet62 = DefaultAlphabet

// compareCase 定义一组对比场景：sqids 侧的 uint64 序列与 idmix 侧的 any 序列一一对应。
type compareCase struct {
	name    string
	numbers []uint64
	// idmixValues 将 numbers 转为带类型的 any 切片（均为非负，便于公平对比长度）
	idmixValues func([]uint64) []any
}

// defaultCompareCases 返回预置的对比场景列表（经典示例、AccessKey、小整数密集等）。
func defaultCompareCases() []compareCase {
	return []compareCase{
		{
			name:    "经典示例 [1,2,3]",
			numbers: []uint64{1, 2, 3},
			idmixValues: func(ns []uint64) []any {
				return []any{uint64(ns[0]), uint64(ns[1]), uint64(ns[2])}
			},
		},
		{
			name:    "单个小ID [42]",
			numbers: []uint64{42},
			idmixValues: func(ns []uint64) []any {
				return []any{uint32(ns[0])}
			},
		},
		{
			name:    "单个大ID [123456789012345]",
			numbers: []uint64{123_456_789_012_345},
			idmixValues: func(ns []uint64) []any {
				return []any{ns[0]}
			},
		},
		{
			name:    fmt.Sprintf("单值 uint32 [%d]", testUint32LargeSingle),
			numbers: []uint64{uint64(testUint32LargeSingle)},
			idmixValues: func(ns []uint64) []any {
				return []any{uint32(ns[0])}
			},
		},
		{
			name:    "AccessKey三元组 [1001,1690000000,3]",
			numbers: []uint64{1001, 1_690_000_000, 3},
			idmixValues: func(ns []uint64) []any {
				return []any{uint32(ns[0]), uint64(ns[1]), uint8(ns[2])}
			},
		},
		{
			name:    "五个递增值 [10,20,30,40,50]",
			numbers: []uint64{10, 20, 30, 40, 50},
			idmixValues: func(ns []uint64) []any {
				out := make([]any, len(ns))
				for i, n := range ns {
					out[i] = uint32(n)
				}
				return out
			},
		},
		{
			name:    "小整数密集 [0,1,2,3,4,5,6,7,8,9]",
			numbers: []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			idmixValues: func(ns []uint64) []any {
				out := make([]any, len(ns))
				for i, n := range ns {
					out[i] = uint8(n)
				}
				return out
			},
		},
		{
			name:    "极值 uint32_max",
			numbers: []uint64{uint64(extremeUint32Max)},
			idmixValues: func(ns []uint64) []any {
				return []any{extremeUint32Max}
			},
		},
		{
			name:    "极值 int64_max",
			numbers: []uint64{uint64(extremeInt64Max)},
			idmixValues: func(ns []uint64) []any {
				return []any{extremeInt64Max}
			},
		},
		{
			name:    "极值 uint64_max",
			numbers: []uint64{extremeUint64Max},
			idmixValues: func(ns []uint64) []any {
				return []any{extremeUint64Max}
			},
		},
		{
			name: "极值三元组 [u32max,i64max,u64max]",
			numbers: []uint64{
				uint64(extremeUint32Max),
				uint64(extremeInt64Max),
				extremeUint64Max,
			},
			idmixValues: func(ns []uint64) []any {
				return []any{extremeUint32Max, extremeInt64Max, extremeUint64Max}
			},
		},
	}
}

// sqidsPerfCases 返回用于吞吐对比的场景（含极值；sqids 仅支持非负整数）。
func sqidsPerfCases(m *IdMix) []struct {
	name    string
	numbers []uint64
	encode  func() (string, error)
	decode  func(string) error
} {
	return []struct {
		name    string
		numbers []uint64
		encode  func() (string, error)
		decode  func(string) error
	}{
		{
			name:    "Encode [1,2,3]",
			numbers: []uint64{1, 2, 3},
			encode: func() (string, error) {
				return m.Encode(uint32(1), uint32(2), uint32(3))
			},
			decode: func(s string) error {
				_, err := m.Decode(s)
				return err
			},
		},
		{
			name:    "Encode [1001,1690000000,3]",
			numbers: []uint64{1001, 1_690_000_000, 3},
			encode: func() (string, error) {
				return m.Encode(uint32(1001), uint64(1_690_000_000), uint8(3))
			},
			decode: func(s string) error {
				_, err := m.Decode(s)
				return err
			},
		},
		{
			name:    fmt.Sprintf("Encode 单值 uint32 [%d]", testUint32LargeSingle),
			numbers: []uint64{uint64(testUint32LargeSingle)},
			encode: func() (string, error) {
				return m.Encode(testUint32LargeSingle)
			},
			decode: func(s string) error {
				_, err := m.Decode(s)
				return err
			},
		},
		{
			name:    "Encode 极值 uint32_max",
			numbers: []uint64{uint64(extremeUint32Max)},
			encode: func() (string, error) {
				return m.Encode(extremeUint32Max)
			},
			decode: func(s string) error {
				_, err := m.Decode(s)
				return err
			},
		},
		{
			name:    "Encode 极值 int64_max",
			numbers: []uint64{uint64(extremeInt64Max)},
			encode: func() (string, error) {
				return m.Encode(extremeInt64Max)
			},
			decode: func(s string) error {
				_, err := m.Decode(s)
				return err
			},
		},
		{
			name:    "Encode 极值 uint64_max",
			numbers: []uint64{extremeUint64Max},
			encode: func() (string, error) {
				return m.Encode(extremeUint64Max)
			},
			decode: func(s string) error {
				_, err := m.Decode(s)
				return err
			},
		},
		{
			name: "Encode 极值三元组 [u32max,i64max,u64max]",
			numbers: []uint64{
				uint64(extremeUint32Max),
				uint64(extremeInt64Max),
				extremeUint64Max,
			},
			encode: func() (string, error) {
				return m.Encode(extremeUint32Max, extremeInt64Max, extremeUint64Max)
			},
			decode: func(s string) error {
				_, err := m.Decode(s)
				return err
			},
		},
	}
}

// lengthStat 汇总多次编码的字符串长度统计（min/max/avg）及样例字符串。
type lengthStat struct {
	min, max, avg float64
	sample        string
}

// measureIdmixLength 对同一组值编码 32 次（覆盖不同变体），统计字符串长度分布。
func measureIdmixLength(m *IdMix, values ...any) (lengthStat, error) {
	const rounds = 32
	minLen, maxLen := int(^uint(0)>>1), 0
	total := 0
	var sample string
	for i := 0; i < rounds; i++ {
		s, err := m.Encode(values...)
		if err != nil {
			return lengthStat{}, err
		}
		l := len(s)
		total += l
		if l < minLen {
			minLen = l
		}
		if l > maxLen {
			maxLen = l
		}
		if i == 0 {
			sample = s
		}
	}
	return lengthStat{
		min:    float64(minLen),
		max:    float64(maxLen),
		avg:    float64(total) / rounds,
		sample: sample,
	}, nil
}

// measureSqidsLength 测量 sqids 单次编码的字符串长度（sqids 无变体，长度固定）。
func measureSqidsLength(s *sqids.Sqids, numbers []uint64) lengthStat {
	id, err := s.Encode(numbers)
	if err != nil {
		panic(err)
	}
	l := len(id)
	return lengthStat{min: float64(l), max: float64(l), avg: float64(l), sample: id}
}

// benchEncode 标准 benchmark 包装，额外报告 ops/s 与 ns/op 指标。
func benchEncode(b *testing.B, name string, fn func()) {
	b.Helper()
	b.ResetTimer()
	start := time.Now()
	for i := 0; i < b.N; i++ {
		fn()
	}
	elapsed := time.Since(start)
	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "ops/s")
	b.ReportMetric(float64(elapsed.Nanoseconds())/float64(b.N), "ns/op")
	_ = name
}

// TestCompareSqids 对比 idmix 与 sqids-go 的编码长度（-v 输出详情）。
func TestCompareSqids(t *testing.T) {
	idmixDefault, err := New()
	if err != nil {
		t.Fatal(err)
	}
	idmix62, err := New(WithAlphabet(compareAlphabet62))
	if err != nil {
		t.Fatal(err)
	}

	sqidsDefault, err := sqids.New()
	if err != nil {
		t.Fatal(err)
	}
	sqids62, err := sqids.New(sqids.Options{Alphabet: compareAlphabet62})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("══════════════════════════════════════════════════════════════")
	t.Log("  idmix vs sqids-go  编码长度对比")
	t.Log("  说明: sqids 仅支持非负整数; idmix 长度含 32 态变体(报告 min/max/avg)")
	t.Log("  字母表: idmix 默认 62 字符; sqids 默认 64 字符; 另附双方均用 62 字符对比")
	t.Log("══════════════════════════════════════════════════════════════")

	header := fmt.Sprintf("%-28s | %-12s | %-12s | %-12s | %-12s",
		"场景", "sqids默认", "idmix默认", "sqids@62", "idmix@62")
	t.Log(header)
	t.Log(strings.Repeat("-", len(header)))

	for _, c := range defaultCompareCases() {
		values := c.idmixValues(c.numbers)

		sDef := measureSqidsLength(sqidsDefault, c.numbers)
		s62 := measureSqidsLength(sqids62, c.numbers)

		iDef, err := measureIdmixLength(idmixDefault, values...)
		if err != nil {
			t.Fatal(err)
		}
		i62, err := measureIdmixLength(idmix62, values...)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%-28s | %4.0f (%-5s) | %4.0f~%-4.0f(%-5s) | %4.0f (%-5s) | %4.0f~%-4.0f(%-5s)",
			c.name,
			sDef.avg, truncSample(sDef.sample, 8),
			iDef.min, iDef.max, truncSample(iDef.sample, 8),
			s62.avg, truncSample(s62.sample, 8),
			i62.min, i62.max, truncSample(i62.sample, 8),
		)
	}

	t.Log("")
	t.Log("── 典型场景编码样例 ──")
	c := defaultCompareCases()[0]
	values := c.idmixValues(c.numbers)
	sid, _ := sqidsDefault.Encode(c.numbers)
	iid, _ := idmixDefault.Encode(values...)
	t.Logf("输入: %v", c.numbers)
	t.Logf("  sqids 默认(64字符表): %q (len=%d)", sid, len(sid))
	t.Logf("  idmix 默认(62字符表): %q (len=%d, 单次采样)", iid, len(iid))

	t.Log("")
	t.Log("── idmix 额外能力（sqids 不支持）──")
	str, err := idmixDefault.Encode(uint16(5), int64(-1), uint32(40))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("带类型+负数: uint16(5), int64(-1), uint32(40) => %q (len=%d)", str, len(str))
}

// TestCompareSqidsPerformance 对比编解码吞吐（-v 输出）。
func TestCompareSqidsPerformance(t *testing.T) {
	const rounds = 20000

	idmixM, err := New()
	if err != nil {
		t.Fatal(err)
	}
	sqidsM, err := sqids.New()
	if err != nil {
		t.Fatal(err)
	}

	cases := sqidsPerfCases(idmixM)

	t.Log("══════════════════════════════════════════════════════════════")
	t.Logf("  性能对比 (各 %d 次, 单线程)", rounds)
	t.Log("══════════════════════════════════════════════════════════════")

	for _, c := range cases {
		sid, err := sqidsM.Encode(c.numbers)
		if err != nil {
			t.Fatal(err)
		}
		iid, err := c.encode()
		if err != nil {
			t.Fatal(err)
		}

		sqEnc := benchOnce(rounds, func() {
			_, _ = sqidsM.Encode(c.numbers)
		})
		idEnc := benchOnce(rounds, func() {
			_, _ = c.encode()
		})
		sqDec := benchOnce(rounds, func() {
			_ = sqidsM.Decode(sid)
		})
		idDec := benchOnce(rounds, func() {
			_ = c.decode(iid)
		})

		t.Logf("▶ %s", c.name)
		t.Logf("  编码  sqids: %8.0f ops/s  (%6.0f ns/op)", sqEnc.opsPerSec, sqEnc.nsPerOp)
		t.Logf("  编码  idmix: %8.0f ops/s  (%6.0f ns/op)  [idmix/sqids = %.2fx]", idEnc.opsPerSec, idEnc.nsPerOp, ratio(idEnc.opsPerSec, sqEnc.opsPerSec))
		t.Logf("  解码  sqids: %8.0f ops/s  (%6.0f ns/op)", sqDec.opsPerSec, sqDec.nsPerOp)
		t.Logf("  解码  idmix: %8.0f ops/s  (%6.0f ns/op)  [idmix/sqids = %.2fx]", idDec.opsPerSec, idDec.nsPerOp, ratio(idDec.opsPerSec, sqDec.opsPerSec))
		t.Logf("  字符串长度  sqids=%d  idmix=%d", len(sid), len(iid))
		t.Log("")
	}
}

// benchResult 单次性能测量的吞吐与延迟结果。
type benchResult struct {
	opsPerSec float64
	nsPerOp   float64
}

// benchOnce 执行 rounds 次 fn 并计算吞吐与单次耗时。
func benchOnce(rounds int, fn func()) benchResult {
	start := time.Now()
	for i := 0; i < rounds; i++ {
		fn()
	}
	elapsed := time.Since(start)
	return benchResult{
		opsPerSec: float64(rounds) / elapsed.Seconds(),
		nsPerOp:   float64(elapsed.Nanoseconds()) / float64(rounds),
	}
}

// truncSample 截断过长样例字符串，便于表格对齐输出。
func truncSample(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// ── go test -bench 用标准 benchmark ─────────────────────────────

// BenchmarkCompareEncode_Small3 小整数三元组 [1,2,3] 的编码吞吐对比。
func BenchmarkCompareEncode_Small3(b *testing.B) {
	m, _ := New()
	s, _ := sqids.New()
	nums := []uint64{1, 2, 3}
	b.Run("sqids", func(b *testing.B) {
		benchEncode(b, "sqids", func() { _, _ = s.Encode(nums) })
	})
	b.Run("idmix", func(b *testing.B) {
		benchEncode(b, "idmix", func() { _, _ = m.Encode(uint32(1), uint32(2), uint32(3)) })
	})
}

// BenchmarkCompareEncode_AccessKey 典型 AccessKey 三元组编码吞吐对比。
func BenchmarkCompareEncode_AccessKey(b *testing.B) {
	m, _ := New()
	s, _ := sqids.New()
	nums := []uint64{1001, 1_690_000_000, 3}
	b.Run("sqids", func(b *testing.B) {
		benchEncode(b, "sqids", func() { _, _ = s.Encode(nums) })
	})
	b.Run("idmix", func(b *testing.B) {
		benchEncode(b, "idmix", func() { _, _ = m.Encode(uint32(1001), uint64(1_690_000_000), uint8(3)) })
	})
}

// BenchmarkCompareEncode_LargeSingle 单个大整数编码吞吐对比。
func BenchmarkCompareEncode_LargeSingle(b *testing.B) {
	m, _ := New()
	s, _ := sqids.New()
	nums := []uint64{123_456_789_012_345}
	b.Run("sqids", func(b *testing.B) {
		benchEncode(b, "sqids", func() { _, _ = s.Encode(nums) })
	})
	b.Run("idmix", func(b *testing.B) {
		benchEncode(b, "idmix", func() { _, _ = m.Encode(uint64(123_456_789_012_345)) })
	})
}

// BenchmarkCompareEncode_Uint32LargeSingle 单值大 uint32（百万~24亿）编码吞吐对比。
func BenchmarkCompareEncode_Uint32LargeSingle(b *testing.B) {
	m, _ := New()
	s, _ := sqids.New()
	nums := []uint64{uint64(testUint32LargeSingle)}
	b.Run("sqids", func(b *testing.B) {
		benchEncode(b, "sqids", func() { _, _ = s.Encode(nums) })
	})
	b.Run("idmix", func(b *testing.B) {
		benchEncode(b, "idmix", func() { _, _ = m.Encode(testUint32LargeSingle) })
	})
}

// BenchmarkCompareDecode_Uint32LargeSingle 单值大 uint32 解码吞吐对比。
func BenchmarkCompareDecode_Uint32LargeSingle(b *testing.B) {
	m, _ := New()
	s, _ := sqids.New()
	nums := []uint64{uint64(testUint32LargeSingle)}
	sid, _ := s.Encode(nums)
	iid, _ := m.Encode(testUint32LargeSingle)
	b.Run("sqids", func(b *testing.B) {
		benchEncode(b, "sqids", func() { _ = s.Decode(sid) })
	})
	b.Run("idmix", func(b *testing.B) {
		benchEncode(b, "idmix", func() { _, _ = m.Decode(iid) })
	})
}

// BenchmarkCompareDecode_Small3 小整数三元组解码吞吐对比。
func BenchmarkCompareDecode_Small3(b *testing.B) {
	m, _ := New()
	s, _ := sqids.New()
	nums := []uint64{1, 2, 3}
	sid, _ := s.Encode(nums)
	iid, _ := m.Encode(uint32(1), uint32(2), uint32(3))
	b.Run("sqids", func(b *testing.B) {
		benchEncode(b, "sqids", func() { _ = s.Decode(sid) })
	})
	b.Run("idmix", func(b *testing.B) {
		benchEncode(b, "idmix", func() { _, _ = m.Decode(iid) })
	})
}

// BenchmarkCompareDecode_AccessKey AccessKey 三元组解码吞吐对比。
func BenchmarkCompareDecode_AccessKey(b *testing.B) {
	m, _ := New()
	s, _ := sqids.New()
	nums := []uint64{1001, 1_690_000_000, 3}
	sid, _ := s.Encode(nums)
	iid, _ := m.Encode(uint32(1001), uint64(1_690_000_000), uint8(3))
	b.Run("sqids", func(b *testing.B) {
		benchEncode(b, "sqids", func() { _ = s.Decode(sid) })
	})
	b.Run("idmix", func(b *testing.B) {
		benchEncode(b, "idmix", func() { _, _ = m.Decode(iid) })
	})
}
