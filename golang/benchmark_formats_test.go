// benchmark_formats_test.go 对比 XID 与 MessagePack、CBOR、Protobuf 的编码长度与性能。
//
// 运行:
//   - go test -v -run TestCompareSerializationFormats          # 长度对比
//   - go test -v -run TestCompareSerializationFormatsPerformance # 吞吐对比
package idmix

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	cbor "github.com/fxamacker/cbor/v2"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/encoding/protowire"
)

// typedPair 表示带类型的整数，供 msgpack/cbor 无 schema 编码。
type typedPair struct {
	OType int   `msgpack:"t" cbor:"t"`
	Val   int64 `msgpack:"v" cbor:"v"`
}

type formatCase struct {
	name   string
	values []typedPair
	idmix  []any
}

func serializationFormatCases() []formatCase {
	return []formatCase{
		{
			name: "spec_example",
			idmix: []any{uint16(5), int64(-1), uint32(40)},
			values: []typedPair{
				{1, 5}, {7, -1}, {2, 40},
			},
		},
		{
			name: "uint32_max",
			idmix: []any{extremeUint32Max},
			values: []typedPair{{2, int64(extremeUint32Max)}},
		},
		{
			name: "int32_min",
			idmix: []any{extremeInt32Min},
			values: []typedPair{{6, int64(extremeInt32Min)}},
		},
		{
			name: "int64_min",
			idmix: []any{extremeInt64Min},
			values: []typedPair{{7, extremeInt64Min}},
		},
		{
			name: "int64_max",
			idmix: []any{extremeInt64Max},
			values: []typedPair{{7, extremeInt64Max}},
		},
		{
			name:  "uint64_max",
			idmix: []any{extremeUint64Max},
			values: []typedPair{{3, -1}}, // uint64 max 的 int64 bit 模式
		},
		{
			name: "mixed_extremes",
			idmix: []any{extremeUint32Max, extremeInt32Min, extremeInt64Min, extremeInt64Max, extremeUint64Max},
			values: []typedPair{
				{2, int64(extremeUint32Max)},
				{6, int64(extremeInt32Min)},
				{7, extremeInt64Min},
				{7, extremeInt64Max},
				{3, -1},
			},
		},
		{
			name: "access_key",
			idmix: []any{uint32(1001), uint64(1690000000), uint8(3)},
			values: []typedPair{{2, 1001}, {3, 1690000000}, {0, 3}},
		},
		{
			name: "embedded_small",
			idmix: []any{uint8(15), int8(-16), uint16(0), int16(-1)},
			values: []typedPair{{0, 15}, {4, -16}, {1, 0}, {5, -1}},
		},
	}
}

func encodeMsgPack(pairs []typedPair) ([]byte, error) {
	return msgpack.Marshal(pairs)
}

func encodeCBOR(pairs []typedPair) ([]byte, error) {
	return cbor.Marshal(pairs)
}

// encodeProtobuf 手工编码 TypedIntList { repeated TypedInt { otype, val } }。
func encodeProtobuf(pairs []typedPair) ([]byte, error) {
	var buf []byte
	for _, p := range pairs {
		var item []byte
		item = protowire.AppendTag(item, 1, protowire.VarintType)
		item = protowire.AppendVarint(item, uint64(p.OType))
		item = protowire.AppendTag(item, 2, protowire.VarintType)
		item = protowire.AppendVarint(item, uint64(p.Val))
		buf = protowire.AppendTag(buf, 1, protowire.BytesType)
		buf = protowire.AppendBytes(buf, item)
	}
	return buf, nil
}

func b64Len(raw []byte) int {
	return len(base64.StdEncoding.EncodeToString(raw))
}

// TestCompareSerializationFormats 对比 XID 与 MessagePack/CBOR/Protobuf 的 base64 编码长度。
func TestCompareSerializationFormats(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("══════════════════════════════════════════════════════════════════════")
	t.Log("  XID vs MessagePack / CBOR / Protobuf — base64 编码后长度对比")
	t.Log("  说明: 二进制格式统一 base64 编码后比较字符长度；XID 为文本层字符串长度")
	t.Log("══════════════════════════════════════════════════════════════════════")

	header := fmt.Sprintf("%-22s | %6s | %6s | %6s | %6s | %6s",
		"场景", "XID", "XID(b64)", "MsgPack", "CBOR", "Proto")
	t.Log(header)
	t.Log(strings.Repeat("-", len(header)))

	for _, c := range serializationFormatCases() {
		stat, err := measureIdmixLength(m, c.idmix...)
		if err != nil {
			t.Fatal(err)
		}

		mp, err := encodeMsgPack(c.values)
		if err != nil {
			t.Fatal(err)
		}
		cb, err := encodeCBOR(c.values)
		if err != nil {
			t.Fatal(err)
		}
		pb, err := encodeProtobuf(c.values)
		if err != nil {
			t.Fatal(err)
		}

		// XID 文本串再 base64 一层，便于与二进制格式公平对比字符长度
		xidB64 := base64.StdEncoding.EncodeToString([]byte(stat.sample))

		t.Logf("%-22s | %6.0f | %6d | %6d | %6d | %6d",
			c.name,
			stat.min,
			len(xidB64),
			b64Len(mp),
			b64Len(cb),
			b64Len(pb),
		)
	}
}

func decodeProtobuf(data []byte) ([]typedPair, error) {
	var pairs []typedPair
	for len(data) > 0 {
		tag, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		data = data[n:]
		if tag != 1 || typ != protowire.BytesType {
			return nil, fmt.Errorf("unexpected tag %d type %v", tag, typ)
		}
		item, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		data = data[n:]

		var p typedPair
		for len(item) > 0 {
			tag, typ, n := protowire.ConsumeTag(item)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			item = item[n:]
			switch tag {
			case 1:
				if typ != protowire.VarintType {
					return nil, fmt.Errorf("otype: unexpected type %v", typ)
				}
				v, n := protowire.ConsumeVarint(item)
				if n < 0 {
					return nil, protowire.ParseError(n)
				}
				p.OType = int(v)
				item = item[n:]
			case 2:
				if typ != protowire.VarintType {
					return nil, fmt.Errorf("val: unexpected type %v", typ)
				}
				v, n := protowire.ConsumeVarint(item)
				if n < 0 {
					return nil, protowire.ParseError(n)
				}
				p.Val = int64(v)
				item = item[n:]
			default:
				return nil, fmt.Errorf("unexpected field tag %d", tag)
			}
		}
		pairs = append(pairs, p)
	}
	return pairs, nil
}

// TestCompareSerializationFormatsPerformance 对比 XID 与 MessagePack/CBOR/Protobuf 编解码吞吐。
func TestCompareSerializationFormatsPerformance(t *testing.T) {
	const rounds = 20000

	m, err := New()
	if err != nil {
		t.Fatal(err)
	}

	perfCases := []formatCase{
		serializationFormatCases()[0], // spec_example
		serializationFormatCases()[7], // access_key
		serializationFormatCases()[8], // embedded_small
		serializationFormatCases()[6], // mixed_extremes
	}

	t.Log("══════════════════════════════════════════════════════════════════════")
	t.Logf("  XID vs MessagePack / CBOR / Protobuf — 性能对比 (各 %d 次, 单线程)", rounds)
	t.Log("  说明: 倍数 = XID ops/s ÷ 对方 ops/s；>1 表示 XID 更快")
	t.Log("══════════════════════════════════════════════════════════════════════")

	for _, c := range perfCases {
		xidStr, err := m.Encode(c.idmix...)
		if err != nil {
			t.Fatal(err)
		}

		mpRaw, err := encodeMsgPack(c.values)
		if err != nil {
			t.Fatal(err)
		}
		cbRaw, err := encodeCBOR(c.values)
		if err != nil {
			t.Fatal(err)
		}
		pbRaw, err := encodeProtobuf(c.values)
		if err != nil {
			t.Fatal(err)
		}

		xidEnc := benchOnce(rounds, func() { _, _ = m.Encode(c.idmix...) })
		mpEnc := benchOnce(rounds, func() { _, _ = encodeMsgPack(c.values) })
		cbEnc := benchOnce(rounds, func() { _, _ = encodeCBOR(c.values) })
		pbEnc := benchOnce(rounds, func() { _, _ = encodeProtobuf(c.values) })

		xidDec := benchOnce(rounds, func() { _, _ = m.Decode(xidStr) })
		mpDec := benchOnce(rounds, func() {
			var out []typedPair
			_ = msgpack.Unmarshal(mpRaw, &out)
		})
		cbDec := benchOnce(rounds, func() {
			var out []typedPair
			_ = cbor.Unmarshal(cbRaw, &out)
		})
		pbDec := benchOnce(rounds, func() { _, _ = decodeProtobuf(pbRaw) })

		t.Logf("▶ %s", c.name)
		t.Logf("  编码  XID:     %8.0f ops/s  (%6.0f ns/op)", xidEnc.opsPerSec, xidEnc.nsPerOp)
		t.Logf("  编码  MsgPack: %8.0f ops/s  (%6.0f ns/op)  [XID/MsgPack = %.2fx]", mpEnc.opsPerSec, mpEnc.nsPerOp, ratio(xidEnc.opsPerSec, mpEnc.opsPerSec))
		t.Logf("  编码  CBOR:    %8.0f ops/s  (%6.0f ns/op)  [XID/CBOR = %.2fx]", cbEnc.opsPerSec, cbEnc.nsPerOp, ratio(xidEnc.opsPerSec, cbEnc.opsPerSec))
		t.Logf("  编码  Proto:   %8.0f ops/s  (%6.0f ns/op)  [XID/Proto = %.2fx]", pbEnc.opsPerSec, pbEnc.nsPerOp, ratio(xidEnc.opsPerSec, pbEnc.opsPerSec))
		t.Logf("  解码  XID:     %8.0f ops/s  (%6.0f ns/op)", xidDec.opsPerSec, xidDec.nsPerOp)
		t.Logf("  解码  MsgPack: %8.0f ops/s  (%6.0f ns/op)  [XID/MsgPack = %.2fx]", mpDec.opsPerSec, mpDec.nsPerOp, ratio(xidDec.opsPerSec, mpDec.opsPerSec))
		t.Logf("  解码  CBOR:    %8.0f ops/s  (%6.0f ns/op)  [XID/CBOR = %.2fx]", cbDec.opsPerSec, cbDec.nsPerOp, ratio(xidDec.opsPerSec, cbDec.opsPerSec))
		t.Logf("  解码  Proto:   %8.0f ops/s  (%6.0f ns/op)  [XID/Proto = %.2fx]", pbDec.opsPerSec, pbDec.nsPerOp, ratio(xidDec.opsPerSec, pbDec.opsPerSec))
		t.Log("")
	}
}

func ratio(a, b float64) float64 {
	if b <= 0 || a <= 0 {
		return 0
	}
	return a / b
}
