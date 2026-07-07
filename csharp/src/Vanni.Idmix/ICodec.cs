namespace Vanni.Idmix;

/// <summary>二进制与文本之间的可插拔编解码器（idmix 文本层插拔点）。</summary>
public interface ICodec
{
    string Encode(byte[] data);
    byte[] Decode(string s);
}
