<?php
namespace Vanni\Idmix;

/** 内部统一的数据对象表示：整数或短字符串。 */
final class DataObject
{
    public function __construct(
        public readonly bool $isString = false,
        public readonly int $otype = 0,
        public readonly string $val = '0',
        public readonly string $str = '',
    ) {
    }

    public static function integer(int $otype, string $val): self
    {
        return new self(false, $otype, $val);
    }

    public static function string(string $str): self
    {
        return new self(true, 0, '0', $str);
    }
}
