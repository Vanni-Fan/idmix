package io.github.vannifan.idmix;

import java.util.Arrays;
import java.util.List;
import java.util.Random;

/** XID v1.1 codec entry point. */
public final class IdMix {
    public static final String DEFAULT_ALPHABET =
            "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    private final RadixCodec radix;
    public final int maxObjects;
    public final int maxVariants;
    public final int checkBits;
    public int countBits;
    public int variantBits;
    public int checkMask;
    public int countMask;
    public int variantMask;
    public int countShift;
    public int variantShift;

    private final Random random = new Random();

    public IdMix() {
        this(DEFAULT_ALPHABET, 511, 32, 2);
    }

    public IdMix(String alphabet) {
        this(alphabet, 511, 32, 2);
    }

    public IdMix(String alphabet, int maxObjects, int maxVariants, int checkBits) {
        this.radix = new RadixCodec(alphabet);
        this.maxObjects = maxObjects;
        this.maxVariants = maxVariants;
        this.checkBits = checkBits;
        finalizeLayout();
    }

    public static IdMix newDefault() {
        return new IdMix();
    }

    public String encode(TypedValue... values) {
        if (values.length < 1) throw new IllegalArgumentException("at least one value is required");
        if (values.length > maxObjects) {
            throw new IllegalArgumentException("too many objects: " + values.length);
        }
        int variantId = random.nextInt(maxVariants);
        byte[] data = XidCodec.encodeBinary(this, Arrays.asList(values), variantId);
        return radix.encodeBytes(data);
    }

    public List<TypedValue> decode(String s) {
        byte[] data = radix.decodeBytes(s);
        return XidCodec.decodeBinary(this, data);
    }

    public RadixCodec getRadix() { return radix; }

    private void finalizeLayout() {
        int variantBits = maxVariants <= 1 ? 1 : bitLen(maxVariants - 1);
        int countBits = maxObjects <= 1 ? 1 : bitLen(maxObjects);
        int total = checkBits + countBits + variantBits;
        if (total > 16) {
            throw new IllegalArgumentException("header layout exceeds 16 bits: " + total);
        }
        this.countBits = countBits;
        this.variantBits = variantBits;
        this.checkMask = (1 << checkBits) - 1;
        this.countMask = ((1 << countBits) - 1) << checkBits;
        this.variantMask = ((1 << variantBits) - 1) << (checkBits + countBits);
        this.countShift = checkBits;
        this.variantShift = checkBits + countBits;
    }

    private static int bitLen(int n) {
        if (n <= 0) return 1;
        return 32 - Integer.numberOfLeadingZeros(n);
    }
}
