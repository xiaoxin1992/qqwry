# qqwry
用于解析qqwry格式文件，支持二分法查找，支持数据转换成json格式并保存到文件

安装
```shell script
go get github.com/xiaoxin1992/qqwry@v1.0.4
```

使用示例
```go
package main

import (
	"github.com/xiaoxin1992/qqwry/qqwry"
	"log"
)

func main() {
	q, err := qqwry.NewQQWry("qqwry.dat")
	if err != nil {
		log.Println(err)
		return
	}
	// 查询IP地理位置，返回一个map[string]string结构
	result := q.Match("192.168.1.1")
	log.Println(result)
	// 数据转换成map格式，转换成JSON格式，写入到文件
	err = q.ConvertMap().DumpToJson("test.json")
	if err != nil {
		log.Println(result)
	}
}
```