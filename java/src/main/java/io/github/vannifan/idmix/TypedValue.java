package io.github.vannifan.idmix;

/** Typed integer value with original type index. */
public final class TypedValue {
    public static final int OTYPE_UINT8 = 0;
    public static final int OTYPE_UINT16 = 1;
    public static final int OTYPE_UINT32 = 2;
    public static final int OTYPE_UINT64 = 3;
    public static final int OTYPE_INT8 = 4;
    public static final int OTYPE_INT16 = 5;
    public static final int OTYPE_INT32 = 6;
    public static final int OTYPE_INT64 = 7;

    public final int otype;
    public final long val;

    public TypedValue(int otype, long val) {
        this.otype = otype;
        this.val = val;
    }

    public static TypedValue u8(int v) { return new TypedValue(OTYPE_UINT8, v); }
    public static TypedValue u16(int v) { return new TypedValue(OTYPE_UINT16, v); }
    public static TypedValue u32(long v) { return new TypedValue(OTYPE_UINT32, v); }
    public static TypedValue u64(long v) {
        if (v > Long.MAX_VALUE) throw new IllegalArgumentException("uint64 overflows");
        return new TypedValue(OTYPE_UINT64, v);
    }
    public static TypedValue i8(int v) { return new TypedValue(OTYPE_INT8, v); }
    public static TypedValue i16(int v) { return new TypedValue(OTYPE_INT16, v); }
    public static TypedValue i32(int v) { return new TypedValue(OTYPE_INT32, v); }
    public static TypedValue i64(long v) { return new TypedValue(OTYPE_INT64, v); }

    @Override
    public boolean equals(Object o) {
        if (!(o instanceof TypedValue tv)) return false;
        return otype == tv.otype && val == tv.val;
    }

    @Override
    public int hashCode() {
        return otype * 31 + Long.hashCode(val);
    }

    @Override
    public String toString() {
        return "TypedValue{otype=" + otype + ", val=" + val + "}";
    }
}
