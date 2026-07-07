<?php
namespace Vanni\Idmix;

/**
 * 类型转换层：将调用方值规范化为内部 DataObject 表示。
 */
final class Number
{
    public const MAX_STRING_LEN = 63;

    private function __construct()
    {
    }

    /** @param mixed[] $values @return DataObject[] */
    public static function normalizeObjects(array $values): array
    {
        $out = [];
        foreach ($values as $i => $v) {
            try {
                $out[] = self::objectFromAny($v);
            } catch (\InvalidArgumentException $e) {
                throw new \InvalidArgumentException("value[$i]: {$e->getMessage()}", 0, $e);
            }
        }
        return $out;
    }

    /** @param DataObject[] $objects @return array<int, TypedValue|string> */
    public static function materializeObjects(array $objects): array
    {
        $out = [];
        foreach ($objects as $i => $obj) {
            if ($obj->isString) {
                $out[] = $obj->str;
                continue;
            }
            try {
                $out[] = self::materializeValue($obj);
            } catch (\InvalidArgumentException $e) {
                throw new \InvalidArgumentException("value[$i]: {$e->getMessage()}", 0, $e);
            }
        }
        return $out;
    }

    public static function objectFromAny(mixed $v): DataObject
    {
        if ($v instanceof TypedValue) {
            return DataObject::integer($v->otype, $v->val);
        }
        if ($v instanceof Bytes) {
            if ($v->data === '') {
                throw new \InvalidArgumentException(
                    'empty byte slice is not allowed (max ' . self::MAX_STRING_LEN . ' bytes)'
                );
            }
            if (strlen($v->data) > self::MAX_STRING_LEN) {
                throw new \InvalidArgumentException(
                    'byte slice length ' . strlen($v->data) . ' exceeds max ' . self::MAX_STRING_LEN
                );
            }
            return DataObject::string($v->data);
        }
        if (is_string($v)) {
            if ($v === '') {
                throw new \InvalidArgumentException(
                    'empty string is not allowed (max ' . self::MAX_STRING_LEN . ' bytes)'
                );
            }
            if (strlen($v) > self::MAX_STRING_LEN) {
                throw new \InvalidArgumentException(
                    'string length ' . strlen($v) . ' exceeds max ' . self::MAX_STRING_LEN
                );
            }
            return DataObject::string($v);
        }
        if (is_int($v)) {
            IntMath::ensureAvailable();
            return DataObject::integer(TypedValue::OTYPE_INT64, (string) $v);
        }
        throw new \InvalidArgumentException(
            'unsupported type ' . get_debug_type($v) . ' (integer or string up to ' . self::MAX_STRING_LEN . ' bytes)'
        );
    }

    private static function materializeValue(DataObject $obj): TypedValue
    {
        return match ($obj->otype) {
            TypedValue::OTYPE_UINT8 => TypedValue::u8($obj->val),
            TypedValue::OTYPE_UINT16 => TypedValue::u16($obj->val),
            TypedValue::OTYPE_UINT32 => TypedValue::u32($obj->val),
            TypedValue::OTYPE_UINT64 => TypedValue::u64($obj->val),
            TypedValue::OTYPE_INT8 => TypedValue::i8($obj->val),
            TypedValue::OTYPE_INT16 => TypedValue::i16($obj->val),
            TypedValue::OTYPE_INT32 => TypedValue::i32($obj->val),
            TypedValue::OTYPE_INT64 => TypedValue::i64($obj->val),
            default => throw new \InvalidArgumentException("invalid otype {$obj->otype}"),
        };
    }
}
