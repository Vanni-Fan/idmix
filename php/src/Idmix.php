<?php
namespace Vanni\Idmix;

const INT_MAX16 = (1<<16)-1;
const INT_MAX32 = (1<<32)-1;
const INT_MAX56 = (1<<56)-1;

class Idmix{
    /**
     * 混淆一个整数ID
     */
    static function Mix(int $key, int $id):int{
        if ($id > INT_MAX56){
            throw new \Exception("数字[".$id."]已超出最大可混淆数字[".INT_MAX56."]");
        }
        $normal_obj = self::normalization($key, $id, false);
        
        // 第一次混淆
        $new_id = $normal_obj->key ^ $id;
        $key_sign = self::ParityCheck($new_id);

        // 第二次混淆
        $new_id = $new_id ^ $normal_obj->rand->slat;
        $rand_sign = self::ParityCheck($new_id);

        // 高位补码
        $padding_sign = 0;
        if ($id>INT_MAX16 && $id<=INT_MAX32 && $new_id <= INT_MAX16){
            $new_id |= 0x1_0000;
            $padding_sign = 1;
        }elseif($id > INT_MAX32 && $new_id <= INT_MAX32){
            $new_id |= 0x1_0000_0000;
            $padding_sign = 1;
        }

        // 管理位
        $manage_bit = ($normal_obj->rand->rand<<3) | ($padding_sign<<2) | ($key_sign<<1) | $rand_sign;

        // 最终ID
        return ($new_id<<8) + $manage_bit;

    }
    /**
     * 解码混淆过的整数ID
     */
    static function Unmix(int $key, int $id):int{
        $normal_obj = self::normalization($key, $id, true);

        if ($normal_obj->rand->rand_check != self::ParityCheck($normal_obj->id)){
            throw new \Exception("校验失败[randsalt]");
        }
        $new_id = $normal_obj->id ^ $normal_obj->rand->slat;
        
        if ($normal_obj->rand->key_check != self::ParityCheck($new_id)){
            throw new \Exception("校验失败[key]");
        }
        return $normal_obj->key ^ $new_id;
    }
    /**
     * 解码一个字符串为整数
     */
    static function Decode(int $key, string $str, Encoder $encoder=null):int{
        if (!$encoder){
            $encoder = new BaseEncoder();
        }
        $id = $encoder->Decode($str);
        return self::Unmix($key, $id);
    }
    /**
    * 编码一个整数为字符串
     */
    static function Encode(int $key, int $id, Encoder $encoder=null):string{
        if (!$encoder){
            $encoder = new BaseEncoder();
        }
        $newId = self::Mix($key, $id);
        return $encoder->Encode($newId);
    }
    /**
     * 获得一个整数的奇偶校验位：二进制中1的个数，奇数返回1； 偶数返回0
     */
    static function ParityCheck(int $id):int{
        $bits = PHP_INT_SIZE * 8;
        $result = 0;
        for ($i=0; $i<$bits; $i++){
            if (((1<<$i) & $id) >0){
                $result ^= 1;
            }
        }
        return $result;
    }

    /**
     * 规整化数据结构，key 和 随机盐
     */
    static protected function normalization(int $user_key, string|int $id_or_str, bool $is_decode):Normalization{
        // 初始化变量
        [$key,$slat,$id,$rand,$padding,$key_check,$rand_check] = [0,0,0,0,0,0,0];
        
        if ($is_decode){ // 编码
            $id          = $id_or_str >> 8;
            $rand        = ($id_or_str&0xFF) >> 3;
            $padding     = ($id_or_str&0b100) >> 2;
            $key_check   = ($id_or_str&0b10) >> 1;
            $rand_check  = $id_or_str&0b1;
        }else{ // 生成随机数
            $id = $id_or_str;
            $rand = mt_rand(0,31);
            $padding = 0;
        }

        // 对齐
        if ($id <= INT_MAX16) {
            $key = $user_key & INT_MAX16;
            $slat = ($rand << 8) ^ $key;
        } elseif ($id <= INT_MAX32) {
            $key = $user_key & INT_MAX32;
            $slat = ($rand << 16) ^ $key;
            if ($padding == 1 && $is_decode) { // 原本32位整数，但是混淆后小于 16 位，那么会在 17 位置 1 以便正确解析大小
                $id = $id & 0xFFFF; // 补码， 32位 => 0xFFFF
            }
        } else {
            $key = $user_key & INT_MAX56;
            $slat = ($rand << 32) ^ $key;
            if ($padding == 1 && $is_decode) { // 原本64位整数，但是混淆后小于 32 位，那么会在 33 位置 1 以便正确解析大小
                $id = $id & 0xFFFF_FFFF; // 64位 => 0x1FFFFFFFF
            }
        }

        return new Normalization(
            $key,
            new RandSlat($slat,$rand,$padding,$key_check,$rand_check),
            $id
        );        
    }
}