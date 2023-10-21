package idmix

import (
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

//go test --count=1 -v -timeout 30s

// 测试基本功能，随机生成数字，测试是否能正确编码和解码
// go test --count=1 -v -timeout 30s -run ^TestIntMix$
func TestIntEncode(t *testing.T) {
	var k = uint64(rand.NewSource(time.Now().UnixMicro()).Int63())
	t.Log("密钥：", k)
	var ids = "1"
	for level := 0; level < 32; level++ {
		ids = strconv.FormatInt(int64(randInt(0, 9)), 10) + ids
		id, _ := strconv.ParseUint(ids, 10, 64)
		for i := 0; i < 32; i++ {
			x, e1 := Encode(k, id)
			if e1 != nil {
				t.Logf("无法编码：%d, 错误：%v", id, e1)
				continue
			}
			y, e2 := Decode(k, x)
			if e2 != nil {
				t.Fatalf("无法解码：%s, 错误：%v", x, e2)
			}
			if id != y {
				t.Fatalf("解码错误：原ID[%d]，编码[%s],解码[%d]", id, x, y)
			}
			t.Logf("原ID[%d]，编码[%s]，解码[%d]", id, x, y)
		}
	}
}

func TestIntMix(t *testing.T) {
	var k = uint64(rand.NewSource(time.Now().UnixMicro()).Int63())
	t.Log("密钥：", k)
	var ids = "1"
	for level := 0; level < 32; level++ {
		ids = strconv.FormatInt(int64(randInt(0, 9)), 10) + ids
		id, _ := strconv.ParseUint(ids, 10, 64)
		for i := 0; i < 32; i++ {
			x, e1 := Mix(k, id)
			if e1 != nil {
				t.Logf("无法编码：%d, 错误：%v", id, e1)
				continue
			}
			y, e2 := Unmix(k, x)
			if e2 != nil {
				t.Fatalf("无法解码：%d, 错误：%v", x, e2)
			}
			if id != y {
				t.Fatalf("解码错误：原ID[%d]，编码[%d],解码[%d]", id, x, y)
			}
			t.Logf("原ID[%d]，编码[%d]，解码[%d]", id, x, y)
		}
	}
}

// 测试随机的字符串，能被解码成ID的概率有多少，约为 20%，有效阻止 80% 的攻击
// go test --count=1 -v -timeout 60s -run ^TestIntMixError$
func TestIntMixError(t *testing.T) {
	var k = uint64(rand.NewSource(time.Now().UnixMicro()).Int63())
	t.Log("密钥：", k)
	pass := 0
	r := rand.New(rand.NewSource(0))
	r.Intn(3)
	for i := 0; i < 100000; i++ {
		srcId := uint64(randInt(1000000000, math.MaxUint32, rand.NewSource(time.Now().UnixMicro()+int64(i))))
		_, err := Decode(k, strconv.FormatUint(srcId, 36))
		if err == nil {
			pass++
		}
	}
	t.Logf("随机字符串测试通过率为：[%d%%]", pass/1000)
}

// 性能测试
// go test --count=1 -benchmem -run=^$ -bench ^BenchmarkMix$
func BenchmarkMix(b *testing.B) {
	var k = uint64(rand.NewSource(time.Now().UnixMicro()).Int63())
	var ids = []uint64{123, 123456, 123456789, 123456789012, 123456789012015, 1<<56 - 1}
	for _, id := range ids {
		x, e1 := Encode(k, id)
		if e1 != nil {
			b.Logf("无法编码：%d, 错误：%v", id, e1)
			continue
		}
		y, e2 := Decode(k, x)
		if e2 != nil {
			b.Fatal("无法解码：", x, e2)
		}
		if id != y {
			b.Fatalf("解码错误：原ID[%d]，编码[%s],解码[%d]", id, x, y)
		}
	}
}

// 自定义编码器的性能测试
// go test --count=1 -benchmem -run=^$ -bench ^BenchmarkMixCustom$
func BenchmarkMixCustom(b *testing.B) {
	encoder, _ := NewCustomEncoder("abcdefghijklnmopqrstuvwxyz0123456789ABCDEFGHIJKLNMOPQRSTUVWXYZ-,.+=!@#$%^&*()_[]<>~自定义的中文加数字")
	var k = uint64(rand.NewSource(time.Now().UnixMicro()).Int63())
	var ids = []uint64{123, 123456, 123456789, 123456789012, 123456789012015, 1<<56 - 1}
	for _, id := range ids {
		x, e1 := Encode(k, id, encoder)
		if e1 != nil {
			b.Logf("无法编码：%d, 错误：%v", id, e1)
			continue
		}
		y, e2 := Decode(k, x, encoder)
		if e2 != nil {
			b.Fatal("无法解码：", x, e2)
		}
		if id != y {
			b.Fatalf("解码错误：原ID[%d]，编码[%s],解码[%d]", id, x, y)
		}
	}
}

func TestCustom(t *testing.T) {
	testCustomEncoder(t, "abcdefghijklnmopqrstuvwxyz0123456789ABCDEFGHIJKLNMOPQRSTUVWXYZ") // 64进制编码
	testCustomEncoder(t, "3456789abcdefghkmnpqrstwxy")                                     // 车牌规则，去掉 1ijl  2z  0o  vu
	testCustomEncoder(t, "abcdefghijklnmopqrstuvwxyz0123456789-_")                         // 域名规则，允许下划线和中划线
	testCustomEncoder(t, "自定义的中文加数字0123456789和abcdefghijklmnopqrstuvwxyz")                 // 文件路径规则，允许.逗号
	testCustomEncoder(t, "KLNMOPQRSTUVWXYZ-,.+=!@#$%^&*()_[]<>~自定义的中文加数字abcdefghijklnmopqrstuvwxyz0123456789ABCDEFGHIJ")
}

// 测试自定义编码器
func testCustomEncoder(t *testing.T, str string) {
	e, err := NewCustomEncoder(str)
	if err != nil {
		t.Fatalf("无法创建编码器：%v", err)
	}
	// var k = uint64(rand.NewSource(time.Now().UnixMicro()).Int63()) // 64位随机密码
	var k = uint64(randInt(10000, 999999999)) // 32 位随机密码
	t.Log("密钥：", k)
	var ids = "1"
	for level := 0; level < 17; level++ {
		ids = strconv.FormatInt(int64(randInt(0, 9)), 10) + ids
		id, _ := strconv.ParseUint(ids, 10, 64)
		t.Log("测试ID:", id)
		for i := 0; i < 32; i++ {
			x, e1 := Encode(k, id, e)
			if e1 != nil {
				t.Logf("无法编码：%d, 错误：%v", id, e1)
				continue
			}
			y, e2 := Decode(k, x, e)
			if e2 != nil {
				t.Fatalf("无法解码：%s, 错误：%v", x, e2)
			}
			if id != y {
				t.Fatalf("解码错误：原ID[%d]，编码[%s],解码[%d]", id, x, y)
			}
			t.Logf("原ID[%d]，编码[%s]，解码[%d]", id, x, y)
		}
	}
}
