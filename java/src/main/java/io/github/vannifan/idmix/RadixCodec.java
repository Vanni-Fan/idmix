package io.github.vannifan.idmix;

import java.math.BigInteger;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.util.HashMap;
import java.util.Map;

/** Custom radix text layer codec (default idmix text layer). */
public final class RadixCodec implements ICodec {
    private final int base;
    private final String chars;
    private final Map<Character, Integer> fromCustom;

    public RadixCodec(String alphabet) {
        if (alphabet.length() < 2) {
            throw new IllegalArgumentException("alphabet must have at least 2 unique characters");
        }
        this.base = alphabet.length();
        this.chars = alphabet;
        this.fromCustom = new HashMap<>();
        for (int i = 0; i < alphabet.length(); i++) {
            char c = alphabet.charAt(i);
            if (fromCustom.containsKey(c)) {
                throw new IllegalArgumentException("alphabet contains duplicate character " + c);
            }
            fromCustom.put(c, i);
        }
    }

    public String getAlphabet() { return chars; }
    public int getBase() { return base; }

    @Override
    public String encode(byte[] data) {
        if (data.length == 0) return String.valueOf(chars.charAt(0));
        ByteBuffer buf = ByteBuffer.allocate(2 + data.length);
        buf.order(ByteOrder.BIG_ENDIAN);
        buf.putShort((short) data.length);
        buf.put(data);
        BigInteger n = new BigInteger(1, buf.array());
        return intToString(n);
    }

    @Override
    public byte[] decode(String s) {
        if (s == null || s.isEmpty()) throw new IllegalArgumentException("empty string");
        BigInteger n = stringToInt(s);
        byte[] raw = n.toByteArray();
        if (raw.length > 1 && raw[0] == 0) {
            byte[] trimmed = new byte[raw.length - 1];
            System.arraycopy(raw, 1, trimmed, 0, trimmed.length);
            raw = trimmed;
        }
        for (int pad = 0; pad <= 1; pad++) {
            byte[] buf = new byte[pad + raw.length];
            System.arraycopy(raw, 0, buf, pad, raw.length);
            if (buf.length < 2) continue;
            int dataLen = ((buf[0] & 0xff) << 8) | (buf[1] & 0xff);
            if (buf.length != 2 + dataLen) continue;
            byte[] out = new byte[dataLen];
            System.arraycopy(buf, 2, out, 0, dataLen);
            return out;
        }
        throw new IllegalArgumentException("invalid encoded data length");
    }

    private String intToString(BigInteger n) {
        if (n.signum() == 0) return String.valueOf(chars.charAt(0));
        BigInteger baseBI = BigInteger.valueOf(base);
        StringBuilder sb = new StringBuilder();
        while (n.signum() > 0) {
            BigInteger[] div = n.divideAndRemainder(baseBI);
            sb.append(chars.charAt(div[1].intValue()));
            n = div[0];
        }
        return sb.reverse().toString();
    }

    private BigInteger stringToInt(String s) {
        BigInteger n = BigInteger.ZERO;
        BigInteger baseBI = BigInteger.valueOf(base);
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            Integer idx = fromCustom.get(c);
            if (idx == null) throw new IllegalArgumentException("invalid character " + c);
            n = n.multiply(baseBI).add(BigInteger.valueOf(idx));
        }
        return n;
    }
}
