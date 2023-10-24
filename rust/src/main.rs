// use idmix::{self, MixError, EncoderDecoder};
use idmix::encoder::custom::CustomEncoder;
use idmix::encoder::traits::EncoderDecoder;
// use std::collections::HashMap;
// use std::collections::VecDeque;

// use idmix::encoder;


fn main() {
    // idmix::mix(2000000,3444);
    // let a:u64 = 1274654654646;
    // println!("{}", idmix::parity_check(126u8));

    let a = CustomEncoder::new("0123456789abcdef");
    match a {
        Err(e) => println!("{}",e),
        Ok(c)=>println!("{}",c.encode(5782456529).unwrap())
    }
    // println!("{:?}", a);
    // let b = CustomEncoder{
    //     base: Vec::new(),
    //     mapping: HashMap::new(),
    // };
    // b.encode(123);
    // idmix::encoder::custom_encoder;

    
}
