<?php
namespace Vanni\Idmix;

class CustomEncoder implements Encoder{
    private array $bases;
    private array $mapping;
    public function __construct(string $bases){
        if (mb_strlen($bases)<2){
            throw new \Exception("进制必须大于2个字符，比如最小的二级制也是0和1两个字符串");
        }
        $this->bases = mb_str_split($bases);
        foreach($this->bases as $index=>$char){
            if(isset($this->mapping[$char])){
                throw new \Exception("进制字符串中不允许有相同的字符：".$char);
            }
            $this->mapping[$char] = $index;
        }
    }
    // 解码
    function Decode(string $str):int{
        $result = 0;
        $base = count($this->mapping);
        $chars = mb_str_split($str);
        $length = count($chars);
        foreach($chars as $index=>$char){
            if(!isset($this->mapping[$char])){
                throw new \Exception("无效字符: ".$char);
            }
            $value = $this->mapping[$char];
            $position = $length - 1 - $index;

            // 计算幂
            $pow = bcmul(bcpow($base, $position), $value);

            // 总计
            $result = bcadd($result, $pow);
        }
        if (bccomp($result, PHP_INT_MAX) > 0){
            throw new \Exception(sprintf("字符串[ %s ]，转成的整数[ %s ]，已超出最大整数范围：[0,%d]", $str, $result, PHP_INT_MAX));
        }
        return intval($result);
    }
    // 编码
    function Encode(int $id):string{
        $result = "";
        $base = count($this->mapping);
        while ($id > 0) {
            // 计算商和余数
            $quotient = bcdiv($id, $base);
            $remainder = bcmod($id, $base);
    
            // 将余数转换成字符，并添加到结果字符串的最前面
            $result.=$this->bases[$remainder];
            $id = $quotient;
        }
    
        // 如果结果字符串为空，则说明原数字为 0
        if (mb_strlen($result) == 0) {
            $result = $this->bases[0];
        }
    
        // 将结果字符串反转
        $strList = mb_str_split($result);
        for ($i=0, $j=count($strList)-1; $i<$j; $i++,$j--) {
            [$strList[$i], $strList[$j]] = [$strList[$j], $strList[$i]];
        }
        return implode($strList);
    }
}