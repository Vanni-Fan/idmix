using System.Numerics;
using System.Text;

namespace Vanni.Idmix;

/// <summary>基于自定义字符表的 Base-N 编解码器（默认 idmix 文本层）。</summary>
public sealed class RadixCodec : ICodec
{
    private readonly int _base;
    private readonly string _chars;
    private readonly Dictionary<char, int> _fromCustom;

    public RadixCodec(string alphabet)
    {
        if (alphabet.Length < 2)
            throw new ArgumentException("alphabet must have at least 2 unique characters", nameof(alphabet));
        _base = alphabet.Length;
        _chars = alphabet;
        _fromCustom = new Dictionary<char, int>();
        for (var i = 0; i < alphabet.Length; i++)
        {
            var c = alphabet[i];
            if (_fromCustom.ContainsKey(c))
                throw new ArgumentException($"alphabet contains duplicate character {c}", nameof(alphabet));
            _fromCustom[c] = i;
        }
    }

    public string Alphabet => _chars;
    public int Base => _base;

    public string Encode(byte[] data)
    {
        if (data.Length == 0) return _chars[0].ToString();
        var buf = new byte[2 + data.Length];
        buf[0] = (byte)((data.Length >> 8) & 0xFF);
        buf[1] = (byte)(data.Length & 0xFF);
        Buffer.BlockCopy(data, 0, buf, 2, data.Length);
        var n = new BigInteger(buf, isUnsigned: true, isBigEndian: true);
        return IntToString(n);
    }

    public byte[] Decode(string s)
    {
        if (string.IsNullOrEmpty(s))
            throw new ArgumentException("empty string", nameof(s));
        var n = StringToInt(s);
        var raw = n.ToByteArray(isUnsigned: true, isBigEndian: true);
        for (var pad = 0; pad <= 1; pad++)
        {
            var buf = new byte[pad + raw.Length];
            Buffer.BlockCopy(raw, 0, buf, pad, raw.Length);
            if (buf.Length < 2) continue;
            var dataLen = (buf[0] << 8) | buf[1];
            if (buf.Length != 2 + dataLen) continue;
            var outBuf = new byte[dataLen];
            Buffer.BlockCopy(buf, 2, outBuf, 0, dataLen);
            return outBuf;
        }
        throw new ArgumentException("invalid encoded data length");
    }

    private string IntToString(BigInteger n)
    {
        if (n.IsZero) return _chars[0].ToString();
        var baseBi = new BigInteger(_base);
        var sb = new StringBuilder();
        while (n > BigInteger.Zero)
        {
            n = BigInteger.DivRem(n, baseBi, out var rem);
            sb.Append(_chars[(int)rem]);
        }
        var chars = sb.ToString().ToCharArray();
        Array.Reverse(chars);
        return new string(chars);
    }

    private BigInteger StringToInt(string s)
    {
        var n = BigInteger.Zero;
        var baseBi = new BigInteger(_base);
        foreach (var c in s)
        {
            if (!_fromCustom.TryGetValue(c, out var idx))
                throw new ArgumentException($"invalid character {c}", nameof(s));
            n = n * baseBi + idx;
        }
        return n;
    }
}
