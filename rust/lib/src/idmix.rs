use rand::Rng;
use std::mem::size_of;
use crate::{ err::MixError, MAX_U16, MAX_U32, MAX_U56 };

#[derive(Default, Debug)]
struct RandType {
    slat: u64,
    rand: u8,
    padding: u8,
    key_check: u8,
    rand_check: u8,
}

fn normalization(user_key: u64, src_id: u64, is_decode: bool) -> (u64, RandType, u64) {
    let key: u64;
    let mut id = src_id;
    let mut rand_obj = RandType::default();
    if is_decode {
        rand_obj = RandType {
            slat: 0,
            rand: ((src_id & 0xff) as u8) >> 3,
            padding: ((src_id & 0b100) as u8) >> 2,
            key_check: ((src_id ^ 0b10) as u8) >> 1,
            rand_check: (src_id & 0b1) as u8,
        };
        id = src_id >> 8;
    } else {
        let mut rng = rand::thread_rng();
        rand_obj.rand = rng.gen_range(0..32);
    }
    if id <= MAX_U16 {
        key = user_key & MAX_U16;
        rand_obj.slat = ((rand_obj.rand as u64) << 8) ^ key;
    } else if id <= MAX_U32 {
        key = user_key & MAX_U32;
        rand_obj.slat = ((rand_obj.rand as u64) << 16) ^ key;
        if rand_obj.padding == 1 && is_decode {
            id = id & 0xffff;
        }
    } else {
        key = user_key & MAX_U56;
        rand_obj.slat = ((rand_obj.rand as u64) << 32) ^ key;
        if rand_obj.padding == 1 && is_decode {
            id = id & 0xffff_ffff;
        }
    }
    (key, rand_obj, id)
}

pub fn mix(key: u64, id: u64) -> Result<u64, MixError> {
    if id > MAX_U56 {
        return Err(MixError {
            msg: String::from(format!("数字[{}]已超出最大可混淆数字[{}]", id, MAX_U56)),
        });
    }
    let (k, rand_obj, id) = normalization(key, id, false);
    println!("{},{:?},{}", k, rand_obj, id);

    // 第一次密码混淆
    let mut new_id = k ^ id;
    let key_sign = parity_check(new_id);

    // 第二次随机盐混淆
    new_id = new_id ^ rand_obj.slat;
    let rand_sign = parity_check(new_id);

    // 高位补码
    let mut padding_sign = 0u8;
    if id > MAX_U16 && id <= MAX_U32 && new_id <= 0xffff {
        new_id |= 0x1_0000;
        padding_sign = 1;
    } else if id > MAX_U32 && new_id <= 0xffff_ffff {
        new_id |= 0x1_0000_0000;
        padding_sign = 1;
    }
    let manager_bit = (rand_obj.rand << 3) | (padding_sign << 2) | (key_sign << 1) | rand_sign;
    new_id = (new_id << 8) + (manager_bit as u64);
    Ok(new_id)
}

pub fn unmix(key: u64, id: u64) -> Result<u64, MixError> {
    let (k, rand_obj, id) = normalization(key, id, false);
    println!("{},{:?},{}", k, rand_obj, id);

    // 第一次解密
    if rand_obj.rand_check != parity_check(id) {
        return Err(MixError { msg: String::from("校验失败[randsalt]") });
    }

    // 第二次解密
    let new_id = id ^ rand_obj.slat;
    if rand_obj.key_check != parity_check(new_id) {
        return Err(MixError { msg: String::from("校验失败[key]") });
    }

    return Ok(k ^ new_id);
}

// pub fn encode<T: EncoderDecoder>(key:u64, id:u64, encode:Option<T>)->&'static str{
//     let a = "asdf";
//     a
// }
// pub fn decode<T: EncoderDecoder>(key:u64, str:&str, encode:Option<T>)->u64{
//     // let n:Option<BaseEncoder> = None;
//     // encode(123, 123, n);
//     0
// }

// 对整数的二进制位进行奇偶校验，如果校验和为奇数返回1，偶数返回0
pub fn parity_check<T: Into<u64>>(id: T) -> u8 {
    let mut result = 0_u8;
    let bits = size_of::<T>() * 8;
    let val = id.into();
    for i in 0..bits {
        if val & (1 << i) > 0 {
            result ^= 1;
        }
    }
    result
}


pub struct Idmix{

}

