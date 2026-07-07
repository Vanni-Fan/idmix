<?php
namespace Vanni\Idmix;

/**
 * 二进制与文本之间的编解码器（idmix 文本层插拔点）。
 */
interface Codec
{
    public function encode(string $data): string;

    public function decode(string $s): string;
}

/** 将任意二进制编码为文本；codec 不传时使用默认 RadixCodec。 */
function encodeBytes(string $data, ?Codec $codec = null): string
{
    return resolveCodec($codec)->encode($data);
}

/** 将文本还原为二进制；codec 不传时使用默认 RadixCodec。 */
function decodeString(string $s, ?Codec $codec = null): string
{
    return resolveCodec($codec)->decode($s);
}

function resolveCodec(?Codec $codec = null): Codec
{
    static $defaultCodec = null;
    if ($codec !== null) {
        return $codec;
    }
    if ($defaultCodec === null) {
        $defaultCodec = new RadixCodec(IdMix::DEFAULT_ALPHABET);
    }
    return $defaultCodec;
}

/**
 * 由闭包实现的 Codec，便于包装 AES/XOR 等自定义逻辑。
 */
final class FuncCodec implements Codec
{
    /** @var callable(string): string|null */
    private $encodeFn;
    /** @var callable(string): string|null */
    private $decodeFn;

    /**
     * @param callable(string): string|null $encodeFn
     * @param callable(string): string|null $decodeFn
     */
    public function __construct(?callable $encodeFn = null, ?callable $decodeFn = null)
    {
        $this->encodeFn = $encodeFn;
        $this->decodeFn = $decodeFn;
    }

    public function encode(string $data): string
    {
        if ($this->encodeFn === null) {
            throw new \RuntimeException('codec function is nil');
        }
        return ($this->encodeFn)($data);
    }

    public function decode(string $s): string
    {
        if ($this->decodeFn === null) {
            throw new \RuntimeException('codec function is nil');
        }
        return ($this->decodeFn)($s);
    }
}
