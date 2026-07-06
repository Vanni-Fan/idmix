<?php
declare(strict_types=1);

require_once __DIR__ . '/../src/IntMath.php';
require_once __DIR__ . '/../src/Idmix.php';
require_once __DIR__ . '/../src/TypedValue.php';
require_once __DIR__ . '/../src/RadixCodec.php';
require_once __DIR__ . '/../src/XidCodec.php';

use Vanni\Idmix\IdMix;
use Vanni\Idmix\IntMath;
use Vanni\Idmix\TypedValue;
use Vanni\Idmix\XidCodec;

function assert_true(bool $cond, string $msg): void
{
    if (!$cond) {
        fwrite(STDERR, "FAIL: $msg\n");
        exit(1);
    }
    echo "OK: $msg\n";
}

function assert_eq(mixed $a, mixed $b, string $msg): void
{
    assert_true($a == $b, $msg);
}

/** @return array<string, mixed> */
function load_vectors(): array
{
    $path = dirname(__DIR__, 2) . '/testdata/cross_language_vectors.json';
    return json_decode(file_get_contents($path), true, 512, JSON_THROW_ON_ERROR);
}

/** @param TypedValue[] $want @param TypedValue[] $got */
function assert_values(array $want, array $got, string $msg): void
{
    assert_true(count($want) === count($got), "$msg count");
    foreach ($want as $i => $w) {
        assert_true($w->otype === $got[$i]->otype && $w->val === $got[$i]->val, "$msg[$i]");
    }
}

IntMath::ensureAvailable();
$m = IdMix::new();

$spec = [TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)];
$data = XidCodec::encodeBinary($m, $spec, 0);
assert_eq($data, "\x0F\x00\x22\x47\xB5\x1F", 'spec example binary');

$cases = [
    'uint32_max' => [TypedValue::u32('4294967295')],
    'int32_min' => [TypedValue::i32('-2147483648')],
    'int64_min' => [TypedValue::i64('-9223372036854775808')],
    'int64_max' => [TypedValue::i64('9223372036854775807')],
    'uint64_max' => [TypedValue::u64('18446744073709551615')],
];
foreach ($cases as $name => $values) {
    $decoded = $m->decode($m->encode(...$values));
    assert_values($values, $decoded, "round trip $name");
}

$vectors = load_vectors();
$m2 = IdMix::new($vectors['alphabet']);
foreach ($vectors['cases'] as $case) {
    $want = array_map(
        static fn(array $v) => match ((int) $v['otype']) {
            0 => TypedValue::u8($v['val']),
            1 => TypedValue::u16($v['val']),
            2 => TypedValue::u32($v['val']),
            3 => TypedValue::u64($v['val']),
            4 => TypedValue::i8($v['val']),
            5 => TypedValue::i16($v['val']),
            6 => TypedValue::i32($v['val']),
            7 => TypedValue::i64($v['val']),
            default => throw new InvalidArgumentException('bad otype'),
        },
        $case['values']
    );
    $got = $m2->decode($case['encoded']);
    assert_values($want, $got, 'cross-language ' . $case['name']);
}

echo "All tests passed.\n";
