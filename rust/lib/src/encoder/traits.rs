use crate::err::MixError;


pub trait EncoderDecoder{
    fn encode(&self, id:u64)->Result<String,MixError>; 
    fn decode(&self, str:String)->Result<u64,MixError>;
}


pub trait Encoder{
    fn mix(&self)->String;
}

pub trait Decoder {
    fn unmix(&self)->u64;
}

impl Encoder for u64{
    // 整数直接转换成字符串
    fn mix(&self)->String {
        format!("{:x}",self)        
    }
}
impl Decoder for &str{
    fn unmix(&self)->u64 {
        0
    }
}
impl Decoder for String {
    fn unmix(&self)->u64 {
        0
    }
}

