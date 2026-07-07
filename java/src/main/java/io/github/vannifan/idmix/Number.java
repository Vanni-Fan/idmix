package io.github.vannifan.idmix;

import java.nio.charset.StandardCharsets;

/** Type normalization for Idx encode/decode. */
final class Number {
    static final int MAX_STRING_LEN = 63;

    private Number() {}

    static final class DataObject {
        final boolean isString;
        final int otype;
        final long val;
        final byte[] str;

        private DataObject(boolean isString, int otype, long val, byte[] str) {
            this.isString = isString;
            this.otype = otype;
            this.val = val;
            this.str = str;
        }

        static DataObject integer(int otype, long val) {
            return new DataObject(false, otype, val, null);
        }

        static DataObject string(byte[] str) {
            return new DataObject(true, 0, 0, str);
        }
    }

    static DataObject[] normalizeObjects(Object[] values) {
        DataObject[] out = new DataObject[values.length];
        for (int i = 0; i < values.length; i++) {
            try {
                out[i] = objectFromAny(values[i]);
            } catch (IllegalArgumentException ex) {
                throw new IllegalArgumentException("value[" + i + "]: " + ex.getMessage(), ex);
            }
        }
        return out;
    }

    static Object[] materializeObjects(DataObject[] objects) {
        Object[] out = new Object[objects.length];
        for (int i = 0; i < objects.length; i++) {
            if (objects[i].isString) {
                out[i] = new String(objects[i].str, StandardCharsets.UTF_8);
            } else {
                out[i] = TypedValue.fromOtype(objects[i].otype, objects[i].val);
            }
        }
        return out;
    }

    static DataObject objectFromAny(Object v) {
        if (v instanceof String s) {
            if (s.isEmpty()) {
                throw new IllegalArgumentException("empty string is not allowed (max " + MAX_STRING_LEN + " bytes)");
            }
            byte[] bytes = s.getBytes(StandardCharsets.UTF_8);
            if (bytes.length > MAX_STRING_LEN) {
                throw new IllegalArgumentException("string length " + bytes.length + " exceeds max " + MAX_STRING_LEN);
            }
            return DataObject.string(bytes);
        }
        if (v instanceof byte[] bs) {
            if (bs.length == 0) {
                throw new IllegalArgumentException("empty byte slice is not allowed (max " + MAX_STRING_LEN + " bytes)");
            }
            if (bs.length > MAX_STRING_LEN) {
                throw new IllegalArgumentException("byte slice length " + bs.length + " exceeds max " + MAX_STRING_LEN);
            }
            byte[] copy = new byte[bs.length];
            System.arraycopy(bs, 0, copy, 0, bs.length);
            return DataObject.string(copy);
        }
        if (v instanceof TypedValue tv) {
            return DataObject.integer(tv.otype, tv.val);
        }
        if (v instanceof Byte x) return DataObject.integer(TypedValue.OTYPE_INT8, x);
        if (v instanceof Short x) return DataObject.integer(TypedValue.OTYPE_INT16, x);
        if (v instanceof Integer x) return DataObject.integer(TypedValue.OTYPE_INT32, x);
        if (v instanceof Long x) return DataObject.integer(TypedValue.OTYPE_INT64, x);
        throw new IllegalArgumentException(
                "unsupported type " + v.getClass().getSimpleName() + " (integer or string up to " + MAX_STRING_LEN + " bytes)");
    }

    private static Object materializeValue(DataObject obj) {
        return TypedValue.fromOtype(obj.otype, obj.val);
    }
}
