<?php
namespace Vanni\Idmix;

/**
 * 使用标准 Base64 的二进制↔文本编解码器。
 */
final class Base64Codec implements Codec
{
    public static function new(): self
    {
        return new self();
    }

    public function encode(string $data): string
    {
        return base64_encode($data);
    }

    public function decode(string $s): string
    {
        $out = base64_decode($s, true);
        if ($out === false) {
            throw new \InvalidArgumentException('invalid base64 string');
        }
        return $out;
    }
}
