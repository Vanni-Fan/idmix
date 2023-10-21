package idmix

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
)

type randType struct {
	slat      uint64
	rand      uint8
	padding   uint8
	keyCheck  uint8
	randCheck uint8
}

const DEBUG = false

// 生成随机数
func randInt(min, max int, seed ...rand.Source) int {
	if min == max {
		return min
	}
	if max < min {
		min, max = max, min
	}

	if len(seed) > 0 {
		return rand.New(seed[0]).Intn(max-min) + min
	}
	return rand.Intn(max-min) + min
}

// 数据结构  [数据：16 ~ 56位][随机数：5位][高位补码标记：1位][奇偶校验位：1位][奇偶校验位：1位]
func normalization(userKey uint64, srcId uint64, isDecode bool) (key uint64, rand randType, id uint64) {
	if isDecode { // 解码的时候，去掉最后一个字节
		rand = randType{
			rand:      uint8(srcId&0xFF) >> 3,  // 去掉后3位
			padding:   uint8(srcId&0b100) >> 2, // 高位补码标记
			keyCheck:  uint8(srcId&0b10) >> 1,  // 最后第二位是加密码校验码
			randCheck: uint8(srcId & 0b1),      // 最后第一位是随机盐校验码
		}
		id = srcId >> 8 // 取出原始ID，剔除管理字节
	} else {
		rand = randType{rand: uint8(randInt(0, (1<<5)-1)), padding: 0} // 随机数5位 2^5
		id = srcId
	}

	// 对齐 Keys 和 Slat
	if id <= math.MaxUint16 {
		key = userKey & math.MaxUint16
		rand.slat = (uint64(rand.rand) << 8) ^ key
	} else if id <= math.MaxUint32 {
		key = userKey & math.MaxUint32
		rand.slat = (uint64(rand.rand) << 16) ^ key
		if rand.padding == 1 && isDecode { // 原本32位整数，但是混淆后小于 16 位，那么会在 17 位置 1 以便正确解析大小
			id = id & 0xFFFF // 补码， 32位 => 0xFFFF
		}
	} else {
		// max := uint64((1 << 56) - 1)
		key = userKey & uint64((1<<56)-1)
		rand.slat = (uint64(rand.rand) << 32) ^ key
		if rand.padding == 1 && isDecode { // 原本64位整数，但是混淆后小于 32 位，那么会在 33 位置 1 以便正确解析大小
			id = id & 0xFFFF_FFFF // 64位 => 0x1FFFFFFFF
		}
	}
	if DEBUG {
		fmt.Println("随机数", rand.rand, "盐", rand.slat, "用户Key", userKey, "Key", key)
	}
	// rand.slat = slat

	return
}

// 编码一个整数
// mixId = 低位rand + (key ^ id ^ (key/低位rand))
func Encode(key uint64, id uint64, encoder ...Encoder) (rs string, err error) {
	var e Encoder
	var middleId uint64
	if len(encoder) == 0 {
		e = &BaseEncoder{}
	} else {
		e = encoder[0]
	}
	if middleId, err = Mix(key, id); err != nil {
		return
	}
	return e.Encode(middleId)
}

// 解密一个字符串到整数
// id = (mixId - 低位rand) ^ 低位rand ^ key
func Decode(key uint64, s string, encoder ...Encoder) (id uint64, err error) {
	var e Encoder
	var middleId uint64
	if len(encoder) == 0 {
		e = &BaseEncoder{}
	} else {
		e = encoder[0]
	}
	if middleId, err = e.Decode(s); err != nil {
		return
	}
	id, err = Unmix(key, middleId)
	return
}

// 奇偶校验，奇数然后1，偶数返回0
func ParityCheck[T int | uint16 | uint32 | uint64](id T) uint8 {
	len, val := uint8(0), uint64(0)
	switch v := (any)(id).(type) {
	case uint16:
		len = 16
		val = uint64(v)
	case uint32:
		len = 32
		val = uint64(v)
	case uint64:
		len = 64
		val = v
	case int:
		len = 32
		val = uint64(v)
	}
	var result uint8
	for i := uint8(0); i < len; i++ {
		if (val & (1 << i)) != 0 {
			result ^= 1
		}
	}
	return result
}

// 混淆ID
func Mix(key, id uint64) (out uint64, err error) {
	if id > (1<<56)-1 {
		err = fmt.Errorf("数字[%d]已超出最大可混淆数字[%d]", id, 1<<56-1)
		return
	}
	k, randObj, _ := normalization(key, id, false)

	if DEBUG {
		fmt.Printf("编码源信息：用户Key[%d]，规整Key[%d]，随机数[%d]，原始ID[%d]\n", key, k, randObj.rand, id)
	}

	// 第一次密码混淆
	newId := k ^ id
	keySign := ParityCheck(newId)
	if DEBUG {
		fmt.Printf("加密码混淆：[%d => %064b]，校验位[%d]\n", newId, newId, keySign)
	}

	// 第二次随机数混淆
	newId = newId ^ randObj.slat
	randSign := ParityCheck(newId)
	if DEBUG {
		fmt.Printf("随机盐混淆：[%d => %064b]，盐[%d]，校验位[%d]\n", newId, newId, randObj.slat, randSign)
	}

	// 高位补码
	paddingSign := uint8(0)
	if id > math.MaxUint16 && id <= math.MaxUint32 && newId <= 0xFFFF { // 大于 16位 小于或等于 32位，但异或结果小于等于 16位 才需要补码
		newId |= 0x1_0000 // 第17位设置为1
		paddingSign = 1
	} else if id > math.MaxUint32 && newId <= 0xFFFF_FFFF { // 大于 32位 ，但异或结果小于等于 32位
		newId |= 0x1_0000_0000 // 第33为设置为1
		paddingSign = 1
	}
	managerBit := (randObj.rand << 3) | (paddingSign << 2) | (keySign << 1) | randSign
	if DEBUG {
		fmt.Printf("管理字节位：[%d => %08b]，随机数[%d],高位补码位[%d],加密码校验位[%d],随机盐校验位[%d]\n", managerBit, managerBit, randObj.rand, paddingSign, keySign, randSign)
	}

	// 将随机数添加到末尾
	out = (newId << 8) + uint64(managerBit)
	if DEBUG {
		fmt.Printf("最终生成数：[%d => %064b]\n", out, out)
	}
	return
}

// 清除混淆
func Unmix(key, id uint64) (out uint64, err error) {
	if DEBUG {
		fmt.Printf("需要解码数：[%d => %064b]\n", id, id)
	}
	k, randObj, id := normalization(key, id, true)

	// 校验随机数奇偶位
	if DEBUG {
		fmt.Printf("解码源信息：用户Key[%d]，规整Key[%d]，随机数[%d]，原始ID[%d]\n", key, k, randObj.rand, id)
		fmt.Printf("管理字信息：随机数[%d],高位补码位[%d],加密码校验位[%d],随机盐校验位[%d]\n", randObj.rand, randObj.padding, randObj.keyCheck, randObj.randCheck)
		fmt.Printf("随机盐校验：[%d => %064b]，盐[%d]，校验位[%d]\n", id, id, randObj.slat, randObj.randCheck)
	}
	if randObj.randCheck != ParityCheck(id) {
		err = errors.New("校验失败[randsalt]")
		return
	}

	out = id ^ randObj.slat
	if DEBUG {
		fmt.Printf("加密码校验：[%d => %064b]，校验位[%d]\n", out, out, randObj.randCheck)
	}
	if randObj.keyCheck != ParityCheck(out) {
		err = errors.New("校验失败[key]")
		return
	}
	out = k ^ out
	return
}
