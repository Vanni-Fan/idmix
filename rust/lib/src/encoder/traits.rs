use crate::{err::MixError, idmixer};

use super::custom::CustomEncoder;


pub trait EncoderDecoder{
    fn encode(&self, id:u64)->Result<String,MixError>; 
    fn decode(&self, str:String)->Result<u64,MixError>;
}


pub trait IntEncoder{
    fn mix(&self, password:u64)->Result<u64,MixError>;
    fn unmix(&self, password:u64)->Result<u64,MixError>;
    fn encode(&self, password:u64)->Result<String,MixError>;
}

pub trait StrDecoder {
    fn decode(&self, password:u64)->Result<u64,MixError>;
}

const BASE36: &str = "0123456789abcdefghijklmnopqrstuvwxyz";

impl IntEncoder for u64{
    fn mix(&self, password:u64)->Result<u64,MixError> {
        idmixer::mix(password, *self)
        // Ok(1)
    }
    fn unmix(&self, password:u64)->Result<u64,MixError> {
        idmixer::unmix(password, *self)
    }
    fn encode(&self, password:u64)->Result<String,MixError> {
        let encoder = CustomEncoder::new(BASE36).unwrap();
        idmixer::encode(password, *self, &encoder)
    }
}
impl StrDecoder for &str{
    fn decode(&self, password:u64)->Result<u64,MixError> {
        let encoder = CustomEncoder::new(BASE36).unwrap();
        idmixer::decode(password, *self, &encoder)
    }
}
impl StrDecoder for String {
    fn decode(&self, password:u64)->Result<u64,MixError> {
        let encoder = CustomEncoder::new(BASE36).unwrap();
        idmixer::decode(password, self.as_str(), &encoder)
    }
}
