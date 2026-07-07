package io.github.vannifan.idmix;

import java.io.ByteArrayOutputStream;
import java.util.ArrayList;
import java.util.List;

/** IDX v1.2 binary codec; usable independently of the idmix text layer. */
public final class Idx {
    private static final int[] SW_BYTES = {1, 2, 4, 8};
    private static final int[][] EMBEDDED_OTYPE = {
            {TypedValue.OTYPE_UINT8, TypedValue.OTYPE_UINT16, TypedValue.OTYPE_UINT32, TypedValue.OTYPE_UINT64},
            {TypedValue.OTYPE_INT8, TypedValue.OTYPE_INT16, TypedValue.OTYPE_INT32, TypedValue.OTYPE_INT64},
    };

    public final int maxObjects;
    public final int maxVariants;
    public final int checkBits;
    final int checkMask;

    private Idx(int maxObjects, int maxVariants, int checkBits) {
        this.maxObjects = maxObjects;
        this.maxVariants = maxVariants;
        this.checkBits = checkBits;
        this.checkMask = (1 << checkBits) - 1;
    }

    public static Idx create() {
        return new Idx(255, 32, 2);
    }

    public static Idx create(IdxBuilder builder) {
        return builder.build();
    }

    public byte[] encode(Object... values) {
        if (values.length < 1) throw new IllegalArgumentException("at least one value is required");
        if (values.length > maxObjects) {
            throw new IllegalArgumentException("too many objects: " + values.length + " (max " + maxObjects + ")");
        }
        return encodeBinary(Number.normalizeObjects(values), 0);
    }

    public byte[] encodeWithVariant(int variantId, Object... values) {
        if (values.length < 1) throw new IllegalArgumentException("at least one value is required");
        if (values.length > maxObjects) {
            throw new IllegalArgumentException("too many objects: " + values.length + " (max " + maxObjects + ")");
        }
        return encodeBinary(Number.normalizeObjects(values), variantId);
    }

    public Object[] decode(byte[] data) {
        return Number.materializeObjects(decodeBinary(data));
    }

    byte[] encodeBinary(Number.DataObject[] objects, int variantId) {
        if (variantId < 0 || variantId >= maxVariants) {
            throw new IllegalArgumentException("invalid variant_id " + variantId + " (max " + (maxVariants - 1) + ")");
        }

        ByteArrayOutputStream stream = new ByteArrayOutputStream(objects.length * 9);
        for (Number.DataObject obj : objects) {
            byte[] ob = encodeObject(obj);
            stream.write(ob, 0, ob.length);
        }
        byte[] objBytes = stream.toByteArray();

        int mask = (variantId * 0x9D + 0x37) & 0xFF;
        for (int i = 0; i < objBytes.length; i++) objBytes[i] ^= (byte) mask;

        int count = objects.length;
        int headerLen = count == 1 ? 1 : 2;
        byte[] data = new byte[headerLen + objBytes.length];
        if (count == 1) {
            data[0] = (byte) (variantId << checkBits);
        } else {
            data[0] = (byte) (0x80 | (variantId << checkBits));
            data[1] = (byte) count;
        }
        System.arraycopy(objBytes, 0, data, headerLen, objBytes.length);

        int xorSum = 0;
        for (byte b : data) xorSum ^= (b & 0xff);
        data[0] |= (byte) (xorSum & checkMask);
        return data;
    }

    Number.DataObject[] decodeBinary(byte[] data) {
        if (data.length < 1) throw new IllegalArgumentException("invalid data: too short");

        int byte0 = data[0] & 0xff;
        int check = byte0 & checkMask;
        boolean multi = (byte0 & 0x80) != 0;
        int variantId = (byte0 & 0x7F) >> checkBits;

        if (variantId >= maxVariants) {
            throw new IllegalArgumentException("invalid variant_id " + variantId + " (max " + (maxVariants - 1) + ")");
        }

        int headerLen = 1;
        int count = 1;
        if (multi) {
            if (data.length < 2) throw new IllegalArgumentException("invalid data: missing count byte");
            headerLen = 2;
            count = data[1] & 0xff;
            if (count < 2 || count > maxObjects) {
                throw new IllegalArgumentException("invalid count " + count);
            }
        }

        byte[] verify = data.clone();
        verify[0] &= ~(byte) checkMask;
        int xorSum = 0;
        for (byte b : verify) xorSum ^= (b & 0xff);
        if ((xorSum & checkMask) != check) throw new IllegalArgumentException("checksum mismatch");

        byte[] objData = new byte[data.length - headerLen];
        System.arraycopy(data, headerLen, objData, 0, objData.length);
        int mask = (variantId * 0x9D + 0x37) & 0xFF;
        for (int i = 0; i < objData.length; i++) objData[i] ^= (byte) mask;

        List<Number.DataObject> result = new ArrayList<>();
        int pos = 0;
        for (int i = 0; i < count; i++) {
            if (pos >= objData.length) throw new IllegalArgumentException("premature end of data");
            DecodeResult dr = decodeObject(objData, pos);
            result.add(dr.obj);
            pos += dr.consumed;
        }
        if (pos != objData.length) throw new IllegalArgumentException("extra bytes after data objects");
        return result.toArray(new Number.DataObject[0]);
    }

    private static byte[] encodeObject(Number.DataObject obj) {
        if (obj.isString) {
            int n = obj.str.length;
            if (n < 1 || n > Number.MAX_STRING_LEN) {
                throw new IllegalArgumentException("string length " + n + " out of range [1, " + Number.MAX_STRING_LEN + "]");
            }
            byte[] out = new byte[1 + n];
            out[0] = (byte) (0xC0 | n);
            System.arraycopy(obj.str, 0, out, 1, n);
            return out;
        }

        validateRange(obj.otype, obj.val);
        Byte embedded = tryEmbeddedHead(obj.otype, obj.val);
        if (embedded != null) return new byte[]{embedded};

        int[] swPayload = payloadForNumber(obj.otype, obj.val);
        int sw = swPayload[0];
        byte[] payload = new byte[swPayload.length - 1];
        for (int i = 0; i < payload.length; i++) payload[i] = (byte) swPayload[i + 1];
        int head = 0x80 | (sw << 4) | obj.otype;
        byte[] out = new byte[1 + payload.length];
        out[0] = (byte) head;
        System.arraycopy(payload, 0, out, 1, payload.length);
        return out;
    }

    private record DecodeResult(Number.DataObject obj, int consumed) {}

    private static DecodeResult decodeObject(byte[] data, int offset) {
        if (offset >= data.length) throw new IllegalArgumentException("truncated object header");
        int head = data[offset] & 0xff;
        if ((head & 0x80) == 0) {
            int sign = (head >> 6) & 1;
            int wb = (head >> 4) & 0x03;
            int v = head & 0x0F;
            int otype = EMBEDDED_OTYPE[sign][wb];
            long val = sign == 0 ? v : -v - 1L;
            return new DecodeResult(Number.DataObject.integer(otype, val), 1);
        }

        if ((head & 0x40) != 0) {
            int n = head & 0x3F;
            if (n < 1 || n > Number.MAX_STRING_LEN) {
                throw new IllegalArgumentException("invalid string length " + n);
            }
            if (data.length < offset + 1 + n) {
                throw new IllegalArgumentException("truncated string payload");
            }
            byte[] str = new byte[n];
            System.arraycopy(data, offset + 1, str, 0, n);
            return new DecodeResult(Number.DataObject.string(str), 1 + n);
        }

        int sw = (head >> 4) & 0x03;
        int otype = head & 0x0F;
        if (otype > TypedValue.OTYPE_INT64) throw new IllegalArgumentException("invalid otype " + otype);
        int numBytes = SW_BYTES[sw];
        if (data.length < offset + 1 + numBytes) throw new IllegalArgumentException("truncated object payload");
        long val = valueFromPayload(otype, data, offset + 1, numBytes);
        validateRange(otype, val);
        return new DecodeResult(Number.DataObject.integer(otype, val), 1 + numBytes);
    }

    /** Returns [sw, ...payload bytes]. */
    private static int[] payloadForNumber(int otype, long val) {
        if (otype == TypedValue.OTYPE_UINT64) {
            int sw = swFromMagnitude(val);
            byte[] payload = uintToLEBytes(val, SW_BYTES[sw]);
            int[] out = new int[1 + payload.length];
            out[0] = sw;
            for (int i = 0; i < payload.length; i++) out[i + 1] = payload[i] & 0xff;
            return out;
        }
        if (isUnsigned(otype)) {
            if (val < 0) throw new IllegalArgumentException("negative value " + val + " for unsigned otype " + otype);
            long mag = val;
            int sw = swFromMagnitude(mag);
            byte[] payload = uintToLEBytes(mag, SW_BYTES[sw]);
            int[] out = new int[1 + payload.length];
            out[0] = sw;
            for (int i = 0; i < payload.length; i++) out[i + 1] = payload[i] & 0xff;
            return out;
        }
        int sw = swFromSignedValue(val);
        byte[] payload = signedToLEBytes(val, SW_BYTES[sw]);
        int[] out = new int[1 + payload.length];
        out[0] = sw;
        for (int i = 0; i < payload.length; i++) out[i + 1] = payload[i] & 0xff;
        return out;
    }

    private static long valueFromPayload(int otype, byte[] data, int offset, int numBytes) {
        if (isUnsigned(otype)) {
            long mag = leBytesToUint(data, offset, numBytes);
            if (otype != TypedValue.OTYPE_UINT64 && Long.compareUnsigned(mag, Long.MAX_VALUE) > 0) {
                throw new IllegalArgumentException("value out of range for otype " + otype);
            }
            return mag;
        }
        return leBytesToSigned(data, offset, numBytes);
    }

    private static int swFromSignedValue(long val) {
        if (val >= Byte.MIN_VALUE && val <= Byte.MAX_VALUE) return 0;
        if (val >= Short.MIN_VALUE && val <= Short.MAX_VALUE) return 1;
        if (val >= Integer.MIN_VALUE && val <= Integer.MAX_VALUE) return 2;
        return 3;
    }

    private static byte[] signedToLEBytes(long val, int size) {
        byte[] buf = new byte[size];
        for (int i = 0; i < size; i++) buf[i] = (byte) (val >>> (8 * i));
        return buf;
    }

    private static long leBytesToSigned(byte[] data, int offset, int size) {
        long u = 0;
        for (int i = 0; i < size; i++) u |= (long) (data[offset + i] & 0xff) << (8 * i);
        int shift = 64 - size * 8;
        return (u << shift) >> shift;
    }

    private static long leBytesToUint(byte[] data, int offset, int size) {
        long u = 0;
        for (int i = 0; i < size; i++) u |= (long) (data[offset + i] & 0xff) << (8 * i);
        return u;
    }

    private static boolean isUnsigned(int otype) { return otype <= TypedValue.OTYPE_UINT64; }

    private static int widthBits(int otype) {
        return switch (otype) {
            case TypedValue.OTYPE_UINT8, TypedValue.OTYPE_INT8 -> 0;
            case TypedValue.OTYPE_UINT16, TypedValue.OTYPE_INT16 -> 1;
            case TypedValue.OTYPE_UINT32, TypedValue.OTYPE_INT32 -> 2;
            default -> 3;
        };
    }

    private static long[] magnitudeFromTyped(int otype, long val) {
        if (isUnsigned(otype)) return new long[]{val, 0};
        if (val < 0) return new long[]{-val, 1};
        return new long[]{val, 0};
    }

    private static int swFromMagnitude(long mag) {
        if (Long.compareUnsigned(mag, 255) <= 0) return 0;
        if (Long.compareUnsigned(mag, 65535) <= 0) return 1;
        if (Long.compareUnsigned(mag, 4294967295L) <= 0) return 2;
        return 3;
    }

    private static Byte tryEmbeddedHead(int otype, long val) {
        long[] magNeg = magnitudeFromTyped(otype, val);
        long mag = magNeg[0];
        boolean neg = magNeg[1] != 0;
        if (Long.compareUnsigned(mag, 17) >= 0) return null;
        int wb = widthBits(otype);
        if (Long.compareUnsigned(mag, 16) == 0) {
            if (neg) return (byte) ((1 << 6) | (wb << 4) | 15);
            return null;
        }
        if (neg) return (byte) ((1 << 6) | (wb << 4) | (int) (mag - 1));
        return (byte) ((wb << 4) | (int) mag);
    }

    private static byte[] uintToLEBytes(long v, int size) {
        byte[] buf = new byte[size];
        for (int i = 0; i < size; i++) buf[i] = (byte) (v >>> (8 * i));
        return buf;
    }

    private static void validateRange(int otype, long val) {
        boolean ok = switch (otype) {
            case TypedValue.OTYPE_UINT8 -> val >= 0 && val <= 0xFF;
            case TypedValue.OTYPE_UINT16 -> val >= 0 && val <= 0xFFFF;
            case TypedValue.OTYPE_UINT32 -> val >= 0 && val <= 0xFFFFFFFFL;
            case TypedValue.OTYPE_UINT64 -> true;
            case TypedValue.OTYPE_INT8 -> val >= -128 && val <= 127;
            case TypedValue.OTYPE_INT16 -> val >= -32768 && val <= 32767;
            case TypedValue.OTYPE_INT32 -> val >= Integer.MIN_VALUE && val <= Integer.MAX_VALUE;
            case TypedValue.OTYPE_INT64 -> true;
            default -> false;
        };
        if (!ok) throw new IllegalArgumentException("value " + val + " out of range for otype " + otype);
    }

    /** Idx configuration builder. */
    public static final class IdxBuilder {
        public int maxObjects = 255;
        public int maxVariants = 32;
        public int checkBits = 2;

        public Idx build() {
            if (maxObjects < 1 || maxObjects > 255) {
                throw new IllegalArgumentException("maxObjects must be between 1 and 255");
            }
            if (maxVariants < 1 || maxVariants > 32) {
                throw new IllegalArgumentException("maxVariants must be between 1 and 32");
            }
            if (checkBits < 1 || checkBits > 2) {
                throw new IllegalArgumentException("checkBits must be 1 or 2");
            }
            return new Idx(maxObjects, maxVariants, checkBits);
        }
    }
}
