<?php
namespace Vanni\Idmix;

/**
 * 带类型整数的包装；数值统一为十进制字符串（经 bcmath 运算，支持完整 uint64）。
 */
class TypedValue
{
    public const OTYPE_UINT8 = 0;
    public const OTYPE_UINT16 = 1;
    public const OTYPE_UINT32 = 2;
    public const OTYPE_UINT64 = 3;
    public const OTYPE_INT8 = 4;
    public const OTYPE_INT16 = 5;
    public const OTYPE_INT32 = 6;
    public const OTYPE_INT64 = 7;

    public function __construct(
        public readonly int $otype,
        public readonly string $val,
    ) {}

    public static function u8(int|string $v): self
    {
        $s = IntMath::normalize($v);
        IntMath::validateRange(self::OTYPE_UINT8, $s);
        return new self(self::OTYPE_UINT8, $s);
    }

    public static function u16(int|string $v): self
    {
        $s = IntMath::normalize($v);
        IntMath::validateRange(self::OTYPE_UINT16, $s);
        return new self(self::OTYPE_UINT16, $s);
    }

    public static function u32(int|string $v): self
    {
        $s = IntMath::normalize($v);
        IntMath::validateRange(self::OTYPE_UINT32, $s);
        return new self(self::OTYPE_UINT32, $s);
    }

    public static function u64(int|string $v): self
    {
        $s = IntMath::normalize($v);
        IntMath::validateRange(self::OTYPE_UINT64, $s);
        return new self(self::OTYPE_UINT64, $s);
    }

    public static function i8(int|string $v): self
    {
        $s = IntMath::normalize($v);
        IntMath::validateRange(self::OTYPE_INT8, $s);
        return new self(self::OTYPE_INT8, $s);
    }

    public static function i16(int|string $v): self
    {
        $s = IntMath::normalize($v);
        IntMath::validateRange(self::OTYPE_INT16, $s);
        return new self(self::OTYPE_INT16, $s);
    }

    public static function i32(int|string $v): self
    {
        $s = IntMath::normalize($v);
        IntMath::validateRange(self::OTYPE_INT32, $s);
        return new self(self::OTYPE_INT32, $s);
    }

    public static function i64(int|string $v): self
    {
        $s = IntMath::normalize($v);
        IntMath::validateRange(self::OTYPE_INT64, $s);
        return new self(self::OTYPE_INT64, $s);
    }
}
