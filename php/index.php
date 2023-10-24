<?php
require 'vendor/autoload.php';

use Vanni\Idmix\Idmix;
$key = 1234567;
$sid = mt_rand(1000000000000,99999999999999);
$str = Idmix::Encode($key, $sid);
$did = Idmix::Decode($key, $str);

printf("原始ID:[%d]，字符串：[%s], 结果ID:[%d]，是否相等：[%s]\n",$sid,$str,$did,$sid==$did);

$my_encoder = new Vanni\Idmix\CustomEncoder("KLNMOPQRSTUVWXYZ-,.+=!@#$%^&*()_<>~自定义的中文加数字abcdefghijklnmopqrstuvwxyz0123456789ABCDEFGHIJ");
$str = Idmix::Encode($key, $sid, $my_encoder);
$did = Idmix::Decode($key, $str, $my_encoder);
printf("原始ID:[%d]，字符串：[%s], 结果ID:[%d]，是否相等：[%s]\n",$sid,$str,$did,$sid==$did);