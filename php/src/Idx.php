<?php
namespace Vanni\Idmix;

/**
 * IDX v1.2 二进制编解码器，可独立于 idmix 文本层使用。
 */
final class Idx
{
    private const SW_BYTES = [1, 2, 4, 8];

    private const EMBEDDED_OTYPE = [
        [TypedValue::OTYPE_UINT8, TypedValue::OTYPE_UINT16, TypedValue::OTYPE_UINT32, TypedValue::OTYPE_UINT64],
        [TypedValue::OTYPE_INT8, TypedValue::OTYPE_INT16, TypedValue::OTYPE_INT32, TypedValue::OTYPE_INT64],
    ];

    public readonly int $maxObjects;
    public readonly int $maxVariants;
    public readonly int $checkBits;
    public readonly int $checkMask;

    private function __construct(int $maxObjects, int $maxVariants, int $checkBits)
    {
        $this->maxObjects = $maxObjects;
        $this->maxVariants = $maxVariants;
        $this->checkBits = $checkBits;
        $this->checkMask = (1 << $checkBits) - 1;
    }

    public static function new(
        ?int $maxObjects = null,
        ?int $maxVariants = null,
        ?int $checkBits = null,
    ): self {
        $mo = $maxObjects ?? 255;
        $mv = $maxVariants ?? 32;
        $cb = $checkBits ?? 2;
        if ($mo < 1 || $mo > 255) {
            throw new \InvalidArgumentException('maxObjects must be between 1 and 255');
        }
        if ($mv < 1 || $mv > 32) {
            throw new \InvalidArgumentException('maxVariants must be between 1 and 32');
        }
        if ($cb < 1 || $cb > 2) {
            throw new \InvalidArgumentException('checkBits must be 1 or 2');
        }
        return new self($mo, $mv, $cb);
    }

    /** @param mixed ...$values */
    public function encode(mixed ...$values): string
    {
        if (count($values) < 1) {
            throw new \InvalidArgumentException('at least one value is required');
        }
        if (count($values) > $this->maxObjects) {
            throw new \InvalidArgumentException(
                'too many objects: ' . count($values) . " (max {$this->maxObjects})"
            );
        }
        return $this->encodeBinary(Number::normalizeObjects($values), 0);
    }

    /** @param mixed ...$values */
    public function encodeWithVariant(int $variantId, mixed ...$values): string
    {
        if (count($values) < 1) {
            throw new \InvalidArgumentException('at least one value is required');
        }
        if (count($values) > $this->maxObjects) {
            throw new \InvalidArgumentException(
                'too many objects: ' . count($values) . " (max {$this->maxObjects})"
            );
        }
        return $this->encodeBinary(Number::normalizeObjects($values), $variantId);
    }

    /** @return array<int, TypedValue|string> */
    public function decode(string $data): array
    {
        return Number::materializeObjects($this->decodeBinary($data));
    }

    /** @param DataObject[] $objects */
    public function encodeBinary(array $objects, int $variantId): string
    {
        IntMath::ensureAvailable();
        if ($variantId < 0 || $variantId >= $this->maxVariants) {
            throw new \InvalidArgumentException(
                "invalid variant_id $variantId (max " . ($this->maxVariants - 1) . ')'
            );
        }

        $objBytes = '';
        foreach ($objects as $obj) {
            $objBytes .= self::encodeObject($obj);
        }

        $mask = ($variantId * 0x9D + 0x37) & 0xFF;
        $objBytes = self::xorBytes($objBytes, $mask);

        $count = count($objects);
        $headerLen = $count === 1 ? 1 : 2;
        if ($count === 1) {
            $data = chr($variantId << $this->checkBits) . $objBytes;
        } else {
            $data = chr(0x80 | ($variantId << $this->checkBits)) . chr($count) . $objBytes;
        }

        $xorSum = 0;
        foreach (str_split($data) as $b) {
            $xorSum ^= ord($b);
        }
        $check = $xorSum & $this->checkMask;
        $data[0] = chr(ord($data[0]) | $check);
        return $data;
    }

    /** @return DataObject[] */
    public function decodeBinary(string $data): array
    {
        IntMath::ensureAvailable();
        if ($data === '') {
            throw new \InvalidArgumentException('invalid data: too short');
        }

        $byte0 = ord($data[0]);
        $check = $byte0 & $this->checkMask;
        $multi = ($byte0 & 0x80) !== 0;
        $variantId = ($byte0 & 0x7F) >> $this->checkBits;

        if ($variantId >= $this->maxVariants) {
            throw new \InvalidArgumentException(
                "invalid variant_id $variantId (max " . ($this->maxVariants - 1) . ')'
            );
        }

        $headerLen = 1;
        $count = 1;
        if ($multi) {
            if (strlen($data) < 2) {
                throw new \InvalidArgumentException('invalid data: missing count byte');
            }
            $headerLen = 2;
            $count = ord($data[1]);
            if ($count < 2 || $count > $this->maxObjects) {
                throw new \InvalidArgumentException("invalid count $count");
            }
        }

        $verify = $data;
        $verify[0] = chr(ord($verify[0]) & ~$this->checkMask);
        $xorSum = 0;
        foreach (str_split($verify) as $b) {
            $xorSum ^= ord($b);
        }
        if (($xorSum & $this->checkMask) !== $check) {
            throw new \InvalidArgumentException('checksum mismatch');
        }

        $objData = substr($data, $headerLen);
        $mask = ($variantId * 0x9D + 0x37) & 0xFF;
        $objData = self::xorBytes($objData, $mask);

        $result = [];
        $pos = 0;
        $len = strlen($objData);
        for ($i = 0; $i < $count; $i++) {
            if ($pos >= $len) {
                throw new \InvalidArgumentException('premature end of data');
            }
            try {
                [$obj, $n] = self::decodeObject(substr($objData, $pos));
            } catch (\InvalidArgumentException $e) {
                throw new \InvalidArgumentException("object[$i]: {$e->getMessage()}", 0, $e);
            }
            $result[] = $obj;
            $pos += $n;
        }
        if ($pos !== $len) {
            throw new \InvalidArgumentException('extra bytes after data objects');
        }
        return $result;
    }

    private static function encodeObject(DataObject $obj): string
    {
        if ($obj->isString) {
            $n = strlen($obj->str);
            if ($n < 1 || $n > Number::MAX_STRING_LEN) {
                throw new \InvalidArgumentException(
                    "string length $n out of range [1, " . Number::MAX_STRING_LEN . ']'
                );
            }
            return chr(0xC0 | $n) . $obj->str;
        }

        IntMath::validateRange($obj->otype, $obj->val);
        $embedded = self::tryEmbeddedHead($obj->otype, $obj->val);
        if ($embedded !== null) {
            return chr($embedded);
        }

        [$sw, $payload] = self::payloadForNumber($obj->otype, $obj->val);
        $head = 0x80 | ($sw << 4) | $obj->otype;
        return chr($head) . $payload;
    }

    /** @return array{0: DataObject, 1: int} */
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

        if (($head & 0x40) !== 0) {
            $n = $head & 0x3F;
            if ($n < 1 || $n > Number::MAX_STRING_LEN) {
                throw new \InvalidArgumentException("invalid string length $n");
            }
            if (strlen($data) < 1 + $n) {
                throw new \InvalidArgumentException('truncated string payload');
            }
            return [DataObject::string(substr($data, 1, $n)), 1 + $n];
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
        $val = self::valueFromPayload($otype, substr($data, 1, $numBytes));
        IntMath::validateRange($otype, $val);
        return [self::fromOtypeVal($otype, $val), 1 + $numBytes];
    }

    /** @return array{0: int, 1: string} */
    private static function payloadForNumber(int $otype, string $val): array
    {
        if ($otype === TypedValue::OTYPE_UINT64) {
            $mag = IntMath::modPositive($val);
            $sw = self::swFromMagnitude($mag);
            return [$sw, IntMath::uintToLEBytes($mag, self::SW_BYTES[$sw])];
        }
        if (self::isUnsigned($otype)) {
            if (IntMath::isNegative($val)) {
                throw new \InvalidArgumentException("negative value $val for unsigned otype $otype");
            }
            $sw = self::swFromMagnitude($val);
            return [$sw, IntMath::uintToLEBytes($val, self::SW_BYTES[$sw])];
        }
        $sw = IntMath::swFromSignedValue($val);
        return [$sw, IntMath::signedToLEBytes($val, self::SW_BYTES[$sw])];
    }

    private static function valueFromPayload(int $otype, string $payload): string
    {
        if (self::isUnsigned($otype)) {
            $mag = IntMath::bytesToUint($payload);
            if ($otype !== TypedValue::OTYPE_UINT64 && IntMath::compare($mag, IntMath::pow2(63)) > 0) {
                throw new \InvalidArgumentException("value out of range for otype $otype");
            }
            return $mag;
        }
        return IntMath::leBytesToSigned($payload);
    }

    private static function isUnsigned(int $otype): bool
    {
        return $otype <= TypedValue::OTYPE_UINT64;
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

    private static function fromOtypeVal(int $otype, string $val): DataObject
    {
        return match ($otype) {
            TypedValue::OTYPE_UINT8 => DataObject::integer($otype, TypedValue::u8($val)->val),
            TypedValue::OTYPE_UINT16 => DataObject::integer($otype, TypedValue::u16($val)->val),
            TypedValue::OTYPE_UINT32 => DataObject::integer($otype, TypedValue::u32($val)->val),
            TypedValue::OTYPE_UINT64 => DataObject::integer($otype, TypedValue::u64($val)->val),
            TypedValue::OTYPE_INT8 => DataObject::integer($otype, TypedValue::i8($val)->val),
            TypedValue::OTYPE_INT16 => DataObject::integer($otype, TypedValue::i16($val)->val),
            TypedValue::OTYPE_INT32 => DataObject::integer($otype, TypedValue::i32($val)->val),
            TypedValue::OTYPE_INT64 => DataObject::integer($otype, TypedValue::i64($val)->val),
            default => throw new \InvalidArgumentException("invalid otype $otype"),
        };
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
