<?php
namespace Vanni\Idmix;

/**
 * XID v1.1 二进制层编解码。
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
        self::validateRange($tv->otype, $tv->val);
        if (self::isUnsigned($tv->otype) && $tv->val >= 0 && $tv->val <= 15) {
            $wb = self::widthBits($tv->otype);
            return chr(($wb << 4) | $tv->val);
        }
        if (self::isSigned($tv->otype) && $tv->val >= -16 && $tv->val <= -1) {
            $wb = self::widthBits($tv->otype);
            $v = -$tv->val - 1;
            return chr((1 << 6) | ($wb << 4) | $v);
        }
        [$sw, $payload] = self::minimalComplementBytes($tv->otype, $tv->val);
        return chr(0x80 | ($sw << 4) | $tv->otype) . $payload;
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
            $val = $sign === 0 ? $v : -$v - 1;
            return [new TypedValue($otype, $val), 1];
        }
        if ((($head >> 6) & 1) !== 0) {
            throw new \InvalidArgumentException('reserved bit set in extended mode');
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
        $raw = 0;
        for ($i = 0; $i < $numBytes; $i++) {
            $raw |= ord($data[1 + $i]) << (8 * $i);
        }
        $val = self::reconstructInt($otype, $sw, $raw);
        return [new TypedValue($otype, $val), 1 + $numBytes];
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

    private static function targetBits(int $otype): int
    {
        return match ($otype) {
            TypedValue::OTYPE_UINT8, TypedValue::OTYPE_INT8 => 8,
            TypedValue::OTYPE_UINT16, TypedValue::OTYPE_INT16 => 16,
            TypedValue::OTYPE_UINT32, TypedValue::OTYPE_INT32 => 32,
            default => 64,
        };
    }

    /** @return array{0: int, 1: string} */
    private static function minimalComplementBytes(int $otype, int $val): array
    {
        if ($val === 0) {
            return [0, "\x00"];
        }
        if (self::isUnsigned($otype)) {
            if ($val < 0) {
                throw new \InvalidArgumentException("negative value $val for unsigned type");
            }
            $uval = $val;
            for ($sw = 0; $sw < 4; $sw++) {
                $size = self::SW_BYTES[$sw];
                if ($size < 8 && $uval >= (1 << ($size * 8))) {
                    continue;
                }
                $buf = self::uintToLEBytes($uval, $size);
                if ((ord($buf[$size - 1]) & 0x80) === 0) {
                    return [$sw, $buf];
                }
            }
            throw new \InvalidArgumentException('value too large for unsigned type');
        }

        $tbits = self::targetBits($otype);
        $mask = $tbits === 64 ? PHP_INT_MAX : ((1 << $tbits) - 1);
        $uval = $val & $mask;

        if ($val < 0) {
            for ($sw = 0; $sw < 4; $sw++) {
                $size = self::SW_BYTES[$sw];
                $shift = $size * 8;
                if ($shift >= $tbits) {
                    return [$sw, self::uintToLEBytes($uval, $size)];
                }
                $lower = $uval & ((1 << $shift) - 1);
                $upper = $uval >> $shift;
                $upperMask = (1 << ($tbits - $shift)) - 1;
                if ($upper !== $upperMask) {
                    continue;
                }
                $highByte = ($lower >> ($shift - 8)) & 0xFF;
                if (($highByte & 0x80) === 0) {
                    continue;
                }
                return [$sw, self::uintToLEBytes($lower, $size)];
            }
        } else {
            for ($sw = 0; $sw < 4; $sw++) {
                $size = self::SW_BYTES[$sw];
                if ($size < 8 && $uval >= (1 << ($size * 8))) {
                    continue;
                }
                $buf = self::uintToLEBytes($uval, $size);
                if ((ord($buf[$size - 1]) & 0x80) === 0) {
                    return [$sw, $buf];
                }
            }
        }

        $sw = match ($tbits) {
            8 => 0, 16 => 1, 32 => 2, default => 3,
        };
        return [$sw, self::uintToLEBytes($uval, self::SW_BYTES[$sw])];
    }

    private static function uintToLEBytes(int $v, int $size): string
    {
        $buf = '';
        for ($i = 0; $i < $size; $i++) {
            $buf .= chr(($v >> (8 * $i)) & 0xFF);
        }
        return $buf;
    }

    private static function reconstructInt(int $otype, int $sw, int $raw): int
    {
        $tbits = self::targetBits($otype);
        $storedBits = self::SW_BYTES[$sw] * 8;
        if (self::isUnsigned($otype)) {
            $mask = $tbits === 64 ? PHP_INT_MAX : ((1 << $tbits) - 1);
            return $raw & $mask;
        }
        $signBit = ($raw >> ($storedBits - 1)) & 1;
        if ($tbits <= $storedBits) {
            $mask = $tbits === 64 ? PHP_INT_MAX : ((1 << $tbits) - 1);
            $val = $raw & $mask;
            if ($signBit === 1 && ($val & (1 << ($tbits - 1))) !== 0) {
                $val -= 1 << $tbits;
            }
            return $val;
        }
        if ($signBit === 1) {
            $extendMask = (~((1 << $storedBits) - 1)) & ((1 << $tbits) - 1);
            $extended = $raw | $extendMask;
        } else {
            $extended = $raw;
        }
        if ($extended >= (1 << ($tbits - 1))) {
            $extended -= 1 << $tbits;
        }
        return $extended;
    }

    private static function validateRange(int $otype, int $val): void
    {
        $ok = match ($otype) {
            TypedValue::OTYPE_UINT8 => $val >= 0 && $val <= 0xFF,
            TypedValue::OTYPE_UINT16 => $val >= 0 && $val <= 0xFFFF,
            TypedValue::OTYPE_UINT32 => $val >= 0 && $val <= 0xFFFFFFFF,
            TypedValue::OTYPE_UINT64 => $val >= 0,
            TypedValue::OTYPE_INT8 => $val >= -128 && $val <= 127,
            TypedValue::OTYPE_INT16 => $val >= -32768 && $val <= 32767,
            TypedValue::OTYPE_INT32 => $val >= -2147483648 && $val <= 2147483647,
            TypedValue::OTYPE_INT64 => true,
            default => false,
        };
        if (!$ok) {
            throw new \InvalidArgumentException("value $val out of range for otype $otype");
        }
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
