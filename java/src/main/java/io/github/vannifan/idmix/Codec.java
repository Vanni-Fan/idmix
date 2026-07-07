package io.github.vannifan.idmix;

import java.util.Base64;
import java.util.function.Function;

/** Package-level text codec helpers and built-in implementations. */
public final class Codec {
    private static final ICodec DEFAULT_CODEC = new RadixCodec(IdMix.DEFAULT_ALPHABET);

    private Codec() {}

    public static String encodeBytes(byte[] data) {
        return encodeBytes(data, null);
    }

    public static String encodeBytes(byte[] data, ICodec codec) {
        return (codec != null ? codec : DEFAULT_CODEC).encode(data);
    }

    public static byte[] decodeString(String s) {
        return decodeString(s, null);
    }

    public static byte[] decodeString(String s, ICodec codec) {
        return (codec != null ? codec : DEFAULT_CODEC).decode(s);
    }

    /** Standard Base64 codec. */
    public static final class Base64Codec implements ICodec {
        public static final Base64Codec INSTANCE = new Base64Codec();

        private Base64Codec() {}

        @Override
        public String encode(byte[] data) {
            return Base64.getEncoder().encodeToString(data);
        }

        @Override
        public byte[] decode(String s) {
            return Base64.getDecoder().decode(s);
        }
    }

    /** Function-based codec for custom encryption/XOR logic. */
    public static final class FuncCodec implements ICodec {
        private final Function<byte[], String> encodeFn;
        private final Function<String, byte[]> decodeFn;

        public FuncCodec(Function<byte[], String> encodeFn, Function<String, byte[]> decodeFn) {
            this.encodeFn = encodeFn;
            this.decodeFn = decodeFn;
        }

        @Override
        public String encode(byte[] data) {
            if (encodeFn == null) throw new IllegalStateException("codec function is nil");
            return encodeFn.apply(data);
        }

        @Override
        public byte[] decode(String s) {
            if (decodeFn == null) throw new IllegalStateException("codec function is nil");
            return decodeFn.apply(s);
        }
    }
}
