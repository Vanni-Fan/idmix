package io.github.vannifan.idmix;

/** Binary↔text codec (pluggable idmix text layer). */
public interface ICodec {
    String encode(byte[] data);
    byte[] decode(String s);
}
