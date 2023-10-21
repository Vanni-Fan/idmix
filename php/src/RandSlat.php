<?php
namespace Vanni\Idmix;

class RandSlat {
    public function __construct(
        public readonly int $slat,
        public readonly int $rand,
        public readonly int $padding,
        public readonly int $key_check,
        public readonly int $rand_check
    ){}
}