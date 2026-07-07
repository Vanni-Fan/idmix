<?php
namespace Vanni\Idmix;

/**
 * 任意精度整数运算（依赖 ext-bcmath）。
 * 所有值以十进制字符串表示，32/64 位 PHP 均可处理完整 uint64 范围。
 */
final class IntMath
{
    private const UINT64_MAX = '18446744073709551615';

    public static function ensureAvailable(): void
    {
        if (!extension_loaded('bcmath')) {
            throw new \RuntimeException('ext-bcmath is required for idmix integer operations');
        }
    }

    public static function normalize(int|string $v): string
    {
        self::ensureAvailable();
        if (is_int($v)) {
            return (string) $v;
        }
        if (!is_string($v) || !preg_match('/^-?\d+$/', $v)) {
            throw new \InvalidArgumentException("invalid integer: $v");
        }
        return $v;
    }

    public static function isZero(string $v): bool
    {
        return bccomp($v, '0') === 0;
    }

    public static function isNegative(string $v): bool
    {
        return bccomp($v, '0') < 0;
    }

    public static function compare(string $a, string $b): int
    {
        return bccomp($a, $b);
    }

    public static function add(string $a, string $b): string
    {
        return bcadd($a, $b);
    }

    public static function sub(string $a, string $b): string
    {
        return bcsub($a, $b);
    }

    public static function andMask(string $v, string $mask): string
    {
        $bits = self::toBitString(self::modPositive($v));
        $mbits = self::toBitString($mask);
        $len = max(strlen($bits), strlen($mbits));
        $bits = str_pad($bits, $len, '0', STR_PAD_LEFT);
        $mbits = str_pad($mbits, $len, '0', STR_PAD_LEFT);
        $out = '';
        for ($i = 0; $i < $len; $i++) {
            $out .= ((int) $bits[$i] & (int) $mbits[$i]) ? '1' : '0';
        }
        return self::fromBitString($out);
    }

    public static function orMask(string $v, string $mask): string
    {
        $bits = self::toBitString(self::modPositive($v));
        $mbits = self::toBitString($mask);
        $len = max(strlen($bits), strlen($mbits));
        $bits = str_pad($bits, $len, '0', STR_PAD_LEFT);
        $mbits = str_pad($mbits, $len, '0', STR_PAD_LEFT);
        $out = '';
        for ($i = 0; $i < $len; $i++) {
            $out .= ((int) $bits[$i] | (int) $mbits[$i]) ? '1' : '0';
        }
        return self::fromBitString($out);
    }

    public static function modPositive(string $v): string
    {
        if (!self::isNegative($v)) {
            return $v;
        }
        $width = self::bitWidthForValue($v);
        $mod = bcpow('2', (string) $width);
        return bcadd($mod, $v);
    }

    public static function maskWidth(string $v, int $bits): string
    {
        if ($bits === 64) {
            return self::andMask($v, self::UINT64_MAX);
        }
        $mask = bcsub(bcpow('2', (string) $bits), '1');
        return self::andMask($v, $mask);
    }

    public static function pow2(int $exp): string
    {
        return bcpow('2', (string) $exp);
    }

    public static function uintToLEBytes(string $v, int $size): string
    {
        $n = self::modPositive($v);
        $buf = '';
        for ($i = 0; $i < $size; $i++) {
            $byte = bcmod($n, '256');
            $buf .= chr((int) $byte);
            $n = bcdiv($n, '256', 0);
        }
        return $buf;
    }

    public static function signedToLEBytes(string $v, int $size): string
    {
        $bits = $size * 8;
        $mod = bcpow('2', (string) $bits);
        $u = self::isNegative($v) ? bcadd($mod, $v) : $v;
        return self::uintToLEBytes($u, $size);
    }

    public static function leBytesToSigned(string $payload): string
    {
        $u = self::bytesToUint($payload);
        $size = strlen($payload);
        $bitWidth = $size * 8;
        $signBit = self::pow2($bitWidth - 1);
        if (self::compare($u, $signBit) >= 0) {
            $u = bcsub($u, bcpow('2', (string) $bitWidth));
        }
        return $u;
    }

    public static function swFromSignedValue(string $val): int
    {
        if (self::compare($val, '-128') >= 0 && self::compare($val, '127') <= 0) {
            return 0;
        }
        if (self::compare($val, '-32768') >= 0 && self::compare($val, '32767') <= 0) {
            return 1;
        }
        if (self::compare($val, '-2147483648') >= 0 && self::compare($val, '2147483647') <= 0) {
            return 2;
        }
        return 3;
    }

    public static function bytesToUint(string $bytes): string
    {
        $n = '0';
        for ($i = strlen($bytes) - 1; $i >= 0; $i--) {
            $n = bcadd(bcmul($n, '256'), (string) ord($bytes[$i]));
        }
        return $n;
    }

    public static function validateRange(int $otype, string $val): void
    {
        $ok = match ($otype) {
            TypedValue::OTYPE_UINT8 => self::compare($val, '0') >= 0 && self::compare($val, '255') <= 0,
            TypedValue::OTYPE_UINT16 => self::compare($val, '0') >= 0 && self::compare($val, '65535') <= 0,
            TypedValue::OTYPE_UINT32 => self::compare($val, '0') >= 0 && self::compare($val, '4294967295') <= 0,
            TypedValue::OTYPE_UINT64 => self::compare($val, '0') >= 0 && self::compare($val, self::UINT64_MAX) <= 0,
            TypedValue::OTYPE_INT8 => self::compare($val, '-128') >= 0 && self::compare($val, '127') <= 0,
            TypedValue::OTYPE_INT16 => self::compare($val, '-32768') >= 0 && self::compare($val, '32767') <= 0,
            TypedValue::OTYPE_INT32 => self::compare($val, '-2147483648') >= 0 && self::compare($val, '2147483647') <= 0,
            TypedValue::OTYPE_INT64 => self::compare($val, '-9223372036854775808') >= 0 && self::compare($val, '9223372036854775807') <= 0,
            default => false,
        };
        if (!$ok) {
            throw new \InvalidArgumentException("value $val out of range for otype $otype");
        }
    }

    private static function bitWidthForValue(string $v): int
    {
        if (self::compare($val = $v, '0') >= 0) {
            return self::decimalBitLen($val);
        }
        $abs = self::sub('0', $val);
        return self::decimalBitLen($abs) + 1;
    }

    private static function decimalBitLen(string $v): int
    {
        if (self::isZero($v)) {
            return 1;
        }
        return strlen(self::toBitString($v));
    }

    private static function toBitString(string $v): string
    {
        if (self::isZero($v)) {
            return '0';
        }
        $bits = '';
        $n = self::modPositive($v);
        while (bccomp($n, '0') > 0) {
            $bits = ((int) bcmod($n, '2')) . $bits;
            $n = bcdiv($n, '2', 0);
        }
        return $bits;
    }

    private static function fromBitString(string $bits): string
    {
        $bits = ltrim($bits, '0');
        if ($bits === '') {
            return '0';
        }
        $n = '0';
        foreach (str_split($bits) as $bit) {
            $n = bcadd(bcmul($n, '2'), $bit);
        }
        return $n;
    }
}
