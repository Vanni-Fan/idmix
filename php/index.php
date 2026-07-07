<?php
spl_autoload_register(function (string $class): void {
    $prefix = 'Vanni\\Idmix\\';
    if (!str_starts_with($class, $prefix)) {
        return;
    }
    $relative = substr($class, strlen($prefix));
    $path = __DIR__ . '/src/' . str_replace('\\', '/', $relative) . '.php';
    if (is_file($path)) {
        require $path;
    }
});

use Vanni\Idmix\IdMix;
use Vanni\Idmix\Idx;
use Vanni\Idmix\TypedValue;

$m = IdMix::new();
$values = [TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)];
$str = $m->encode(...$values);
$out = $m->decode($str);
printf("规范示例: %s => %s\n", $str, implode(', ', array_map(
    static fn($v) => $v instanceof TypedValue ? $v->val : $v,
    $out
)));

$large = [TypedValue::u32('2000000000')];
$str = $m->encode(...$large);
$out = $m->decode($str);
printf("单值 uint32(2000000000): %s => %s\n", $str, $out[0]->val);

$custom = IdMix::withAlphabet('abcd');
$values = [TypedValue::u16(100), TypedValue::i32(-10), TypedValue::u8(3)];
$str = $custom->encode(...$values);
$out = $custom->decode($str);
printf("四进制: %s => %s\n", $str, implode(', ', array_map(fn($v) => $v->val, $out)));

$idx = Idx::new();
$data = $idx->encodeWithVariant(0, TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40));
$hex = strtoupper(implode(' ', str_split(bin2hex($data), 2)));
$want = '80 03 22 47 B5 1F';
printf("规范二进制 (variant=0): %s %s\n", $hex, $hex === $want ? 'OK' : "WANT $want");

$strExample = $m->encodeWithVariant(0, 'hello', TypedValue::u16(5), '世界');
$decoded = $m->decode($strExample);
printf("字符串示例: %s => %s\n", $strExample, json_encode($decoded, JSON_UNESCAPED_UNICODE));
