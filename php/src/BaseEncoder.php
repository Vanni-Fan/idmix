<?php
namespace Vanni\Idmix;
// 默认的编码器是 10 进制和 36 进制的转换
class BaseEncoder implements Encoder{
    function Decode(string $str):int{
        return intval($str, 36);
    }
    function Encode(int $id):string{
        return base_convert($id, 10, 36);
    }
}
