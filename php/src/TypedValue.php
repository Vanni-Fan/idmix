<?php
namespace Vanni\Idmix;

/**
 * 带类型整数的包装，用于保留编码时的原始类型信息。
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
        public readonly int $val,
    ) {}

    public static function u8(int $v): self { return new self(self::OTYPE_UINT8, $v); }
    public static function u16(int $v): self { return new self(self::OTYPE_UINT16, $v); }
    public static function u32(int $v): self { return new self(self::OTYPE_UINT32, $v); }
    public static function u64(int $v): self { return new self(self::OTYPE_UINT64, $v); }
    public static function i8(int $v): self { return new self(self::OTYPE_INT8, $v); }
    public static function i16(int $v): self { return new self(self::OTYPE_INT16, $v); }
    public static function i32(int $v): self { return new self(self::OTYPE_INT32, $v); }
    public static function i64(int $v): self { return new self(self::OTYPE_INT64, $v); }
}
