const MAX_U16:u64 = (u16::MAX) as u64;
const MAX_U32:u64 = (u32::MAX) as u64;
const MAX_U56:u64 = ((1u64<<56)-1) as u64;
const MAX_U64:u64 = (u64::MAX) as u64;

// pub mod encoder;
pub mod encoder;
pub mod idmix;
pub mod err;


#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn it_works() {
        assert_eq!(2+2, 4);
    }
}
