use idmix::{IdMix, Value};

fn main() {
    let m = IdMix::new().expect("create IdMix");

    let values = [Value::U16(5), Value::I64(-1), Value::U32(40)];
    let encoded = m.encode(&values).expect("encode");
    let decoded = m.decode(&encoded).expect("decode");
    println!("规范示例: {:?} => {:?} => {:?}", values, encoded, decoded);

    let large = [Value::U32(2_000_000_000)];
    let encoded = m.encode(&large).expect("encode");
    let decoded = m.decode(&encoded).expect("decode");
    println!("单值 uint32(2000000000): {:?} => {:?} => {:?}", large, encoded, decoded);

    let custom = IdMix::builder()
        .alphabet("abcd")
        .build()
        .expect("custom alphabet");
    let values = [Value::U16(100), Value::I32(-10), Value::U8(3)];
    let encoded = custom.encode(&values).expect("encode");
    let decoded = custom.decode(&encoded).expect("decode");
    println!("四进制: {:?} => {:?} => {:?}", values, encoded, decoded);
}
