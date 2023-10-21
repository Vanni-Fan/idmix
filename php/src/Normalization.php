<?php

namespace Vanni\Idmix;

class Normalization{
    public function __construct(
        public readonly int $key,
        public readonly RandSlat $rand,
        public readonly int $id
    ){}
}