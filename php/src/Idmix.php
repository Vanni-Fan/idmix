<?php
namespace Vanni\Idmix;

/**
 * XID v1.1 编解码器主入口。
 */
class IdMix
{
    public const DEFAULT_ALPHABET = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';

    private RadixCodec $radix;
    public int $maxObjects;
    public int $maxVariants;
    public int $checkBits;
    public int $countBits;
    public int $variantBits;
    public int $checkMask;
    public int $countMask;
    public int $variantMask;
    public int $countShift;
    public int $variantShift;

    private function __construct(
        RadixCodec $radix,
        int $maxObjects = 511,
        int $maxVariants = 32,
        int $checkBits = 2,
    ) {
        $this->radix = $radix;
        $this->maxObjects = $maxObjects;
        $this->maxVariants = $maxVariants;
        $this->checkBits = $checkBits;
        $this->finalizeLayout();
    }

    public static function new(?string $alphabet = null): self
    {
        $alphabet ??= self::DEFAULT_ALPHABET;
        return new self(new RadixCodec($alphabet));
    }

    public static function withAlphabet(string $alphabet): self
    {
        return self::new($alphabet);
    }

    /** @param TypedValue ...$values */
    public function encode(TypedValue ...$values): string
    {
        if (count($values) < 1) {
            throw new \InvalidArgumentException('at least one value is required');
        }
        if (count($values) > $this->maxObjects) {
            throw new \InvalidArgumentException('too many objects');
        }
        $variantId = random_int(0, $this->maxVariants - 1);
        $data = XidCodec::encodeBinary($this, $values, $variantId);
        return $this->radix->encodeBytes($data);
    }

    /** @return TypedValue[] */
    public function decode(string $s): array
    {
        $data = $this->radix->decodeBytes($s);
        return XidCodec::decodeBinary($this, $data);
    }

    private function finalizeLayout(): void
    {
        $variantBits = $this->maxVariants <= 1 ? 1 : self::bitLen($this->maxVariants - 1);
        $countBits = $this->maxObjects <= 1 ? 1 : self::bitLen($this->maxObjects);
        $total = $this->checkBits + $countBits + $variantBits;
        if ($total > 16) {
            throw new \InvalidArgumentException("header layout exceeds 16 bits: $total");
        }
        $this->countBits = $countBits;
        $this->variantBits = $variantBits;
        $this->checkMask = (1 << $this->checkBits) - 1;
        $this->countMask = ((1 << $countBits) - 1) << $this->checkBits;
        $this->variantMask = ((1 << $variantBits) - 1) << ($this->checkBits + $countBits);
        $this->countShift = $this->checkBits;
        $this->variantShift = $this->checkBits + $countBits;
    }

    private static function bitLen(int $n): int
    {
        if ($n <= 0) {
            return 1;
        }
        $bits = 0;
        while ($n > 0) {
            $n >>= 1;
            $bits++;
        }
        return $bits;
    }
}
