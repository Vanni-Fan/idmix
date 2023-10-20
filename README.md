# idmix 用途
- 用于将数字 id 编码成无规律的字符串
- 最大可编码 id 大小为 2^56-1 （因为预留了1个字节做管理）
- 相同的数字可编码成最多 32 种字符串
  - 同一个 id 会有最低 32 种可能，避免被猜测和枚举测试
- 防止数字 id 被猜测到
  - 比如数据库的自增ID，则原来 id 上加减数字便可获得有效 ID
  - idmix 生成的字符串是随机无规律的
- 防止不必要的数据库查询
  - 传统的 id，只要是数字，在前端则会判断合法，然后需要在后台查询数据库，才知道是否合法
  - idmix 有字符串组成，并且无规律，恶意用户随机生成的字符串，有 75% 的记录 idmix 就能判断为非法，降低CC对数据库的影响
- 注意： **原始的ID，和最终编码的字符串，不能同时展示给终端用户，否则Key将暴露（算法所决定的）**
# idmix 算法和存储格式
- 简单使用异或处理，并固定使用1个字节进行校验和随机数存储
- 注意： 因为使用异或算法，所以如果用户指定原始的ID，和最终编码的字符串，那么就能算出原始的Key，因此不要将ID和编码后的信息一同输出
- 生成的ID为8个字节（64位）的整数，其中最末尾的一个字节用来保存随机数和校验信息
- 格式为
  ```
  [数据：16 ~ 56位][管理字节：8位]
  ```
# 自定义ID转字符串的算法
- 默认是将 10进制 的 uint64 转成 36进制（0-9 + 26个字母） 的字符串，你可以定义自己的编码，只需要实现相关的接口即可
- 假如你区分大小写，可以使用62进制（0-9 + 52个字母）
  
# demo
```golang
package main

import (
	"fmt"
	"math/rand"

	idmix "github.com/Vanni-Fan/idmix/golang"
)

func main() {
	key := uint64(1234567890)
	scrId := uint64(rand.Int31())
	str, err := idmix.Encode(key, scrId)
	if err != nil {
		fmt.Println("编码失败", err)
		return
	}
	dstId, err := idmix.Decode(key, str)
	if err != nil {
		fmt.Println("解码失败", err)
		return
	}
	fmt.Printf("编码ID[%d]，字符串[%s]，解码ID[%d]，是否正确[%v]\n", scrId, str, dstId, scrId == dstId)

	// 自定义编码器
	e, err := idmix.NewCustomEncoder("你好+abcdefg_0123456789-ABCDEFG.")
	str, err = idmix.Encode(key, scrId, e)
	if err != nil {
		fmt.Println("编码失败", err)
		return
	}
	dstId, err = idmix.Decode(key, str, e)
	if err != nil {
		fmt.Println("解码失败", err)
		return
	}
	fmt.Printf("编码ID[%d]，字符串[%s]，解码ID[%d]，是否正确[%v]\n", scrId, str, dstId, scrId == dstId)
}
// 可能输出：
//编码ID[1005185537]，字符串[66ihzsdl]，解码ID[1005185537]，是否正确[true]
//编码ID[1005185537]，字符串[A7你Gd好7.]，解码ID[1005185537]，是否正确[true]
```