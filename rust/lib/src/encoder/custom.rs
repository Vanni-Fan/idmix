use std::collections::HashMap;
use std::collections::VecDeque;
use std::fmt;

use crate::MAX_U56;
use crate::MAX_U64;
use crate::err::MixError;

use super::traits::EncoderDecoder;

#[derive(Debug)]
pub struct CustomEncoder{
    base:Vec<char>,
    mapping:HashMap<char,u8>
}
impl std::fmt::Display for CustomEncoder {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "base:{:?}\nmapping:{:?}", self.base, self.mapping)
    }
}
impl CustomEncoder {
    pub fn new(base:&str)->Result<Self,MixError>{
        let chars = base.chars();
        let mut obj = CustomEncoder{
            base:Vec::new(),
            mapping: HashMap::new()
        };
        let mut j = 0;
        for (i,c) in chars.enumerate(){
            if obj.mapping.contains_key(&c){
                return Err(MixError{msg: String::from(format!("{}{}", &"进制字符串中不允许有相同的字符：" , c))});
            }
            obj.base.insert(i, c);
            obj.mapping.insert(c,i as u8);
            j += 1;
        }
        if j<2{
            return Err(MixError{msg: String::from("进制必须大于2个字符，比如最小的二级制也是0和1两个字符串")});
        }
        Ok(obj)
    }
}
impl EncoderDecoder for CustomEncoder {
    fn encode(&self, id:u64)->Result<String,MixError> {
        let mut result = VecDeque::new();
        let b = self.base.len() as u64;
        let mut n = id;
        while n > 0 {
            let quotient = n / b;
            let remainder = n % b;

            // 将余数转换为字符,并添加到结果字符串的最前面
            result.push_front(self.base[remainder as usize] as char);
            
            n = quotient;
        }
        
        // 如果结果为空,则原数为 0
        if result.is_empty() {
            result.push_front(self.base[0] as char);
        }
        
        // 返回结果
        let s: String = result.into_iter().collect();
        Ok(String::from(s))
    }
    fn decode(&self, str:String)->Result<u64,MixError> {
        let mut result = 0u128;
        let base = self.mapping.len() as u128;
    
        let chars: Vec<char> = str.chars().collect();
        let length = chars.len();
    
        for (index, char) in chars.iter().enumerate() {
          let value = self.mapping.get(char);
          if value.is_none() {
            return Err(MixError{msg: String::from(format!("无效字符:{}",  char))});
          }
          let value = value.unwrap();
          let position = length - index - 1;
          result += base.pow(position as u32) * (*value as u128);
        }
    
        if result > MAX_U64 as u128 {
            return Err(MixError{msg: String::from(format!("字符串[ {} ]，转成的整数[ {} ]，已超出最大整数范围：[0,{}]",  str, result, MAX_U56))});
        }
    
        Ok(result as u64)
      
    }
}

