using System.Numerics;

namespace Vanni.Idmix;

/// <summary>XID v1.1 编解码器主入口。</summary>
public sealed class IdMix
{
    public const string DefaultAlphabet =
        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    private readonly RadixCodec _radix;
    private readonly Random _random = new();

    public int MaxObjects { get; }
    public int MaxVariants { get; }
    public int CheckBits { get; }
    public int CountBits { get; private set; }
    public int VariantBits { get; private set; }
    public int CheckMask { get; private set; }
    public int CountMask { get; private set; }
    public int VariantMask { get; private set; }
    public int CountShift { get; private set; }
    public int VariantShift { get; private set; }

    public IdMix() : this(DefaultAlphabet, 511, 32, 2) { }

    public IdMix(string alphabet) : this(alphabet, 511, 32, 2) { }

    public IdMix(string alphabet, int maxObjects, int maxVariants, int checkBits)
    {
        _radix = new RadixCodec(alphabet);
        MaxObjects = maxObjects;
        MaxVariants = maxVariants;
        CheckBits = checkBits;
        FinalizeLayout();
    }

    public static IdMix NewDefault() => new();

    public RadixCodec Radix => _radix;

    public string Encode(params TypedValue[] values)
    {
        if (values.Length < 1) throw new ArgumentException("at least one value is required");
        if (values.Length > MaxObjects)
            throw new ArgumentException($"too many objects: {values.Length}");
        var variantId = _random.Next(MaxVariants);
        var data = XidCodec.EncodeBinary(this, values, variantId);
        return _radix.EncodeBytes(data);
    }

    public List<TypedValue> Decode(string s)
    {
        var data = _radix.DecodeBytes(s);
        return XidCodec.DecodeBinary(this, data);
    }

    private void FinalizeLayout()
    {
        var variantBits = MaxVariants <= 1 ? 1 : BitLen(MaxVariants - 1);
        var countBits = MaxObjects <= 1 ? 1 : BitLen(MaxObjects);
        var total = CheckBits + countBits + variantBits;
        if (total > 16)
            throw new ArgumentException($"header layout exceeds 16 bits: {total}");
        CountBits = countBits;
        VariantBits = variantBits;
        CheckMask = (1 << CheckBits) - 1;
        CountMask = ((1 << countBits) - 1) << CheckBits;
        VariantMask = ((1 << variantBits) - 1) << (CheckBits + countBits);
        CountShift = CheckBits;
        VariantShift = CheckBits + countBits;
    }

    private static int BitLen(int n)
    {
        if (n <= 0) return 1;
        return 32 - BitOperations.LeadingZeroCount((uint)n);
    }
}
