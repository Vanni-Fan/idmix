// benchmark_formats_test.go 对比 IDX 与 MessagePack、CBOR、Protobuf、FlatBuffers、Cap'n Proto 的二进制长度与性能。
//
// 运行:
//   - go test -v -run TestCompareSerializationFormats          # 长度对比
//   - go test -v -run TestCompareSerializationFormatsPerformance # 吞吐对比
package idmix

import (
	"fmt"
	"strings"
	"testing"

	cbor "github.com/fxamacker/cbor/v2"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/encoding/protowire"
	capnp "capnproto.org/go/capnp/v3"
)

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
			name:  "spec_example",
			idmix: []any{uint16(5), int64(-1), uint32(40)},
			values: []typedPair{
				{1, 5}, {7, -1}, {2, 40},
			},
		},
		{
			name:   "uint32_max",
			idmix:  []any{extremeUint32Max},
			values: []typedPair{{2, int64(extremeUint32Max)}},
		},
		{
			name:   "int32_min",
			idmix:  []any{extremeInt32Min},
			values: []typedPair{{6, int64(extremeInt32Min)}},
		},
		{
			name:   "int64_min",
			idmix:  []any{extremeInt64Min},
			values: []typedPair{{7, extremeInt64Min}},
		},
		{
			name:   "int64_max",
			idmix:  []any{extremeInt64Max},
			values: []typedPair{{7, extremeInt64Max}},
		},
		{
			name:   "uint64_max",
			idmix:  []any{extremeUint64Max},
			values: []typedPair{{3, -1}},
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
			name:  "access_key",
			idmix: []any{uint32(1001), uint64(1690000000), uint8(3)},
			values: []typedPair{
				{2, 1001}, {3, 1690000000}, {0, 3},
			},
		},
		{
			name:  "embedded_small",
			idmix: []any{uint8(15), int8(-16), uint16(0), int16(-1)},
			values: []typedPair{
				{0, 15}, {4, -16}, {1, 0}, {5, -1},
			},
		},
		{
			name:  "string_example",
			idmix: []any{"hello", uint16(5), "世界"},
			values: []typedPair{
				{1, 5},
			},
		},
	}
}

func encodeMsgPack(pairs []typedPair) ([]byte, error) {
	return msgpack.Marshal(pairs)
}

func encodeCBOR(pairs []typedPair) ([]byte, error) {
	return cbor.Marshal(pairs)
}

// ── Protobuf (raw protowire) ──────────────────────────────────
// Encodes each typedPair as a length-delimited sub-message with
// field 1 (otype varint) + field 2 (val varint).

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

// ── FlatBuffers ───────────────────────────────────────────────
// Uses the flatbuffers.Builder API directly (no schema compilation needed).
//
// Schema equivalent:
//   table TypedPair { otype:int32; val:int64; }
//   table TypedPairList { pairs:[TypedPair]; }
//   root_type TypedPairList;

func encodeFlatBuffers(pairs []typedPair) ([]byte, error) {
	builder := flatbuffers.NewBuilder(0)

	// Build leaf tables in reverse order (FlatBuffers convention).
	offsets := make([]flatbuffers.UOffsetT, len(pairs))
	for i := len(pairs) - 1; i >= 0; i-- {
		builder.StartObject(2)
		builder.PrependInt32Slot(0, int32(pairs[i].OType), 0)
		builder.PrependInt64Slot(1, pairs[i].Val, 0)
		offsets[i] = builder.EndObject()
	}

	// Build a vector of table offsets (field 0 in root table).
	builder.StartVector(4, len(pairs), 4)
	for i := len(pairs) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	vecOff := builder.EndVector(len(pairs))

	// Root table: TypedPairList, field 0 = pairs vector.
	builder.StartObject(1)
	builder.PrependUOffsetTSlot(0, vecOff, 0)
	rootOff := builder.EndObject()
	builder.Finish(rootOff)

	return builder.FinishedBytes(), nil
}

func decodeFlatBuffers(data []byte) ([]typedPair, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("flatbuffers: data too short")
	}
	rootTable := flatbuffers.Table{
		Bytes: data,
		Pos:   flatbuffers.GetUOffsetT(data[0:]),
	}

	// Read pairs vector (field 0, vtable slot offset = 4).
	o := rootTable.Offset(4)
	if o == 0 {
		return nil, fmt.Errorf("flatbuffers: pairs field not found")
	}
	fieldOff := flatbuffers.UOffsetT(o)
	vecLen := rootTable.VectorLen(fieldOff)
	vecStart := rootTable.Vector(fieldOff)
	result := make([]typedPair, 0, vecLen)
	for i := 0; i < vecLen; i++ {
		elemOff := rootTable.Indirect(vecStart + flatbuffers.UOffsetT(i*4))
		elemTable := flatbuffers.Table{Bytes: data, Pos: elemOff}
		result = append(result, typedPair{
			OType: int(elemTable.GetInt32Slot(4, 0)),
			Val:   elemTable.GetInt64Slot(6, 0),
		})
	}
	return result, nil
}

// ── Cap'n Proto ───────────────────────────────────────────────
// Uses capnproto.org/go/capnp/v3 segment-level API (no schema compilation needed).
//
// Schema equivalent (schema_types.capnp):
//   struct TypedPair { otype @0 :Int32; val @1 :Int64; }
//   struct TypedPairList { pairs @0 :List(TypedPair); }
//
// In wire format, a composite list element is:
//   Int32@0 (4B) + padding (4B) + Int64@8 (8B) = 16B per element.

const capnpPairDataSize = 16 // bytes per TypedPair struct data section

func encodeCapnp(pairs []typedPair) ([]byte, error) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	sz := capnp.ObjectSize{DataSize: capnpPairDataSize}
	list, err := capnp.NewCompositeList(seg, sz, int32(len(pairs)))
	if err != nil {
		return nil, err
	}

	for i, p := range pairs {
		st := list.Struct(i)
		st.SetUint32(0, uint32(p.OType))
		st.SetUint64(8, uint64(p.Val))
	}

	// Set root to list pointer.
	seg.Message().SetRoot(list.ToPtr())

	return seg.Message().Marshal()
}

func decodeCapnp(data []byte) ([]typedPair, error) {
	msg, err := capnp.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	root, err := msg.Root()
	if err != nil {
		return nil, err
	}

	list := root.List()
	n := list.Len()
	result := make([]typedPair, 0, n)
	for i := 0; i < n; i++ {
		// Each element is 2 words: Int32 at offset 0, Int64 at offset 8.
		elem := list.Struct(i)
		result = append(result, typedPair{
			OType: int(int32(elem.Uint32(0))),
			Val:   int64(elem.Uint64(8)),
		})
	}
	return result, nil
}

// ── Test helpers ──────────────────────────────────────────────

func idxBinaryLen(m *IdMix, values ...any) (int, error) {
	data, err := m.encodeBinary(values, 0)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// TestCompareSerializationFormats 对比 IDX 与 MessagePack / CBOR / Protobuf / FlatBuffers / Cap'n Proto 的二进制字节数。
func TestCompareSerializationFormats(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("══════════════════════════════════════════════════════════════════════════════")
	t.Log("  IDX vs MsgPack / CBOR / Protobuf / FlatBuffers / Cap'n Proto — 二进制字节数对比")
	t.Log("══════════════════════════════════════════════════════════════════════════════")

	header := fmt.Sprintf("%-18s | %4s | %6s | %4s | %4s | %4s | %4s",
		"Scenario", "IDX", "MsgPack", "CBOR", "Proto", "FB", "Capnp")
	t.Log(header)
	t.Log(strings.Repeat("-", len(header)))

	for _, c := range serializationFormatCases() {
		idxLen, err := idxBinaryLen(m, c.idmix...)
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
		fb, err := encodeFlatBuffers(c.values)
		if err != nil {
			t.Fatal(err)
		}
		cap, err := encodeCapnp(c.values)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%-18s | %4d | %6d | %4d | %4d | %4d | %4d",
			c.name, idxLen, len(mp), len(cb), len(pb), len(fb), len(cap))
	}
}

// TestCompareSerializationFormatsPerformance 对比 IDX 与 MessagePack / CBOR / Protobuf / FlatBuffers / Cap'n Proto 编解码吞吐。
func TestCompareSerializationFormatsPerformance(t *testing.T) {
	const rounds = 20000

	idx, err := NewIdx()
	if err != nil {
		t.Fatal(err)
	}
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

	t.Log("══════════════════════════════════════════════════════════════════════════════")
	t.Logf("  IDX vs MsgPack / CBOR / Protobuf / FlatBuffers / Cap'n Proto — 性能对比 (各 %d 次, 单线程)", rounds)
	t.Log("  倍数 = IDX ops/s ÷ 对方 ops/s；>1 表示 IDX 更快")
	t.Log("══════════════════════════════════════════════════════════════════════════════")

	for _, c := range perfCases {
		idxRaw, err := m.encodeBinary(c.idmix, 0)
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
		fbRaw, err := encodeFlatBuffers(c.values)
		if err != nil {
			t.Fatal(err)
		}
		capRaw, err := encodeCapnp(c.values)
		if err != nil {
			t.Fatal(err)
		}

		idxEnc := benchOnce(rounds, func() { _, _ = m.encodeBinary(c.idmix, 0) })
		mpEnc := benchOnce(rounds, func() { _, _ = encodeMsgPack(c.values) })
		cbEnc := benchOnce(rounds, func() { _, _ = encodeCBOR(c.values) })
		pbEnc := benchOnce(rounds, func() { _, _ = encodeProtobuf(c.values) })
		fbEnc := benchOnce(rounds, func() { _, _ = encodeFlatBuffers(c.values) })
		capEnc := benchOnce(rounds, func() { _, _ = encodeCapnp(c.values) })

		idxDec := benchOnce(rounds, func() { _, _ = idx.Decode(idxRaw) })
		mpDec := benchOnce(rounds, func() {
			var out []typedPair
			_ = msgpack.Unmarshal(mpRaw, &out)
		})
		cbDec := benchOnce(rounds, func() {
			var out []typedPair
			_ = cbor.Unmarshal(cbRaw, &out)
		})
		pbDec := benchOnce(rounds, func() { _, _ = decodeProtobuf(pbRaw) })
		fbDec := benchOnce(rounds, func() { _, _ = decodeFlatBuffers(fbRaw) })
		capDec := benchOnce(rounds, func() { _, _ = decodeCapnp(capRaw) })

		t.Logf("▶ %s", c.name)
		t.Logf("  编码  IDX:     %8.0f ops/s  (%6.0f ns/op)", idxEnc.opsPerSec, idxEnc.nsPerOp)
		t.Logf("  编码  MsgPack: %8.0f ops/s  (%6.0f ns/op)  [IDX/MsgPack = %.2fx]", mpEnc.opsPerSec, mpEnc.nsPerOp, ratio(idxEnc.opsPerSec, mpEnc.opsPerSec))
		t.Logf("  编码  CBOR:    %8.0f ops/s  (%6.0f ns/op)  [IDX/CBOR = %.2fx]", cbEnc.opsPerSec, cbEnc.nsPerOp, ratio(idxEnc.opsPerSec, cbEnc.opsPerSec))
		t.Logf("  编码  Proto:   %8.0f ops/s  (%6.0f ns/op)  [IDX/Proto = %.2fx]", pbEnc.opsPerSec, pbEnc.nsPerOp, ratio(idxEnc.opsPerSec, pbEnc.opsPerSec))
		t.Logf("  编码  FlatBuf: %8.0f ops/s  (%6.0f ns/op)  [IDX/FB = %.2fx]", fbEnc.opsPerSec, fbEnc.nsPerOp, ratio(idxEnc.opsPerSec, fbEnc.opsPerSec))
		t.Logf("  编码  Capnp:   %8.0f ops/s  (%6.0f ns/op)  [IDX/Capnp = %.2fx]", capEnc.opsPerSec, capEnc.nsPerOp, ratio(idxEnc.opsPerSec, capEnc.opsPerSec))
		t.Logf("  解码  IDX:     %8.0f ops/s  (%6.0f ns/op)", idxDec.opsPerSec, idxDec.nsPerOp)
		t.Logf("  解码  MsgPack: %8.0f ops/s  (%6.0f ns/op)  [IDX/MsgPack = %.2fx]", mpDec.opsPerSec, mpDec.nsPerOp, ratio(idxDec.opsPerSec, mpDec.opsPerSec))
		t.Logf("  解码  CBOR:    %8.0f ops/s  (%6.0f ns/op)  [IDX/CBOR = %.2fx]", cbDec.opsPerSec, cbDec.nsPerOp, ratio(idxDec.opsPerSec, cbDec.opsPerSec))
		t.Logf("  解码  Proto:   %8.0f ops/s  (%6.0f ns/op)  [IDX/Proto = %.2fx]", pbDec.opsPerSec, pbDec.nsPerOp, ratio(idxDec.opsPerSec, pbDec.opsPerSec))
		t.Logf("  解码  FlatBuf: %8.0f ops/s  (%6.0f ns/op)  [IDX/FB = %.2fx]", fbDec.opsPerSec, fbDec.nsPerOp, ratio(idxDec.opsPerSec, fbDec.opsPerSec))
		t.Logf("  解码  Capnp:   %8.0f ops/s  (%6.0f ns/op)  [IDX/Capnp = %.2fx]", capDec.opsPerSec, capDec.nsPerOp, ratio(idxDec.opsPerSec, capDec.opsPerSec))
		t.Log("")
	}
}

func ratio(a, b float64) float64 {
	if b <= 0 || a <= 0 {
		return 0
	}
	return a / b
}
