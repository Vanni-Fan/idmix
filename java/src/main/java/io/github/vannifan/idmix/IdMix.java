package io.github.vannifan.idmix;

import java.util.Random;

/** IDX v1.2 codec: combines Idx binary layer with a pluggable text codec. */
public final class IdMix {
    public static final String DEFAULT_ALPHABET =
            "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    private final Idx idx;
    private final ICodec codec;
    private final Random random = new Random();

    public IdMix(Idx idx, ICodec codec) {
        if (idx == null) throw new IllegalArgumentException("idx cannot be nil");
        if (codec == null) throw new IllegalArgumentException("codec cannot be nil");
        this.idx = idx;
        this.codec = codec;
    }

    public static IdMix create() {
        return new IdMix(Idx.create(), new RadixCodec(DEFAULT_ALPHABET));
    }

    public static IdMix create(IdMixBuilder builder) {
        return builder.build();
    }

    public static IdMix newDefault() {
        return create();
    }

    public Idx getIdx() { return idx; }
    public ICodec getCodec() { return codec; }

    public String encode(Object... values) {
        if (values.length < 1) throw new IllegalArgumentException("at least one value is required");
        int variantId = random.nextInt(idx.maxVariants);
        return codec.encode(encodeBinary(values, variantId));
    }

    public String encodeWithVariant(int variantId, Object... values) {
        return codec.encode(encodeBinary(values, variantId));
    }

    public Object[] decode(String s) {
        return idx.decode(codec.decode(s));
    }

    private byte[] encodeBinary(Object[] values, int variantId) {
        return idx.encodeBinary(Number.normalizeObjects(values), variantId);
    }

    /** IdMix configuration builder. */
    public static final class IdMixBuilder {
        private Idx idx = Idx.create();
        private ICodec codec = new RadixCodec(DEFAULT_ALPHABET);

        public IdMixBuilder withIdx(Idx idx) {
            if (idx == null) throw new IllegalArgumentException("idx cannot be nil");
            this.idx = idx;
            return this;
        }

        public IdMixBuilder withCodec(ICodec codec) {
            if (codec == null) throw new IllegalArgumentException("codec cannot be nil");
            this.codec = codec;
            return this;
        }

        public IdMixBuilder withAlphabet(String alphabet) {
            this.codec = new RadixCodec(alphabet);
            return this;
        }

        public IdMix build() {
            return new IdMix(idx, codec);
        }
    }
}
