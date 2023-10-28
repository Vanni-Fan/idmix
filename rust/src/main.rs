use idmix::idmixer;

fn main() {
    let password = 12345678962342344;
    let id_before = 123123345234;

    // 基础用法
    use idmix::encoder::traits::IntEncoder; // 整数加密使用的特质
    let x = id_before.encode(password).unwrap(); // 整数加密

    use idmix::encoder::traits::StrDecoder; // 字符串解密使用的特质
    let id_after = x.decode(password).unwrap(); // 整数解密
    println!("[{}] => [{}] => [{}]", id_before, x, id_after);

    // 自定义用法，创建自己的编码器
    use idmix::encoder::custom::CustomEncoder; // 自定义编解码器
    let encoder = CustomEncoder::new("0123456789abcdef").unwrap();
    let x = idmixer::encode(password, id_before, &encoder).unwrap();
    let id_after = idmixer::decode(password, x.as_str(), &encoder).unwrap();
    println!("[{}] => [{}] => [{}]", id_before, x, id_after);
}