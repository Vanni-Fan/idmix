use rand::Rng;
use std::mem::size_of;
use crate::{ err::MixError, MAX_U16, MAX_U32, MAX_U56, encoder::traits::EncoderDecoder };

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
            key_check: ((src_id & 0b10) as u8) >> 1,
            rand_check: (src_id & 0b1) as u8,
        };
        id = src_id >> 8;
    } else {
        let mut rng = rand::thread_rng();
        rand_obj.rand = rng.gen_range(0..32);
    }
    if DEBUG && is_decode {
        println!("原始ID[{}]，去掉管理字节后的ID[{}]", src_id, id);
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

const DEBUG: bool = false;

pub fn mix(key: u64, id: u64) -> Result<u64, MixError> {
    if id > MAX_U56 {
        return Err(MixError {
            msg: String::from(format!("数字[{}]已超出最大可混淆数字[{}]", id, MAX_U56)),
        });
    }
    let (k, rand_obj, id) = normalization(key, id, false);
    if DEBUG {
        println!(
            "编码源信息：用户Key[{}]，规整Key[{}]，随机数[{}]，原始ID[{}]",
            key,
            k,
            rand_obj.rand,
            id
        );
    }

    // 第一次密码混淆
    let mut new_id = k ^ id;
    let key_sign = parity_check(new_id);
    if DEBUG {
        println!("加密码混淆：[{} => {:064b}]，校验位[{}]", new_id, new_id, key_sign);
    }

    // 第二次随机盐混淆
    new_id = new_id ^ rand_obj.slat;
    let rand_sign = parity_check(new_id);
    if DEBUG {
        println!(
            "随机盐混淆：[{} => {:064b}]，盐[{}]，校验位[{}]",
            new_id,
            new_id,
            rand_obj.slat,
            rand_sign
        );
    }

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
    if DEBUG {
        println!(
            "管理字节位：[{} => {:08b}]，随机数[{}],高位补码位[{}],加密码校验位[{}],随机盐校验位[{}]",
            manager_bit,
            manager_bit,
            rand_obj.rand,
            padding_sign,
            key_sign,
            rand_sign
        );
    }

    new_id = (new_id << 8) + (manager_bit as u64);
    if DEBUG {
        println!("最终生成数：[{} => {:064b}]", new_id, new_id);
    }
    Ok(new_id)
}

pub fn unmix(key: u64, id: u64) -> Result<u64, MixError> {
    let (k, rand_obj, id) = normalization(key, id, true);
    if DEBUG {
        println!(
            "解码源信息：用户Key[{}]，规整Key[{}]，随机数[{}]，原始ID[{}]",
            key,
            k,
            rand_obj.rand,
            id
        );
        println!(
            "管理字信息：随机数[{}],高位补码位[{:08b}],加密码校验位[{}],随机盐校验位[{}]",
            rand_obj.rand,
            rand_obj.padding,
            rand_obj.key_check,
            rand_obj.rand_check
        );
        println!(
            "随机盐校验：[{} => {:064b}]，盐[{}]，校验位[{}]",
            id,
            id,
            rand_obj.slat,
            rand_obj.rand_check
        );
    }

    // 第一次解密
    if rand_obj.rand_check != parity_check(id) {
        return Err(MixError { msg: String::from("校验失败[randsalt]") });
    }

    // 第二次解密
    let new_id = id ^ rand_obj.slat;
    if DEBUG {
        println!("加密码校验：[{} => {:064b}]，校验位[{}]", new_id, new_id, rand_obj.key_check);
    }
    if rand_obj.key_check != parity_check(new_id) {
        return Err(MixError { msg: String::from("校验失败[key]") });
    }

    return Ok(k ^ new_id);
}

pub fn encode(key: u64, id: u64, encode: &dyn EncoderDecoder) -> Result<String, MixError> {
    if DEBUG {
        println!("编码：id:{} ", id);
    }
    let new_id = mix(key, id)?;
    if DEBUG {
        println!("编码：id:{} mixid:{} ", id, new_id);
    }
    let rs = encode.encode(new_id)?;
    if DEBUG {
        println!("编码：id:{} mixid:{} str:{}", id, new_id, rs);
    }
    Ok(rs)
}

pub fn decode(key: u64, str: &str, encode: &dyn EncoderDecoder) -> Result<u64, MixError> {
    if DEBUG {
        println!("解码：str:{} ", str);
    }
    let new_id = encode.decode(str.to_string())?;
    if DEBUG {
        println!("解码：str:{} mixid:{} ", str, new_id);
    }
    let id = unmix(key, new_id)?;
    if DEBUG {
        println!("解码：str:{} mixid:{} id:{}", str, new_id, id);
    }
    Ok(id)
}

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
    if DEBUG {
        println!("整数{}的奇偶校验结果为{}", val, result);
    }
    result
}
