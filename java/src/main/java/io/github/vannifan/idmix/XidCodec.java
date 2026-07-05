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
        if (isUnsigned(tv.otype) && tv.val >= 0 && tv.val <= 15) {
            int wb = widthBits(tv.otype);
            return new byte[]{(byte) ((wb << 4) | tv.val)};
        }
        if (isSigned(tv.otype) && tv.val >= -16 && tv.val <= -1) {
            int wb = widthBits(tv.otype);
            int v = (int) (-tv.val - 1);
            return new byte[]{(byte) ((1 << 6) | (wb << 4) | v)};
        }
        int[] swPayload = minimalComplementBytes(tv.otype, tv.val);
        int sw = swPayload[0];
        byte[] payload = new byte[swPayload.length - 1];
        for (int i = 0; i < payload.length; i++) payload[i] = (byte) swPayload[i + 1];
        byte[] out = new byte[1 + payload.length];
        out[0] = (byte) (0x80 | (sw << 4) | tv.otype);
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
        if (((head >> 6) & 1) != 0) throw new IllegalArgumentException("reserved bit set in extended mode");
        int sw = (head >> 4) & 0x03;
        int otype = head & 0x0F;
        if (otype > TypedValue.OTYPE_INT64) throw new IllegalArgumentException("invalid otype " + otype);
        int numBytes = SW_BYTES[sw];
        if (data.length < offset + 1 + numBytes) throw new IllegalArgumentException("truncated object payload");
        long raw = 0;
        for (int i = 0; i < numBytes; i++) raw |= (long) (data[offset + 1 + i] & 0xff) << (8 * i);
        long val = reconstructInt(otype, sw, raw);
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

    private static int targetBits(int otype) {
        return switch (otype) {
            case TypedValue.OTYPE_UINT8, TypedValue.OTYPE_INT8 -> 8;
            case TypedValue.OTYPE_UINT16, TypedValue.OTYPE_INT16 -> 16;
            case TypedValue.OTYPE_UINT32, TypedValue.OTYPE_INT32 -> 32;
            default -> 64;
        };
    }

    /** Returns [sw, ...payload bytes] */
    private static int[] minimalComplementBytes(int otype, long val) {
        if (val == 0) return new int[]{0, 0};
        if (isUnsigned(otype)) {
            if (val < 0) throw new IllegalArgumentException("negative value for unsigned type");
            for (int sw = 0; sw < 4; sw++) {
                int size = SW_BYTES[sw];
                if (size < 8 && val >= (1L << (size * 8))) continue;
                int[] buf = uintToLEBytes(val, size);
                if ((buf[size - 1] & 0x80) == 0) {
                    int[] out = new int[1 + size];
                    out[0] = sw;
                    System.arraycopy(buf, 0, out, 1, size);
                    return out;
                }
            }
            throw new IllegalArgumentException("value too large for unsigned type");
        }
        int tbits = targetBits(otype);
        long mask = tbits == 64 ? -1L : (1L << tbits) - 1;
        long uval = val & mask;

        if (val < 0) {
            for (int sw = 0; sw < 4; sw++) {
                int size = SW_BYTES[sw];
                int shift = size * 8;
                if (shift >= tbits) {
                    int[] buf = uintToLEBytes(uval, size);
                    int[] out = new int[1 + size];
                    out[0] = sw;
                    System.arraycopy(buf, 0, out, 1, size);
                    return out;
                }
                long lower = uval & ((1L << shift) - 1);
                long upper = uval >> shift;
                long upperMask = (1L << (tbits - shift)) - 1;
                if (upper != upperMask) continue;
                int highByte = (int) ((lower >> (shift - 8)) & 0xFF);
                if ((highByte & 0x80) == 0) continue;
                int[] buf = uintToLEBytes(lower, size);
                int[] out = new int[1 + size];
                out[0] = sw;
                System.arraycopy(buf, 0, out, 1, size);
                return out;
            }
        } else {
            for (int sw = 0; sw < 4; sw++) {
                int size = SW_BYTES[sw];
                if (size < 8 && uval >= (1L << (size * 8))) continue;
                int[] buf = uintToLEBytes(uval, size);
                if ((buf[size - 1] & 0x80) == 0) {
                    int[] out = new int[1 + size];
                    out[0] = sw;
                    System.arraycopy(buf, 0, out, 1, size);
                    return out;
                }
            }
        }
        int sw = switch (tbits) { case 8 -> 0; case 16 -> 1; case 32 -> 2; default -> 3; };
        int[] buf = uintToLEBytes(uval, SW_BYTES[sw]);
        int[] out = new int[1 + buf.length];
        out[0] = sw;
        System.arraycopy(buf, 0, out, 1, buf.length);
        return out;
    }

    private static int[] uintToLEBytes(long v, int size) {
        int[] buf = new int[size];
        for (int i = 0; i < size; i++) buf[i] = (int) ((v >> (8 * i)) & 0xFF);
        return buf;
    }

    private static long reconstructInt(int otype, int sw, long raw) {
        int tbits = targetBits(otype);
        int storedBits = SW_BYTES[sw] * 8;
        if (isUnsigned(otype)) {
            long mask = tbits == 64 ? -1L : (1L << tbits) - 1;
            return raw & mask;
        }
        int signBit = (int) ((raw >> (storedBits - 1)) & 1);
        if (tbits <= storedBits) {
            long mask = (1L << tbits) - 1;
            long val = raw & mask;
            if (signBit == 1 && (val & (1L << (tbits - 1))) != 0) val -= 1L << tbits;
            return val;
        }
        long extended;
        if (signBit == 1) {
            long extendMask = (~((1L << storedBits) - 1)) & ((1L << tbits) - 1);
            extended = raw | extendMask;
        } else extended = raw;
        if (extended >= (1L << (tbits - 1))) extended -= 1L << tbits;
        return extended;
    }

    private static void validateRange(int otype, long val) {
        boolean ok = switch (otype) {
            case TypedValue.OTYPE_UINT8 -> val >= 0 && val <= 0xFF;
            case TypedValue.OTYPE_UINT16 -> val >= 0 && val <= 0xFFFF;
            case TypedValue.OTYPE_UINT32 -> val >= 0 && val <= 0xFFFFFFFFL;
            case TypedValue.OTYPE_UINT64 -> val >= 0;
            case TypedValue.OTYPE_INT8 -> val >= -128 && val <= 127;
            case TypedValue.OTYPE_INT16 -> val >= -32768 && val <= 32767;
            case TypedValue.OTYPE_INT32 -> val >= Integer.MIN_VALUE && val <= Integer.MAX_VALUE;
            case TypedValue.OTYPE_INT64 -> true;
            default -> false;
        };
        if (!ok) throw new IllegalArgumentException("value " + val + " out of range for otype " + otype);
    }
}
