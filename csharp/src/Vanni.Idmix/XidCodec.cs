namespace Vanni.Idmix;

/// <summary>XID v1.1 二进制层编解码。</summary>
public static class XidCodec
{
    private static readonly int[] SwBytes = { 1, 2, 4, 8 };
    private static readonly int[][] EmbeddedOType =
    {
        [TypedValue.OTypeUint8, TypedValue.OTypeUint16, TypedValue.OTypeUint32, TypedValue.OTypeUint64],
        [TypedValue.OTypeInt8, TypedValue.OTypeInt16, TypedValue.OTypeInt32, TypedValue.OTypeInt64],
    };

    public static byte[] EncodeBinary(IdMix m, IReadOnlyList<TypedValue> typed, int variantId)
    {
        using var objects = new MemoryStream(typed.Count * 9);
        foreach (var tv in typed)
            objects.Write(EncodeObject(tv));
        var objBytes = objects.ToArray();

        var mask = (byte)((variantId * 0x9D + 0x37) & 0xFF);
        for (var i = 0; i < objBytes.Length; i++)
            objBytes[i] ^= mask;

        var count = typed.Count;
        var header = (variantId << m.VariantShift) | (count << m.CountShift);
        var data = new byte[2 + objBytes.Length];
        data[0] = (byte)(header & 0xFF);
        data[1] = (byte)((header >> 8) & 0xFF);
        Buffer.BlockCopy(objBytes, 0, data, 2, objBytes.Length);

        var xorSum = 0;
        foreach (var b in data) xorSum ^= b;
        header |= xorSum & m.CheckMask;
        data[0] = (byte)(header & 0xFF);
        data[1] = (byte)((header >> 8) & 0xFF);
        return data;
    }

    public static List<TypedValue> DecodeBinary(IdMix m, byte[] data)
    {
        if (data.Length < 2) throw new ArgumentException("invalid data: too short");
        var header = data[0] | (data[1] << 8);
        var check = header & m.CheckMask;
        var count = (header & m.CountMask) >> m.CountShift;
        var variantId = (header & m.VariantMask) >> m.VariantShift;

        if (variantId >= m.MaxVariants) throw new ArgumentException($"invalid variant_id {variantId}");
        if (count > m.MaxObjects) throw new ArgumentException($"invalid count {count}");

        var verify = (byte[])data.Clone();
        verify[0] &= (byte)~m.CheckMask;
        var xorSum = 0;
        foreach (var b in verify) xorSum ^= b;
        if ((xorSum & m.CheckMask) != check) throw new ArgumentException("checksum mismatch");

        var objects = new byte[data.Length - 2];
        Buffer.BlockCopy(data, 2, objects, 0, objects.Length);
        var mask = (byte)((variantId * 0x9D + 0x37) & 0xFF);
        for (var i = 0; i < objects.Length; i++) objects[i] ^= mask;

        var result = new List<TypedValue>();
        var pos = 0;
        for (var i = 0; i < count; i++)
        {
            if (pos >= objects.Length) throw new ArgumentException("premature end of data");
            var dr = DecodeObject(objects, pos);
            result.Add(dr.Tv);
            pos += dr.Consumed;
        }
        if (pos != objects.Length) throw new ArgumentException("extra bytes after data objects");
        return result;
    }

    private static byte[] EncodeObject(TypedValue tv)
    {
        ValidateRange(tv.OType, tv.Val);
        if (IsUnsigned(tv.OType) && tv.Val >= 0 && tv.Val <= 15)
        {
            var wb = WidthBits(tv.OType);
            return new byte[] { (byte)(((uint)wb << 4) | ((uint)tv.Val & 0x0Fu)) };
        }
        if (IsSigned(tv.OType) && tv.Val >= -16 && tv.Val <= -1)
        {
            var wb = WidthBits(tv.OType);
            var v = (int)(-tv.Val - 1);
            return new byte[] { (byte)((1 << 6) | (wb << 4) | v) };
        }
        var swPayload = MinimalComplementBytes(tv.OType, tv.Val);
        var sw = swPayload[0];
        var outBuf = new byte[1 + swPayload.Length - 1];
        outBuf[0] = (byte)(0x80 | (sw << 4) | tv.OType);
        Buffer.BlockCopy(swPayload, 1, outBuf, 1, swPayload.Length - 1);
        return outBuf;
    }

    private sealed class DecodeResult
    {
        public TypedValue Tv { get; }
        public int Consumed { get; }

        public DecodeResult(TypedValue tv, int consumed)
        {
            Tv = tv;
            Consumed = consumed;
        }
    }

    private static DecodeResult DecodeObject(byte[] data, int offset)
    {
        if (offset >= data.Length) throw new ArgumentException("truncated object header");
        var head = data[offset];
        if ((head & 0x80) == 0)
        {
            var sign = (head >> 6) & 1;
            var wb = (head >> 4) & 0x03;
            var v = head & 0x0F;
            var otype = EmbeddedOType[sign][wb];
            var val = sign == 0 ? v : -v - 1L;
            return new DecodeResult(new TypedValue(otype, val), 1);
        }
        if (((head >> 6) & 1) != 0) throw new ArgumentException("reserved bit set in extended mode");
        var sw = (head >> 4) & 0x03;
        var otype2 = head & 0x0F;
        if (otype2 > TypedValue.OTypeInt64) throw new ArgumentException($"invalid otype {otype2}");
        var numBytes = SwBytes[sw];
        if (data.Length < offset + 1 + numBytes) throw new ArgumentException("truncated object payload");
        long raw = 0;
        for (var i = 0; i < numBytes; i++)
            raw |= (long)(data[offset + 1 + i] & 0xFF) << (8 * i);
        var val2 = ReconstructInt(otype2, sw, raw);
        return new DecodeResult(new TypedValue(otype2, val2), 1 + numBytes);
    }

    private static bool IsUnsigned(int otype) => otype <= TypedValue.OTypeUint64;
    private static bool IsSigned(int otype) => otype >= TypedValue.OTypeInt8;

    private static int WidthBits(int otype) => otype switch
    {
        TypedValue.OTypeUint8 or TypedValue.OTypeInt8 => 0,
        TypedValue.OTypeUint16 or TypedValue.OTypeInt16 => 1,
        TypedValue.OTypeUint32 or TypedValue.OTypeInt32 => 2,
        _ => 3,
    };

    private static int TargetBits(int otype) => otype switch
    {
        TypedValue.OTypeUint8 or TypedValue.OTypeInt8 => 8,
        TypedValue.OTypeUint16 or TypedValue.OTypeInt16 => 16,
        TypedValue.OTypeUint32 or TypedValue.OTypeInt32 => 32,
        _ => 64,
    };

    private static int[] MinimalComplementBytes(int otype, long val)
    {
        if (val == 0) return [0, 0];
        if (IsUnsigned(otype))
        {
            if (val < 0) throw new ArgumentException("negative value for unsigned type");
            for (var sw = 0; sw < 4; sw++)
            {
                var size = SwBytes[sw];
                if (size < 8 && val >= (1L << (size * 8))) continue;
                var buf = UintToLeBytes(val, size);
                if ((buf[size - 1] & 0x80) == 0)
                {
                    var outArr = new int[1 + size];
                    outArr[0] = sw;
                    Buffer.BlockCopy(buf, 0, outArr, 1, size);
                    return outArr;
                }
            }
            throw new ArgumentException("value too large for unsigned type");
        }
        var tbits = TargetBits(otype);
        var mask = tbits == 64 ? -1L : (1L << tbits) - 1;
        var uval = val & mask;

        if (val < 0)
        {
            for (var sw = 0; sw < 4; sw++)
            {
                var size = SwBytes[sw];
                var shift = size * 8;
                if (shift >= tbits)
                {
                    var buf = UintToLeBytes(uval, size);
                    var outArr = new int[1 + size];
                    outArr[0] = sw;
                    Buffer.BlockCopy(buf, 0, outArr, 1, size);
                    return outArr;
                }
                var lower = uval & ((1L << shift) - 1);
                var upper = uval >> shift;
                var upperMask = (1L << (tbits - shift)) - 1;
                if (upper != upperMask) continue;
                var highByte = (int)((lower >> (shift - 8)) & 0xFF);
                if ((highByte & 0x80) == 0) continue;
                var buf2 = UintToLeBytes(lower, size);
                var outArr2 = new int[1 + size];
                outArr2[0] = sw;
                Buffer.BlockCopy(buf2, 0, outArr2, 1, size);
                return outArr2;
            }
        }
        else
        {
            for (var sw = 0; sw < 4; sw++)
            {
                var size = SwBytes[sw];
                if (size < 8 && uval >= (1L << (size * 8))) continue;
                var buf = UintToLeBytes(uval, size);
                if ((buf[size - 1] & 0x80) == 0)
                {
                    var outArr = new int[1 + size];
                    outArr[0] = sw;
                    Buffer.BlockCopy(buf, 0, outArr, 1, size);
                    return outArr;
                }
            }
        }
        var swFinal = tbits switch { 8 => 0, 16 => 1, 32 => 2, _ => 3 };
        var bufFinal = UintToLeBytes(uval, SwBytes[swFinal]);
        var outFinal = new int[1 + bufFinal.Length];
        outFinal[0] = swFinal;
        Buffer.BlockCopy(bufFinal, 0, outFinal, 1, bufFinal.Length);
        return outFinal;
    }

    private static byte[] UintToLeBytes(long v, int size)
    {
        var buf = new byte[size];
        for (var i = 0; i < size; i++) buf[i] = (byte)((v >> (8 * i)) & 0xFF);
        return buf;
    }

    private static long ReconstructInt(int otype, int sw, long raw)
    {
        var tbits = TargetBits(otype);
        var storedBits = SwBytes[sw] * 8;
        if (IsUnsigned(otype))
        {
            var mask = tbits == 64 ? -1L : (1L << tbits) - 1;
            return raw & mask;
        }
        var signBit = (int)((raw >> (storedBits - 1)) & 1);
        if (tbits <= storedBits)
        {
            var mask = (1L << tbits) - 1;
            var val = raw & mask;
            if (signBit == 1 && (val & (1L << (tbits - 1))) != 0) val -= 1L << tbits;
            return val;
        }
        long extended;
        if (signBit == 1)
        {
            var extendMask = (~((1L << storedBits) - 1)) & ((1L << tbits) - 1);
            extended = raw | extendMask;
        }
        else extended = raw;
        if (extended >= (1L << (tbits - 1))) extended -= 1L << tbits;
        return extended;
    }

    private static void ValidateRange(int otype, long val)
    {
        var ok = otype switch
        {
            TypedValue.OTypeUint8 => val >= 0 && val <= 0xFF,
            TypedValue.OTypeUint16 => val >= 0 && val <= 0xFFFF,
            TypedValue.OTypeUint32 => val >= 0 && val <= 0xFFFFFFFFL,
            TypedValue.OTypeUint64 => val >= 0,
            TypedValue.OTypeInt8 => val >= -128 && val <= 127,
            TypedValue.OTypeInt16 => val >= -32768 && val <= 32767,
            TypedValue.OTypeInt32 => val >= int.MinValue && val <= int.MaxValue,
            TypedValue.OTypeInt64 => true,
            _ => false,
        };
        if (!ok) throw new ArgumentException($"value {val} out of range for otype {otype}");
    }
}
