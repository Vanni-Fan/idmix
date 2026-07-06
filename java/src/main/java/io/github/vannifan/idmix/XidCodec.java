package io.github.vannifan.idmix;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.util.ArrayList;
import java.util.List;

/**
 * XID v1.1 binary layer codec.
 */
public final class XidCodec {
    private static final int[] SW_BYTES = {1, 2, 4, 8};
    private static final int[][] EMBEDDED_OTYPE = {
            {TypedValue.OTYPE_UINT8, TypedValue.OTYPE_UINT16, TypedValue.OTYPE_UINT32, TypedValue.OTYPE_UINT64},
            {TypedValue.OTYPE_INT8, TypedValue.OTYPE_INT16, TypedValue.OTYPE_INT32, TypedValue.OTYPE_INT64},
    };

    private XidCodec() {}

    public static byte[] encodeBinary(IdMix m, List<TypedValue> typed, int variantId) {
        ByteBuffer objects = ByteBuffer.allocate(typed.size() * 9);
        for (TypedValue tv : typed) {
            byte[] obj = encodeObject(tv);
            objects.put(obj);
        }
        objects.flip();
        byte[] objBytes = new byte[objects.remaining()];
        objects.get(objBytes);

        int mask = (variantId * 0x9D + 0x37) & 0xFF;
        for (int i = 0; i < objBytes.length; i++) objBytes[i] ^= (byte) mask;

        int count = typed.size();
        int header = (variantId << m.variantShift) | (count << m.countShift);
        ByteBuffer data = ByteBuffer.allocate(2 + objBytes.length).order(ByteOrder.LITTLE_ENDIAN);
        data.putShort((short) header);
        data.put(objBytes);

        byte[] arr = data.array();
        int xorSum = 0;
        for (byte b : arr) xorSum ^= (b & 0xff);
        header |= (xorSum & m.checkMask);
        arr[0] = (byte) (header & 0xff);
        arr[1] = (byte) ((header >> 8) & 0xff);
        return arr;
    }

    public static List<TypedValue> decodeBinary(IdMix m, byte[] data) {
        if (data.length < 2) throw new IllegalArgumentException("invalid data: too short");
        int header = (data[0] & 0xff) | ((data[1] & 0xff) << 8);
        int check = header & m.checkMask;
        int count = (header & m.countMask) >> m.countShift;
        int variantId = (header & m.variantMask) >> m.variantShift;

        if (variantId >= m.maxVariants) throw new IllegalArgumentException("invalid variant_id " + variantId);
        if (count > m.maxObjects) throw new IllegalArgumentException("invalid count " + count);

        byte[] verify = data.clone();
        verify[0] &= ~(byte) m.checkMask;
        int xorSum = 0;
        for (byte b : verify) xorSum ^= (b & 0xff);
        if ((xorSum & m.checkMask) != check) throw new IllegalArgumentException("checksum mismatch");

        byte[] objects = new byte[data.length - 2];
        System.arraycopy(data, 2, objects, 0, objects.length);
        int mask = (variantId * 0x9D + 0x37) & 0xFF;
        for (int i = 0; i < objects.length; i++) objects[i] ^= (byte) mask;

        List<TypedValue> result = new ArrayList<>();
        int pos = 0;
        for (int i = 0; i < count; i++) {
            if (pos >= objects.length) throw new IllegalArgumentException("premature end of data");
            DecodeResult dr = decodeObject(objects, pos);
            result.add(dr.tv);
            pos += dr.consumed;
        }
        if (pos != objects.length) throw new IllegalArgumentException("extra bytes after data objects");
        return result;
    }

    private static byte[] encodeObject(TypedValue tv) {
        validateRange(tv.otype, tv.val);
        Byte embedded = tryEmbeddedHead(tv.otype, tv.val);
        if (embedded != null) return new byte[]{embedded};
        long[] magNeg = magnitudeFromTyped(tv.otype, tv.val);
        long mag = magNeg[0];
        boolean neg = magNeg[1] != 0;
        int sw = swFromMagnitude(mag);
        byte[] payload = uintToLEBytes(mag, SW_BYTES[sw]);
        int head = 0x80 | (sw << 4) | tv.otype;
        if (neg) head |= 1 << 6;
        byte[] out = new byte[1 + payload.length];
        out[0] = (byte) head;
        System.arraycopy(payload, 0, out, 1, payload.length);
        return out;
    }

    private record DecodeResult(TypedValue tv, int consumed) {}

    private static DecodeResult decodeObject(byte[] data, int offset) {
        if (offset >= data.length) throw new IllegalArgumentException("truncated object header");
        int head = data[offset] & 0xff;
        if ((head & 0x80) == 0) {
            int sign = (head >> 6) & 1;
            int wb = (head >> 4) & 0x03;
            int v = head & 0x0F;
            int otype = EMBEDDED_OTYPE[sign][wb];
            long val = sign == 0 ? v : -v - 1L;
            return new DecodeResult(new TypedValue(otype, val), 1);
        }
        int sw = (head >> 4) & 0x03;
        int otype = head & 0x0F;
        if (otype > TypedValue.OTYPE_INT64) throw new IllegalArgumentException("invalid otype " + otype);
        int numBytes = SW_BYTES[sw];
        if (data.length < offset + 1 + numBytes) throw new IllegalArgumentException("truncated object payload");
        long mag = 0;
        for (int i = 0; i < numBytes; i++) mag |= (long) (data[offset + 1 + i] & 0xff) << (8 * i);
        boolean neg = ((head >> 6) & 1) != 0;
        long val = valueFromMagnitude(mag, neg);
        validateRange(otype, val);
        return new DecodeResult(new TypedValue(otype, val), 1 + numBytes);
    }

    private static boolean isUnsigned(int otype) { return otype <= TypedValue.OTYPE_UINT64; }
    private static boolean isSigned(int otype) { return otype >= TypedValue.OTYPE_INT8; }

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

    private static long valueFromMagnitude(long mag, boolean neg) {
        if (!neg) return mag;
        if (mag == 1L << 63) return Long.MIN_VALUE;
        return -mag;
    }

    private static byte[] uintToLEBytes(long v, int size) {
        byte[] buf = new byte[size];
        for (int i = 0; i < size; i++) buf[i] = (byte) ((v >>> (8 * i)) & 0xFF);
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
}
