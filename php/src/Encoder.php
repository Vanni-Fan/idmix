<?php
namespace Vanni\Idmix;
interface Encoder{
    function Decode(string $str):int;
    function Encode(int $id):string;
}