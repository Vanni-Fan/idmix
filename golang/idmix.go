package idmix

import (
	"errors"
	"math"
	"math/rand"
)

type randType struct {
	rand      uint8
	keyCheck  uint8
	randCheck uint8
}

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

// 数据结构  [数据：16 ~ 56位][随机数：6位][奇偶校验位：1位][奇偶校验位：1位]
func normalization(userKey uint64, srcId uint64, isDecode bool) (key uint64, rand randType, id uint64, err error) {
	if isDecode { // 解码的时候，去掉最后一个字节
		rand = randType{
			rand:      uint8(srcId&0xFF) >> 2, // 去掉后3位
			randCheck: uint8(srcId & 0b1),     // 最后第一位是Random校验码
			keyCheck:  uint8(srcId&0b10) >> 1, // 最后第二位是Key校验码
		}
		id = srcId >> 8 // 取出原始ID
	} else {
		rand = randType{rand: uint8(randInt(0, (1<<6)-1))} // 随机数5位 2^5
		id = srcId
	}

	if id <= math.MaxUint16 {
		key = userKey & math.MaxUint16
	} else if id <= math.MaxUint32 {
		key = userKey & math.MaxUint32
	} else {
		key = userKey & ((1 << 56) - 1) // 如果大于32位，那么编码后的字符串>7
	}

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
	k, randObj, _, err := normalization(key, id, false)
	if err != nil {
		return
	}
	// fmt.Printf("编码：Key[%d]，规整Key[%d]，随机数[%d]，原始ID[%d]\n", key, k, randObj.rand, id)

	// 第一次密码混淆
	first := k ^ id
	firstCheck := ParityCheck(first)
	finalRand := (randObj.rand << 1) + firstCheck // 校验码
	// fmt.Printf("第一次密码混淆：[%d => %064b]，校验位：[%d]\n", first, first, firstCheck)

	// 第二次随机数混淆
	randSalt := k / uint64(randObj.rand+1) % key
	second := first ^ randSalt
	secondCheck := ParityCheck(second)
	finalRand = (finalRand << 1) + secondCheck
	// fmt.Printf("第二次随机混淆：[%d => %064b]，盐：[%d]，校验位：[%d]\n", second, second, randSalt, secondCheck)
	// fmt.Printf("随机数[%d => %08b],校验位[%d,%d]，最终随机数：[%d => %08b]\n", randObj.rand, randObj.rand, firstCheck, secondCheck, finalRand, finalRand)

	// 将随机数添加到末尾
	out = (second << 8) + uint64(finalRand)
	// fmt.Printf("最终生成的数[%s %d => %064b]\n", rs, final, final)
	return
}

// 清除混淆
func Unmix(key, id uint64) (out uint64, err error) {
	k, randObj, id, err := normalization(key, id, true)
	if err != nil {
		return
	}

	// 校验随机数奇偶位
	firstCheck := randObj.keyCheck
	secondCheck := randObj.randCheck
	// fmt.Printf("解码：Key[%d]，规整Key[%d]，随机数[%d => %06b.%d.%d]，解码ID[%d => %064b]\n", key, k, randObj.rand, randObj.rand, firstCheck, secondCheck, id, id)

	// fmt.Printf("第一层随机校验：[%d => %064b]，校验位：[%d]\n", id, id, secondCheck)
	if secondCheck != ParityCheck(id) {
		err = errors.New("第一层校验失败")
		return
	}

	randSalt := k / uint64(randObj.rand+1) % key
	middle := id ^ randSalt
	// fmt.Printf("第二层随机校验：[%d => %064b]，盐：[%d]，校验位：[%d]\n", middle, middle, randSalt, firstCheck)
	if firstCheck != ParityCheck(middle) {
		err = errors.New("第二层校验失败")
		return
	}
	out = k ^ middle
	return
}
