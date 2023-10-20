package idmix

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
)

// 自定义的加密接口
type Encoder interface {
	Encode(uint64) (string, error) // 将整数转成字符串
	Decode(string) (uint64, error) // 将字符串转成整数
}

// 默认的加解密对象
type BaseEncoder struct{}

// 整数转成 36 进制的字符串
func (BaseEncoder) Encode(i uint64) (string, error) {
	return strconv.FormatUint(i, 36), nil
}

// 36 进制的字符串转成10进制数字
func (BaseEncoder) Decode(s string) (id uint64, err error) {
	id, err = strconv.ParseUint(s, 36, 64)
	if err != nil {
		err = errors.New("包含非法字符，无法解码：" + err.Error())
	}
	return
}

// Cursou
type CustomEncoder struct {
	baseRune []rune
	mapping  map[rune]uint8
}

// 用指定的字符集创建自定义编码
func NewCustomEncoder(baseStr string) (*CustomEncoder, error) {
	if len(baseStr) < 2 {
		return nil, errors.New("进制必须大于2个字符，比如最小的二级制也是0和1两个字符串")
	}
	encode := &CustomEncoder{
		mapping: make(map[rune]uint8),
	}
	for i, s := range bytes.Runes([]byte(baseStr)) {
		if _, ok := encode.mapping[s]; ok {
			return nil, errors.New("进制字符串中不允许有相同的字符：" + string(s))
		}
		encode.baseRune = append(encode.baseRune, s)
		encode.mapping[s] = uint8(i)
	}
	return encode, nil
}

// 将一个数字转换成指定编码
func (c *CustomEncoder) Encode(n uint64) (string, error) {
	var result bytes.Buffer
	b := uint64(len(c.mapping))
	for n > 0 {
		// 计算商和余数
		quotient := n / b
		remainder := n % b

		// 将余数转换成字符，并添加到结果字符串的最前面
		result.WriteString(string(c.baseRune[remainder]))
		n = quotient
	}

	// 如果结果字符串为空，则说明原数字为 0
	if result.Len() == 0 {
		result.WriteString(string(c.baseRune[0]))
	}

	// 将结果字符串反转
	strList := bytes.Runes(result.Bytes())
	for i, j := 0, len(strList)-1; i < j; i, j = i+1, j-1 {
		strList[i], strList[j] = strList[j], strList[i]
	}

	return string(strList), nil
}

// 将自定编码转换10进制
func (c *CustomEncoder) Decode(str string) (id uint64, err error) {
	var result uint64
	b := uint64(len(c.mapping))
	rs := bytes.Runes([]byte(str))
	length := len(rs)
	for i, s := range rs {
		// 查找当前字符在映射表中的值
		value, ok := c.mapping[s]
		if !ok {
			return 0, fmt.Errorf("无效字符: %c", s)
		}

		// 计算当前位的值，并累加到结果中
		position := int64(length - 1 - i)
		// todo ,当转出大于 uint64 时，会截断，需要用 big.Int 来改
		result += uint64(value) * uint64(math.Pow(float64(b), float64(position)))
	}
	return result, nil
}
