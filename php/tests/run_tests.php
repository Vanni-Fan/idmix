<?php
declare(strict_types=1);

require_once __DIR__ . '/../src/IntMath.php';
require_once __DIR__ . '/../src/DataObject.php';
require_once __DIR__ . '/../src/Bytes.php';
require_once __DIR__ . '/../src/Number.php';
require_once __DIR__ . '/../src/TypedValue.php';
require_once __DIR__ . '/../src/Codec.php';
require_once __DIR__ . '/../src/Base64Codec.php';
require_once __DIR__ . '/../src/RadixCodec.php';
require_once __DIR__ . '/../src/Idx.php';
require_once __DIR__ . '/../src/Idmix.php';

use Vanni\Idmix\Bytes;
use Vanni\Idmix\IdMix;
use Vanni\Idmix\Idx;
use Vanni\Idmix\IntMath;
use Vanni\Idmix\Number;
use Vanni\Idmix\TypedValue;
use function Vanni\Idmix\encodeBytes;
use function Vanni\Idmix\decodeString;

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
    assert_true($a === $b, $msg);
}

function assert_throws(callable $fn, string $msg): void
{
    try {
        $fn();
        fwrite(STDERR, "FAIL: expected exception: $msg\n");
        exit(1);
    } catch (\Throwable) {
        echo "OK: $msg\n";
    }
}

/** @return array<string, mixed> */
function load_vectors(): array
{
    $path = dirname(__DIR__, 2) . '/testdata/cross_language_vectors.json';
    return json_decode(file_get_contents($path), true, 512, JSON_THROW_ON_ERROR);
}

function materialize_input(array $entry): mixed
{
    if (!empty($entry['str'])) {
        return $entry['str'];
    }
    return match ((int) $entry['otype']) {
        0 => TypedValue::u8($entry['val']),
        1 => TypedValue::u16($entry['val']),
        2 => TypedValue::u32($entry['val']),
        3 => TypedValue::u64($entry['val']),
        4 => TypedValue::i8($entry['val']),
        5 => TypedValue::i16($entry['val']),
        6 => TypedValue::i32($entry['val']),
        7 => TypedValue::i64($entry['val']),
        default => throw new InvalidArgumentException('bad otype'),
    };
}

function materialize_want(array $entry): mixed
{
    if (!empty($entry['str'])) {
        return $entry['str'];
    }
    return match ((int) $entry['otype']) {
        0 => TypedValue::u8($entry['val']),
        1 => TypedValue::u16($entry['val']),
        2 => TypedValue::u32($entry['val']),
        3 => TypedValue::u64($entry['val']),
        4 => TypedValue::i8($entry['val']),
        5 => TypedValue::i16($entry['val']),
        6 => TypedValue::i32($entry['val']),
        7 => TypedValue::i64($entry['val']),
        default => throw new InvalidArgumentException('bad otype'),
    };
}

function assert_value(mixed $want, mixed $got, string $msg): void
{
    if ($want instanceof TypedValue && $got instanceof TypedValue) {
        assert_true($want->otype === $got->otype && $want->val === $got->val, $msg);
        return;
    }
    assert_eq($want, $got, $msg);
}

IntMath::ensureAvailable();
$m = IdMix::new();
$idx = Idx::new();

$spec = [TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)];
$data = $m->encodeBinary($spec, 0);
assert_eq($data, "\x80\x03\x22\x47\xB5\x1F", 'spec example binary');

$cases = [
    'uint32_max' => [TypedValue::u32('4294967295')],
    'int32_min' => [TypedValue::i32('-2147483648')],
    'int64_min' => [TypedValue::i64('-9223372036854775808')],
    'int64_max' => [TypedValue::i64('9223372036854775807')],
    'uint64_max' => [TypedValue::u64('18446744073709551615')],
];
foreach ($cases as $name => $values) {
    $decoded = $m->decode($m->encode(...$values));
    assert_value($values[0], $decoded[0], "round trip $name");
}

$vectors = load_vectors();
$m2 = IdMix::new($vectors['alphabet']);
foreach ($vectors['cases'] as $case) {
    $want = array_map(materialize_want(...), $case['values']);
    $got = $m2->decode($case['encoded']);
    assert_true(count($want) === count($got), 'cross-language ' . $case['name'] . ' count');
    foreach ($want as $i => $w) {
        assert_value($w, $got[$i], 'cross-language ' . $case['name'] . "[$i]");
    }
    $inputs = array_map(materialize_input(...), $case['values']);
    $enc = $m2->encodeWithVariant((int) ($case['variant'] ?? 0), ...$inputs);
    assert_eq($case['encoded'], $enc, 'cross-language encode ' . $case['name']);
}

// --- Idx boundary tests ---
$ok63 = str_repeat('a', Number::MAX_STRING_LEN);
$tooLong = str_repeat('b', Number::MAX_STRING_LEN + 1);

assert_throws(static fn() => $idx->encode(''), 'empty string rejected');
assert_throws(static fn() => $idx->encode(new Bytes('')), 'empty bytes rejected');

$data = $idx->encodeWithVariant(0, 'x');
$out = $idx->decode($data);
assert_eq('x', $out[0], 'len 1 string ok');

$data = $idx->encodeWithVariant(0, $ok63);
assert_eq(1 + 1 + Number::MAX_STRING_LEN, strlen($data), 'len 63 binary size');
$out = $idx->decode($data);
assert_eq($ok63, $out[0], 'len 63 string ok');

assert_throws(static fn() => $idx->encode($tooLong), 'len 64 string rejected');
assert_throws(static fn() => $idx->encode(new Bytes($tooLong)), 'len 64 bytes rejected');

$str63 = $m->encodeWithVariant(0, $ok63);
$out = $m->decode($str63);
assert_eq($ok63, $out[0], 'idmix end-to-end 63');
assert_throws(static fn() => $m->encode($tooLong), 'idmix len 64 rejected');

foreach ([0, 256] as $n) {
    assert_throws(static fn() => Idx::new($n), "maxObjects=$n invalid");
}
$idx2 = Idx::new(2);
assert_throws(static fn() => $idx2->encode(TypedValue::u8(1), TypedValue::u8(2), TypedValue::u8(3)), 'encode over maxObjects');
$data = $idx2->encodeWithVariant(0, TypedValue::u8(1), TypedValue::u8(2));
assert_true((ord($data[0]) & 0x80) !== 0, 'two objects use 2-byte header');
assert_eq(2, ord($data[1]), 'count byte is 2');

foreach ([0, 33] as $n) {
    assert_throws(static fn() => Idx::new(null, $n), "maxVariants=$n invalid");
}
$idx4 = Idx::new(null, 4);
assert_throws(static fn() => $idx4->encodeWithVariant(4, TypedValue::u8(1)), 'variant out of range');
$data = $idx4->encodeWithVariant(3, TypedValue::u8(42));
$out = $idx4->decode($data);
assert_value(TypedValue::u8(42), $out[0], 'variant at limit');

$large = Idx::new(null, 32);
$data = $large->encodeWithVariant(20, TypedValue::u8(7));
$small = Idx::new(null, 16);
assert_throws(static fn() => $small->decode($data), 'decode rejects high variant');

foreach ([0, 3] as $n) {
    assert_throws(static fn() => Idx::new(null, null, $n), "checkBits=$n invalid");
}
$idx1 = Idx::new(null, null, 1);
assert_eq(1, $idx1->checkBits, 'checkBits 1');
assert_eq(0x01, $idx1->checkMask, 'checkMask 0x01');
$data = $idx1->encodeWithVariant(0, TypedValue::u16(5), TypedValue::i64(-1));
$out = $idx1->decode($data);
assert_value(TypedValue::u16(5), $out[0], 'checkBits 1 roundtrip');
$data = $idx1->encodeWithVariant(0, TypedValue::u32(1));
$data[strlen($data) - 1] = chr(ord($data[strlen($data) - 1]) ^ 0x01);
assert_throws(static fn() => $idx1->decode($data), 'checkBits 1 tamper rejected');

assert_eq(encodeBytes("\xde\xad"), encodeBytes("\xde\xad", null), 'encodeBytes default codec');

echo "All tests passed.\n";
