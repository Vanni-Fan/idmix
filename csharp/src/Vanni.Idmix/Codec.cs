using System.Text;

namespace Vanni.Idmix;

/// <summary>包级文本编解码辅助与内置 Codec 实现。</summary>
public static class Codec
{
    private static readonly Lazy<ICodec> DefaultCodec = new(() => new RadixCodec(IdMix.DefaultAlphabet));

    /// <summary>将任意二进制编码为文本；codec 不传时使用默认 RadixCodec。</summary>
    public static string EncodeBytes(byte[] data, ICodec? codec = null) =>
        (codec ?? DefaultCodec.Value).Encode(data);

    /// <summary>将文本还原为二进制；codec 不传时使用默认 RadixCodec。</summary>
    public static byte[] DecodeString(string s, ICodec? codec = null) =>
        (codec ?? DefaultCodec.Value).Decode(s);
}

/// <summary>标准 Base64 编解码器。</summary>
public sealed class Base64Codec : ICodec
{
    public static Base64Codec Instance { get; } = new();

    public string Encode(byte[] data) => Convert.ToBase64String(data);

    public byte[] Decode(string s) => Convert.FromBase64String(s);
}

/// <summary>由函数实现的 Codec，便于包装 AES/XOR 等自定义逻辑。</summary>
public sealed class FuncCodec : ICodec
{
    public Func<byte[], string>? EncodeFn { get; init; }
    public Func<string, byte[]>? DecodeFn { get; init; }

    public string Encode(byte[] data)
    {
        if (EncodeFn == null) throw new InvalidOperationException("codec function is nil");
        return EncodeFn(data);
    }

    public byte[] Decode(string s)
    {
        if (DecodeFn == null) throw new InvalidOperationException("codec function is nil");
        return DecodeFn(s);
    }
}
