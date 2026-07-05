<?php
namespace Vanni\Idmix;

/**
 * 自定义进制文本层：二进制块 ↔ 字符串。
 */
class RadixCodec
{
    private int $base;
    /** @var string[] */
    private array $chars;
    /** @var array<string, int> */
    private array $fromCustom;

    public function __construct(string $alphabet)
    {
        $runes = preg_split('//u', $alphabet, -1, PREG_SPLIT_NO_EMPTY);
        if (count($runes) < 2) {
            throw new \InvalidArgumentException('alphabet must have at least 2 unique characters');
        }
        $this->base = count($runes);
        $this->chars = $runes;
        $this->fromCustom = [];
        foreach ($runes as $i => $r) {
            if (isset($this->fromCustom[$r])) {
                throw new \InvalidArgumentException("alphabet contains duplicate character $r");
            }
            $this->fromCustom[$r] = $i;
        }
    }

    public function encodeBytes(string $data): string
    {
        if ($data === '') {
            return $this->chars[0];
        }
        $len = strlen($data);
        $wrapped = pack('n', $len) . $data;
        $decimal = self::bytesToDecimal($wrapped);
        return $this->intToString($decimal);
    }

    public function decodeBytes(string $s): string
    {
        if ($s === '') {
            throw new \InvalidArgumentException('empty string');
        }
        $decimal = $this->stringToInt($s);
        $raw = self::decimalToBytes($decimal);
        for ($pad = 0; $pad <= 1; $pad++) {
            $buf = str_repeat("\0", $pad) . $raw;
            if (strlen($buf) < 2) {
                continue;
            }
            $unpacked = unpack('n', substr($buf, 0, 2));
            $dataLen = $unpacked[1];
            if (strlen($buf) !== 2 + $dataLen) {
                continue;
            }
            return substr($buf, 2);
        }
        throw new \InvalidArgumentException('invalid encoded data length');
    }

    private function intToString(string $decimal): string
    {
        if (bccomp($decimal, '0') === 0) {
            return $this->chars[0];
        }
        $chars = [];
        $base = (string)$this->base;
        $zero = '0';
        while (bccomp($decimal, $zero) > 0) {
            $rem = (int)bcmod($decimal, $base);
            $chars[] = $this->chars[$rem];
            $decimal = bcdiv($decimal, $base, 0);
        }
        return implode('', array_reverse($chars));
    }

    private function stringToInt(string $s): string
    {
        $runes = preg_split('//u', $s, -1, PREG_SPLIT_NO_EMPTY);
        $n = '0';
        $base = (string)$this->base;
        foreach ($runes as $r) {
            if (!isset($this->fromCustom[$r])) {
                throw new \InvalidArgumentException("invalid character $r");
            }
            $n = bcadd(bcmul($n, $base), (string)$this->fromCustom[$r]);
        }
        return $n;
    }

    private static function bytesToDecimal(string $bytes): string
    {
        $result = '0';
        foreach (str_split($bytes) as $byte) {
            $result = bcadd(bcmul($result, '256'), (string)ord($byte));
        }
        return $result;
    }

    private static function decimalToBytes(string $decimal): string
    {
        if (bccomp($decimal, '0') === 0) {
            return '';
        }
        $bytes = [];
        while (bccomp($decimal, '0') > 0) {
            $rem = (int)bcmod($decimal, '256');
            $bytes[] = chr($rem);
            $decimal = bcdiv($decimal, '256', 0);
        }
        return implode('', array_reverse($bytes));
    }
}
