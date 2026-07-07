namespace Vanni.Idmix;

/// <summary>IDX v1.2 编解码器：组合 Idx 二进制层与可插拔文本 Codec。</summary>
public sealed class IdMix
{
    public const string DefaultAlphabet =
        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    private readonly Idx _idx;
    private readonly ICodec _codec;
    private readonly Random _random = new();

    public IdMix(Idx idx, ICodec codec)
    {
        _idx = idx ?? throw new ArgumentNullException(nameof(idx));
        _codec = codec ?? throw new ArgumentNullException(nameof(codec));
    }

    /// <summary>创建 IdMix 实例（默认 Idx + 默认 RadixCodec）。</summary>
    public static IdMix Create() => new(Idx.Create(), new RadixCodec(DefaultAlphabet));

    /// <summary>使用配置委托创建 IdMix 实例。</summary>
    public static IdMix Create(Action<IdMixBuilder> configure)
    {
        var b = new IdMixBuilder();
        configure(b);
        return b.Build();
    }

    public static IdMix NewDefault() => Create();

    public Idx Idx => _idx;
    public ICodec Codec => _codec;

    public string Encode(params object[] values)
    {
        if (values.Length < 1) throw new ArgumentException("at least one value is required");
        var variantId = _random.Next(_idx.MaxVariants);
        var data = EncodeBinary(values, variantId);
        return _codec.Encode(data);
    }

    public string EncodeWithVariant(int variantId, params object[] values)
    {
        var data = EncodeBinary(values, variantId);
        return _codec.Encode(data);
    }

    public object[] Decode(string s)
    {
        var data = _codec.Decode(s);
        return _idx.Decode(data);
    }

    private byte[] EncodeBinary(object[] values, int variantId)
    {
        var objects = Number.NormalizeObjects(values);
        return _idx.EncodeBinary(objects, variantId);
    }

    /// <summary>IdMix 配置构建器。</summary>
    public sealed class IdMixBuilder
    {
        private Idx _idx = Idx.Create();
        private ICodec _codec = new RadixCodec(DefaultAlphabet);

        public IdMixBuilder WithIdx(Idx idx)
        {
            _idx = idx ?? throw new ArgumentException("idx cannot be nil");
            return this;
        }

        public IdMixBuilder WithCodec(ICodec codec)
        {
            _codec = codec ?? throw new ArgumentException("codec cannot be nil");
            return this;
        }

        public IdMixBuilder WithAlphabet(string alphabet)
        {
            _codec = new RadixCodec(alphabet);
            return this;
        }

        public IdMix Build() => new(_idx, _codec);
    }
}
