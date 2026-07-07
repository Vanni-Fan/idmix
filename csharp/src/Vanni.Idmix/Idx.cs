namespace Vanni.Idmix;

/// <summary>IDX v1.2 二进制编解码器，可独立于 idmix 文本层使用。</summary>
public sealed class Idx
{
    private static readonly int[] SwBytes = { 1, 2, 4, 8 };
    private static readonly int[][] EmbeddedOType =
    {
        [TypedValue.OTypeUint8, TypedValue.OTypeUint16, TypedValue.OTypeUint32, TypedValue.OTypeUint64],
        [TypedValue.OTypeInt8, TypedValue.OTypeInt16, TypedValue.OTypeInt32, TypedValue.OTypeInt64],
    };

    public int MaxObjects { get; }
    public int MaxVariants { get; }
    public int CheckBits { get; }
    internal int CheckMask { get; }

    private Idx(int maxObjects, int maxVariants, int checkBits)
    {
        MaxObjects = maxObjects;
        MaxVariants = maxVariants;
        CheckBits = checkBits;
        CheckMask = (1 << checkBits) - 1;
    }

    /// <summary>创建 Idx 实例（默认 maxObjects=255, maxVariants=32, checkBits=2）。</summary>
    public static Idx Create() => new(255, 32, 2);

    /// <summary>使用配置委托创建 Idx 实例。</summary>
    public static Idx Create(Action<IdxBuilder> configure)
    {
        var b = new IdxBuilder();
        configure(b);
        return b.Build();
    }

    public byte[] Encode(params object[] values)
    {
        if (values.Length < 1) throw new ArgumentException("at least one value is required");
        if (values.Length > MaxObjects)
            throw new ArgumentException($"too many objects: {values.Length} (max {MaxObjects})");
        var objects = Number.NormalizeObjects(values);
        return EncodeBinary(objects, 0);
    }

    public byte[] EncodeWithVariant(int variantId, params object[] values)
    {
        if (values.Length < 1) throw new ArgumentException("at least one value is required");
        if (values.Length > MaxObjects)
            throw new ArgumentException($"too many objects: {values.Length} (max {MaxObjects})");
        var objects = Number.NormalizeObjects(values);
        return EncodeBinary(objects, variantId);
    }

    public object[] Decode(byte[] data)
    {
        var objects = DecodeBinary(data);
        return Number.MaterializeObjects(objects);
    }

    internal byte[] EncodeBinary(Number.DataObject[] objects, int variantId)
    {
        if (variantId < 0 || variantId >= MaxVariants)
            throw new ArgumentException($"invalid variant_id {variantId} (max {MaxVariants - 1})");

        using var stream = new MemoryStream(objects.Length * 9);
        foreach (var obj in objects)
            stream.Write(EncodeObject(obj));
        var objBytes = stream.ToArray();

        var mask = (byte)((variantId * 0x9D + 0x37) & 0xFF);
        for (var i = 0; i < objBytes.Length; i++)
            objBytes[i] ^= mask;

        var count = objects.Length;
        var headerLen = count == 1 ? 1 : 2;
        var data = new byte[headerLen + objBytes.Length];
        if (count == 1)
            data[0] = (byte)(variantId << CheckBits);
        else
        {
            data[0] = (byte)(0x80 | (variantId << CheckBits));
            data[1] = (byte)count;
        }
        Buffer.BlockCopy(objBytes, 0, data, headerLen, objBytes.Length);

        var xorSum = 0;
        foreach (var b in data) xorSum ^= b;
        data[0] |= (byte)(xorSum & CheckMask);
        return data;
    }

    internal Number.DataObject[] DecodeBinary(byte[] data)
    {
        if (data.Length < 1) throw new ArgumentException("invalid data: too short");

        var byte0 = data[0];
        var check = byte0 & CheckMask;
        var multi = (byte0 & 0x80) != 0;
        var variantId = (byte0 & 0x7F) >> CheckBits;

        if (variantId >= MaxVariants)
            throw new ArgumentException($"invalid variant_id {variantId} (max {MaxVariants - 1})");

        var headerLen = 1;
        var count = 1;
        if (multi)
        {
            if (data.Length < 2) throw new ArgumentException("invalid data: missing count byte");
            headerLen = 2;
            count = data[1];
            if (count < 2 || count > MaxObjects)
                throw new ArgumentException($"invalid count {count}");
        }

        var verify = (byte[])data.Clone();
        verify[0] &= (byte)~CheckMask;
        var xorSum = 0;
        foreach (var b in verify) xorSum ^= b;
        if ((xorSum & CheckMask) != check) throw new ArgumentException("checksum mismatch");

        var objData = new byte[data.Length - headerLen];
        Buffer.BlockCopy(data, headerLen, objData, 0, objData.Length);
        var mask = (byte)((variantId * 0x9D + 0x37) & 0xFF);
        for (var i = 0; i < objData.Length; i++) objData[i] ^= mask;

        var result = new List<Number.DataObject>();
        var pos = 0;
        for (var i = 0; i < count; i++)
        {
            if (pos >= objData.Length) throw new ArgumentException("premature end of data");
            var dr = DecodeObject(objData, pos);
            result.Add(dr.Obj);
            pos += dr.Consumed;
        }
        if (pos != objData.Length) throw new ArgumentException("extra bytes after data objects");
        return result.ToArray();
    }

    private static byte[] EncodeObject(Number.DataObject obj)
    {
        if (obj.IsString)
        {
            var n = obj.Str.Length;
            if (n < 1 || n > Number.MaxStringLen)
                throw new ArgumentException($"string length {n} out of range [1, {Number.MaxStringLen}]");
            var outBuf = new byte[1 + n];
            outBuf[0] = (byte)(0xC0 | n);
            Buffer.BlockCopy(obj.Str, 0, outBuf, 1, n);
            return outBuf;
        }

        ValidateRange(obj.OType, obj.Val);
        var embedded = TryEmbeddedHead(obj.OType, obj.Val);
        if (embedded.HasValue) return [embedded.Value];

        var (sw, payload) = PayloadForNumber(obj.OType, obj.Val);
        var head = (byte)(0x80 | (sw << 4) | obj.OType);
        var outBytes = new byte[1 + payload.Length];
        outBytes[0] = head;
        Buffer.BlockCopy(payload, 0, outBytes, 1, payload.Length);
        return outBytes;
    }

    private sealed record DecodeResult(Number.DataObject Obj, int Consumed);

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
            return new DecodeResult(new Number.DataObject { OType = otype, Val = val }, 1);
        }

        if ((head & 0x40) != 0)
        {
            var n = head & 0x3F;
            if (n < 1 || n > Number.MaxStringLen)
                throw new ArgumentException($"invalid string length {n}");
            if (data.Length < offset + 1 + n)
                throw new ArgumentException("truncated string payload");
            var str = new byte[n];
            Buffer.BlockCopy(data, offset + 1, str, 0, n);
            return new DecodeResult(new Number.DataObject { IsString = true, Str = str }, 1 + n);
        }

        var sw = (head >> 4) & 0x03;
        var otype2 = head & 0x0F;
        if (otype2 > TypedValue.OTypeInt64) throw new ArgumentException($"invalid otype {otype2}");
        var numBytes = SwBytes[sw];
        if (data.Length < offset + 1 + numBytes) throw new ArgumentException("truncated object payload");
        var val2 = ValueFromPayload(otype2, data, offset + 1, numBytes);
        ValidateRange(otype2, val2);
        return new DecodeResult(new Number.DataObject { OType = otype2, Val = val2 }, 1 + numBytes);
    }

    private static (int Sw, byte[] Payload) PayloadForNumber(int otype, long val)
    {
        if (otype == TypedValue.OTypeUint64)
        {
            var mag = unchecked((ulong)val);
            var sw = SwFromMagnitude(mag);
            return (sw, UintToLeBytes(mag, SwBytes[sw]));
        }
        if (IsUnsigned(otype))
        {
            if (val < 0) throw new ArgumentException($"negative value {val} for unsigned otype {otype}");
            var mag = unchecked((ulong)val);
            var sw = SwFromMagnitude(mag);
            return (sw, UintToLeBytes(mag, SwBytes[sw]));
        }
        var sw2 = SwFromSignedValue(val);
        return (sw2, SignedToLeBytes(val, SwBytes[sw2]));
    }

    private static long ValueFromPayload(int otype, byte[] data, int offset, int numBytes)
    {
        if (IsUnsigned(otype))
        {
            var mag = LeBytesToUint(data, offset, numBytes);
            if (otype != TypedValue.OTypeUint64 && mag > (ulong)long.MaxValue)
                throw new ArgumentException($"value out of range for otype {otype}");
            return unchecked((long)mag);
        }
        return LeBytesToSigned(data, offset, numBytes);
    }

    private static int SwFromSignedValue(long val)
    {
        if (val >= sbyte.MinValue && val <= sbyte.MaxValue) return 0;
        if (val >= short.MinValue && val <= short.MaxValue) return 1;
        if (val >= int.MinValue && val <= int.MaxValue) return 2;
        return 3;
    }

    private static byte[] SignedToLeBytes(long val, int size)
    {
        var buf = new byte[size];
        var u = unchecked((ulong)val);
        for (var i = 0; i < size; i++) buf[i] = (byte)(u >> (8 * i));
        return buf;
    }

    private static long LeBytesToSigned(byte[] data, int offset, int size)
    {
        ulong u = 0;
        for (var i = 0; i < size; i++) u |= (ulong)(data[offset + i] & 0xFF) << (8 * i);
        var shift = 64 - size * 8;
        return unchecked((long)(u << shift)) >> shift;
    }

    private static ulong LeBytesToUint(byte[] data, int offset, int size)
    {
        ulong u = 0;
        for (var i = 0; i < size; i++) u |= (ulong)(data[offset + i] & 0xFF) << (8 * i);
        return u;
    }

    private static bool IsUnsigned(int otype) => otype <= TypedValue.OTypeUint64;

    private static int WidthBits(int otype) => otype switch
    {
        TypedValue.OTypeUint8 or TypedValue.OTypeInt8 => 0,
        TypedValue.OTypeUint16 or TypedValue.OTypeInt16 => 1,
        TypedValue.OTypeUint32 or TypedValue.OTypeInt32 => 2,
        _ => 3,
    };

    private static (ulong Mag, bool Neg) MagnitudeFromTyped(int otype, long val)
    {
        if (IsUnsigned(otype)) return (unchecked((ulong)val), false);
        if (val < 0) return (unchecked((ulong)(-val)), true);
        return (unchecked((ulong)val), false);
    }

    private static int SwFromMagnitude(ulong mag)
    {
        if (mag < 256) return 0;
        if (mag < 65536) return 1;
        if (mag < 4294967296) return 2;
        return 3;
    }

    private static byte? TryEmbeddedHead(int otype, long val)
    {
        var (mag, neg) = MagnitudeFromTyped(otype, val);
        if (mag >= 17) return null;
        var wb = WidthBits(otype);
        if (mag == 16)
        {
            if (neg) return (byte)((1 << 6) | (wb << 4) | 15);
            return null;
        }
        if (neg) return (byte)((1 << 6) | (wb << 4) | (int)(mag - 1));
        return (byte)((wb << 4) | (int)mag);
    }

    private static byte[] UintToLeBytes(ulong v, int size)
    {
        var buf = new byte[size];
        for (var i = 0; i < size; i++) buf[i] = (byte)(v >> (8 * i));
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

    /// <summary>Idx 配置构建器。</summary>
    public sealed class IdxBuilder
    {
        public int MaxObjects { get; set; } = 255;
        public int MaxVariants { get; set; } = 32;
        public int CheckBits { get; set; } = 2;

        public Idx Build()
        {
            if (MaxObjects < 1 || MaxObjects > 255)
                throw new ArgumentException("maxObjects must be between 1 and 255");
            if (MaxVariants < 1 || MaxVariants > 32)
                throw new ArgumentException("maxVariants must be between 1 and 32");
            if (CheckBits < 1 || CheckBits > 2)
                throw new ArgumentException("checkBits must be 1 or 2");
            return new Idx(MaxObjects, MaxVariants, CheckBits);
        }
    }
}

/// <summary>Idx 配置便捷方法。</summary>
public static class IdxOptions
{
    public static void WithMaxObjects(Idx.IdxBuilder b, int n) => b.MaxObjects = n;
    public static void WithMaxVariants(Idx.IdxBuilder b, int n) => b.MaxVariants = n;
    public static void WithCheckBits(Idx.IdxBuilder b, int n) => b.CheckBits = n;
}
