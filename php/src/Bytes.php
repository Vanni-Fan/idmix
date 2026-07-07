<?php
namespace Vanni\Idmix;

/** 原始字节串包装（与 UTF-8 文本 string 区分）。 */
final class Bytes
{
    public function __construct(public readonly string $data)
    {
    }
}
