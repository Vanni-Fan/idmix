<?php
namespace Vanni\Idmix;

/**
 * IDX v1.2：组合 IDX 二进制编解码与可插拔文本 Codec。
 */
final class IdMix
{
    public const DEFAULT_ALPHABET = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';

    private Idx $idx;
    private Codec $codec;

    public function __construct(?Idx $idx = null, ?Codec $codec = null)
    {
        $this->idx = $idx ?? Idx::new();
        $this->codec = $codec ?? new RadixCodec(self::DEFAULT_ALPHABET);
    }

    public static function new(?string $alphabet = null): self
    {
        $codec = $alphabet !== null
            ? new RadixCodec($alphabet)
            : new RadixCodec(self::DEFAULT_ALPHABET);
        return new self(null, $codec);
    }

    public static function withAlphabet(string $alphabet): self
    {
        return new self(null, new RadixCodec($alphabet));
    }

    public static function create(?Idx $idx = null, ?Codec $codec = null): self
    {
        return new self($idx, $codec);
    }

    public function idx(): Idx
    {
        return $this->idx;
    }

    public function codec(): Codec
    {
        return $this->codec;
    }

    /** @param mixed ...$values */
    public function encode(mixed ...$values): string
    {
        if (count($values) < 1) {
            throw new \InvalidArgumentException('at least one value is required');
        }
        $variantId = random_int(0, $this->idx->maxVariants - 1);
        return $this->codec->encode($this->encodeBinary($values, $variantId));
    }

    /** @param mixed ...$values */
    public function encodeWithVariant(int $variantId, mixed ...$values): string
    {
        return $this->codec->encode($this->encodeBinary($values, $variantId));
    }

    /** @return array<int, TypedValue|string> */
    public function decode(string $s): array
    {
        return $this->idx->decode($this->codec->decode($s));
    }

    /** @param mixed[] $values */
    public function encodeBinary(array $values, int $variantId): string
    {
        return $this->idx->encodeBinary(Number::normalizeObjects($values), $variantId);
    }
}
