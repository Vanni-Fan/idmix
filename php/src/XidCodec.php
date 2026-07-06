<?php
namespace Vanni\Idmix;

/**
 * XID v1.1 二进制层编解码（整数运算经 IntMath / bcmath）。
 */
class XidCodec
{
    private const SW_BYTES = [1, 2, 4, 8];

    private const EMBEDDED_OTYPE = [
        [TypedValue::OTYPE_UINT8, TypedValue::OTYPE_UINT16, TypedValue::OTYPE_UINT32, TypedValue::OTYPE_UINT64],
        [TypedValue::OTYPE_INT8, TypedValue::OTYPE_INT16, TypedValue::OTYPE_INT32, TypedValue::OTYPE_INT64],
    ];

    /** @param TypedValue[] $typed */
    public static function encodeBinary(IdMix $m, array $typed, int $variantId): string
    {
        IntMath::ensureAvailable();
        $objects = '';
        foreach ($typed as $tv) {
            $objects .= self::encodeObject($tv);
        }
        $mask = ($variantId * 0x9D + 0x37) & 0xFF;
        $objects = self::xorBytes($objects, $mask);

        $count = count($typed);
        $header = ($variantId << $m->variantShift) | ($count << $m->countShift);
        $data = pack('v', $header) . $objects;

        $xorSum = 0;
        foreach (str_split($data) as $b) {
            $xorSum ^= ord($b);
        }
        $check = $xorSum & $m->checkMask;
        $header |= $check;
        return pack('v', $header) . $objects;
    }

    /** @return TypedValue[] */
    public static function decodeBinary(IdMix $m, string $data): array
    {
        IntMath::ensureAvailable();
        if (strlen($data) < 2) {
            throw new \InvalidArgumentException('invalid data: too short');
        }
        $header = unpack('v', substr($data, 0, 2))[1];
        $check = $header & $m->checkMask;
        $count = ($header & $m->countMask) >> $m->countShift;
        $variantId = ($header & $m->variantMask) >> $m->variantShift;

        if ($variantId >= $m->maxVariants) {
            throw new \InvalidArgumentException("invalid variant_id $variantId");
        }
        if ($count > $m->maxObjects) {
            throw new \InvalidArgumentException("invalid count $count");
        }

        $verify = $data;
        $verify[0] = chr(ord($verify[0]) & ~$m->checkMask);
        $xorSum = 0;
        foreach (str_split($verify) as $b) {
            $xorSum ^= ord($b);
        }
        if (($xorSum & $m->checkMask) !== $check) {
            throw new \InvalidArgumentException('checksum mismatch');
        }

        $objects = substr($data, 2);
        $mask = ($variantId * 0x9D + 0x37) & 0xFF;
        $objects = self::xorBytes($objects, $mask);

        $result = [];
        $pos = 0;
        $len = strlen($objects);
        for ($i = 0; $i < $count; $i++) {
            if ($pos >= $len) {
                throw new \InvalidArgumentException('premature end of data');
            }
            [$tv, $n] = self::decodeObject(substr($objects, $pos));
            $result[] = $tv;
            $pos += $n;
        }
        if ($pos !== $len) {
            throw new \InvalidArgumentException('extra bytes after data objects');
        }
        return $result;
    }

    private static function encodeObject(TypedValue $tv): string
    {
        IntMath::validateRange($tv->otype, $tv->val);
        $embedded = self::tryEmbeddedHead($tv->otype, $tv->val);
        if ($embedded !== null) {
            return chr($embedded);
        }
        [$mag, $neg] = self::magnitudeFromTyped($tv->otype, $tv->val);
        $sw = self::swFromMagnitude($mag);
        $payload = IntMath::uintToLEBytes($mag, self::SW_BYTES[$sw]);
        $head = 0x80 | ($sw << 4) | $tv->otype;
        if ($neg) {
            $head |= 1 << 6;
        }
        return chr($head) . $payload;
    }

    /** @return array{0: TypedValue, 1: int} */
    private static function decodeObject(string $data): array
    {
        if ($data === '') {
            throw new \InvalidArgumentException('truncated object header');
        }
        $head = ord($data[0]);
        if (($head & 0x80) === 0) {
            $sign = ($head >> 6) & 1;
            $wb = ($head >> 4) & 0x03;
            $v = $head & 0x0F;
            $otype = self::EMBEDDED_OTYPE[$sign][$wb];
            $val = $sign === 0 ? (string) $v : (string) (-$v - 1);
            return [self::fromOtypeVal($otype, $val), 1];
        }
        $sw = ($head >> 4) & 0x03;
        $otype = $head & 0x0F;
        if ($otype > TypedValue::OTYPE_INT64) {
            throw new \InvalidArgumentException("invalid otype $otype");
        }
        $numBytes = self::SW_BYTES[$sw];
        if (strlen($data) < 1 + $numBytes) {
            throw new \InvalidArgumentException('truncated object payload');
        }
        $rawBytes = substr($data, 1, $numBytes);
        $mag = IntMath::bytesToUint($rawBytes);
        $neg = (($head >> 6) & 1) !== 0;
        $val = self::valueFromMagnitude($mag, $neg);
        IntMath::validateRange($otype, $val);
        return [self::fromOtypeVal($otype, $val), 1 + $numBytes];
    }

    private static function fromOtypeVal(int $otype, string $val): TypedValue
    {
        return match ($otype) {
            TypedValue::OTYPE_UINT8 => TypedValue::u8($val),
            TypedValue::OTYPE_UINT16 => TypedValue::u16($val),
            TypedValue::OTYPE_UINT32 => TypedValue::u32($val),
            TypedValue::OTYPE_UINT64 => TypedValue::u64($val),
            TypedValue::OTYPE_INT8 => TypedValue::i8($val),
            TypedValue::OTYPE_INT16 => TypedValue::i16($val),
            TypedValue::OTYPE_INT32 => TypedValue::i32($val),
            TypedValue::OTYPE_INT64 => TypedValue::i64($val),
            default => throw new \InvalidArgumentException("invalid otype $otype"),
        };
    }

    private static function isUnsigned(int $otype): bool
    {
        return $otype <= TypedValue::OTYPE_UINT64;
    }

    private static function isSigned(int $otype): bool
    {
        return $otype >= TypedValue::OTYPE_INT8;
    }

    private static function widthBits(int $otype): int
    {
        return match ($otype) {
            TypedValue::OTYPE_UINT8, TypedValue::OTYPE_INT8 => 0,
            TypedValue::OTYPE_UINT16, TypedValue::OTYPE_INT16 => 1,
            TypedValue::OTYPE_UINT32, TypedValue::OTYPE_INT32 => 2,
            default => 3,
        };
    }

    /** @return array{0: string, 1: bool} */
    private static function magnitudeFromTyped(int $otype, string $val): array
    {
        if (self::isUnsigned($otype)) {
            return [IntMath::modPositive($val), false];
        }
        if (IntMath::isNegative($val)) {
            return [IntMath::sub('0', $val), true];
        }
        return [$val, false];
    }

    private static function swFromMagnitude(string $mag): int
    {
        if (IntMath::compare($mag, '256') < 0) {
            return 0;
        }
        if (IntMath::compare($mag, '65536') < 0) {
            return 1;
        }
        if (IntMath::compare($mag, '4294967296') < 0) {
            return 2;
        }
        return 3;
    }

    private static function tryEmbeddedHead(int $otype, string $val): ?int
    {
        [$mag, $neg] = self::magnitudeFromTyped($otype, $val);
        if (IntMath::compare($mag, '17') >= 0) {
            return null;
        }
        $wb = self::widthBits($otype);
        if (IntMath::compare($mag, '16') === 0) {
            if ($neg) {
                return (1 << 6) | ($wb << 4) | 15;
            }
            return null;
        }
        if ($neg) {
            return (1 << 6) | ($wb << 4) | ((int) IntMath::sub($mag, '1'));
        }
        return ($wb << 4) | (int) $mag;
    }

    private static function valueFromMagnitude(string $mag, bool $neg): string
    {
        if (!$neg) {
            return $mag;
        }
        if (IntMath::compare($mag, IntMath::pow2(63)) === 0) {
            return IntMath::sub('0', IntMath::pow2(63));
        }
        return IntMath::sub('0', $mag);
    }

    private static function xorBytes(string $data, int $mask): string
    {
        $out = '';
        foreach (str_split($data) as $b) {
            $out .= chr(ord($b) ^ $mask);
        }
        return $out;
    }
}
