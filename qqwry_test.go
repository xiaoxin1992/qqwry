package qqwry

import (
	"log"
	"testing"
)

func TestNewQQWry(t *testing.T) {
	q, err := NewQQWry("./qqwry.dat")
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
