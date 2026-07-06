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
        var embedded = TryEmbeddedHead(tv.OType, tv.Val);
        if (embedded.HasValue) return new byte[] { embedded.Value };
        var (mag, neg) = MagnitudeFromTyped(tv.OType, tv.Val);
        var sw = SwFromMagnitude(mag);
        var payload = UintToLeBytes(mag, SwBytes[sw]);
        var head = 0x80 | (sw << 4) | tv.OType;
        if (neg) head |= 1 << 6;
        var outBuf = new byte[1 + payload.Length];
        outBuf[0] = (byte)head;
        Buffer.BlockCopy(payload, 0, outBuf, 1, payload.Length);
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
        var sw = (head >> 4) & 0x03;
        var otype2 = head & 0x0F;
        if (otype2 > TypedValue.OTypeInt64) throw new ArgumentException($"invalid otype {otype2}");
        var numBytes = SwBytes[sw];
        if (data.Length < offset + 1 + numBytes) throw new ArgumentException("truncated object payload");
        long mag = 0;
        for (var i = 0; i < numBytes; i++)
            mag |= (long)(data[offset + 1 + i] & 0xFF) << (8 * i);
        var neg = ((head >> 6) & 1) != 0;
        var val2 = ValueFromMagnitude(mag, neg);
        ValidateRange(otype2, val2);
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

    private static (long Mag, bool Neg) MagnitudeFromTyped(int otype, long val)
    {
        if (IsUnsigned(otype)) return (val, false);
        if (val < 0) return (-val, true);
        return (val, false);
    }

    private static int SwFromMagnitude(long mag)
    {
        if (unchecked((ulong)mag) < 256) return 0;
        if (unchecked((ulong)mag) < 65536) return 1;
        if (unchecked((ulong)mag) < 4294967296L) return 2;
        return 3;
    }

    private static byte? TryEmbeddedHead(int otype, long val)
    {
        var (mag, neg) = MagnitudeFromTyped(otype, val);
        if (unchecked((ulong)mag) >= 17) return null;
        var wb = WidthBits(otype);
        if (unchecked((ulong)mag) == 16)
        {
            if (neg) return (byte)((1 << 6) | (wb << 4) | 15);
            return null;
        }
        if (neg) return (byte)((1 << 6) | (wb << 4) | (int)(mag - 1));
        return (byte)((wb << 4) | (int)mag);
    }

    private static long ValueFromMagnitude(long mag, bool neg)
    {
        if (!neg) return mag;
        if (mag == 1L << 63) return long.MinValue;
        return -mag;
    }

    private static byte[] UintToLeBytes(long v, int size)
    {
        var buf = new byte[size];
        for (var i = 0; i < size; i++) buf[i] = (byte)((unchecked((ulong)v) >> (8 * i)) & 0xFF);
        return buf;
    }

    private static void ValidateRange(int otype, long val)
    {
        var ok = otype switch
        {
            TypedValue.OTypeUint8 => val >= 0 && val <= 0xFF,
            TypedValue.OTypeUint16 => val >= 0 && val <= 0xFFFF,
            TypedValue.OTypeUint32 => val >= 0 && val <= 0xFFFFFFFFL,
            TypedValue.OTypeUint64 => true,
            TypedValue.OTypeInt8 => val >= -128 && val <= 127,
            TypedValue.OTypeInt16 => val >= -32768 && val <= 32767,
            TypedValue.OTypeInt32 => val >= int.MinValue && val <= int.MaxValue,
            TypedValue.OTypeInt64 => true,
            _ => false,
        };
        if (!ok) throw new ArgumentException($"value {val} out of range for otype {otype}");
    }
}
