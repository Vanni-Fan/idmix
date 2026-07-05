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
use Vanni\Idmix\TypedValue;

$m = IdMix::new();
$values = [TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)];
$str = $m->encode(...$values);
$out = $m->decode($str);
printf("规范示例: %s => %s\n", $str, implode(', ', array_map(fn($v) => $v->val, $out)));

$large = [TypedValue::u32(2_000_000_000)];
$str = $m->encode(...$large);
$out = $m->decode($str);
printf("单值 uint32(2000000000): %s => %d\n", $str, $out[0]->val);

$custom = IdMix::withAlphabet('abcd');
$values = [TypedValue::u16(100), TypedValue::i32(-10), TypedValue::u8(3)];
$str = $custom->encode(...$values);
$out = $custom->decode($str);
printf("四进制: %s => %s\n", $str, implode(', ', array_map(fn($v) => $v->val, $out)));

// 规范二进制块校验 (variant=0)
$typed = [TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)];
$data = \Vanni\Idmix\XidCodec::encodeBinary($m, $typed, 0);
$hex = strtoupper(implode(' ', str_split(bin2hex($data), 2)));
$want = '0F 00 22 47 B5 1F';
printf("规范二进制 (variant=0): %s %s\n", $hex, $hex === $want ? 'OK' : "WANT $want");
