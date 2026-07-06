namespace Vanni.Idmix;

/// <summary>原始类型索引与带类型整数值。</summary>
public sealed class TypedValue
{
    public const int OTypeUint8 = 0;
    public const int OTypeUint16 = 1;
    public const int OTypeUint32 = 2;
    public const int OTypeUint64 = 3;
    public const int OTypeInt8 = 4;
    public const int OTypeInt16 = 5;
    public const int OTypeInt32 = 6;
    public const int OTypeInt64 = 7;

    public int OType { get; }
    public long Val { get; }

    public TypedValue(int otype, long val)
    {
        OType = otype;
        Val = val;
    }

    public static TypedValue U8(int v) => new(OTypeUint8, v);
    public static TypedValue U16(int v) => new(OTypeUint16, v);
    public static TypedValue U32(long v) => new(OTypeUint32, v);
    public static TypedValue U64(long v) => new(OTypeUint64, v);
    public static TypedValue U64(ulong v) => new(OTypeUint64, unchecked((long)v));
    public static TypedValue U64(string v) => new(OTypeUint64, unchecked((long)ulong.Parse(v)));
    public static TypedValue I8(int v) => new(OTypeInt8, v);
    public static TypedValue I16(int v) => new(OTypeInt16, v);
    public static TypedValue I32(int v) => new(OTypeInt32, v);
    public static TypedValue I64(long v) => new(OTypeInt64, v);

    public override bool Equals(object? obj) =>
        obj is TypedValue tv && OType == tv.OType && Val == tv.Val;

    public override int GetHashCode() => HashCode.Combine(OType, Val);

    public override string ToString() => $"TypedValue{{otype={OType}, val={Val}}}";
}
