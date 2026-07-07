using System.Text;

namespace Vanni.Idmix;

internal static class Number
{
    internal const int MaxStringLen = 63;

    internal sealed class DataObject
    {
        public bool IsString { get; init; }
        public int OType { get; init; }
        public long Val { get; init; }
        public byte[] Str { get; init; } = [];
    }

    internal static DataObject[] NormalizeObjects(object[] values)
    {
        var outArr = new DataObject[values.Length];
        for (var i = 0; i < values.Length; i++)
        {
            try
            {
                outArr[i] = ObjectFromAny(values[i]);
            }
            catch (Exception ex)
            {
                throw new ArgumentException($"value[{i}]: {ex.Message}", ex);
            }
        }
        return outArr;
    }

    internal static object[] MaterializeObjects(DataObject[] objects)
    {
        var outArr = new object[objects.Length];
        for (var i = 0; i < objects.Length; i++)
        {
            if (objects[i].IsString)
            {
                outArr[i] = Encoding.UTF8.GetString(objects[i].Str);
                continue;
            }
            outArr[i] = MaterializeValue(objects[i]);
        }
        return outArr;
    }

    internal static DataObject ObjectFromAny(object v)
    {
        switch (v)
        {
            case string s:
                if (s.Length == 0)
                    throw new ArgumentException($"empty string is not allowed (max {MaxStringLen} bytes)");
                var strBytes = Encoding.UTF8.GetBytes(s);
                if (strBytes.Length > MaxStringLen)
                    throw new ArgumentException($"string length {strBytes.Length} exceeds max {MaxStringLen}");
                return new DataObject { IsString = true, Str = strBytes };
            case byte[] bs:
                if (bs.Length == 0)
                    throw new ArgumentException($"empty byte slice is not allowed (max {MaxStringLen} bytes)");
                if (bs.Length > MaxStringLen)
                    throw new ArgumentException($"byte slice length {bs.Length} exceeds max {MaxStringLen}");
                var copy = new byte[bs.Length];
                Buffer.BlockCopy(bs, 0, copy, 0, bs.Length);
                return new DataObject { IsString = true, Str = copy };
            case TypedValue tv:
                return new DataObject { OType = tv.OType, Val = tv.Val };
            case byte x: return new DataObject { OType = TypedValue.OTypeUint8, Val = x };
            case ushort x: return new DataObject { OType = TypedValue.OTypeUint16, Val = x };
            case uint x: return new DataObject { OType = TypedValue.OTypeUint32, Val = x };
            case ulong x: return new DataObject { OType = TypedValue.OTypeUint64, Val = unchecked((long)x) };
            case sbyte x: return new DataObject { OType = TypedValue.OTypeInt8, Val = x };
            case short x: return new DataObject { OType = TypedValue.OTypeInt16, Val = x };
            case int x: return new DataObject { OType = TypedValue.OTypeInt32, Val = x };
            case long x: return new DataObject { OType = TypedValue.OTypeInt64, Val = x };
            default:
                throw new ArgumentException(
                    $"unsupported type {v.GetType().Name} (integer or string up to {MaxStringLen} bytes)");
        }
    }

    private static object MaterializeValue(DataObject obj) => obj.OType switch
    {
        TypedValue.OTypeUint8 => (byte)obj.Val,
        TypedValue.OTypeUint16 => (ushort)obj.Val,
        TypedValue.OTypeUint32 => (uint)obj.Val,
        TypedValue.OTypeUint64 => unchecked((ulong)obj.Val),
        TypedValue.OTypeInt8 => (sbyte)obj.Val,
        TypedValue.OTypeInt16 => (short)obj.Val,
        TypedValue.OTypeInt32 => (int)obj.Val,
        TypedValue.OTypeInt64 => obj.Val,
        _ => throw new ArgumentException($"invalid otype {obj.OType}"),
    };
}
