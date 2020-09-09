package qqwry

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io/ioutil"
	"net"
	"os"
	"strconv"
)

const (
	indexLen        = 7
	redirectModeOne = 0x01
	redirectModeTwo = 0x02
)

func Int64ToString(ipInt int64) string {
	b0 := strconv.FormatInt((ipInt>>24)&0xff, 10)
	b1 := strconv.FormatInt((ipInt>>16)&0xff, 10)
	b2 := strconv.FormatInt((ipInt>>8)&0xff, 10)
	b3 := strconv.FormatInt(ipInt&0xff, 10)
	return fmt.Sprintf("%s.%s.%s.%s", b0, b1, b2, b3)
}

type Address struct {
	AddressFirst int64  `json:"address_first"`
	AddressLast  int64  `json:"address_last"`
	Country      string `json:"country"`
	Area         string `json:"area"`
}

func (ip *Address) Int64ToString() (addressFirst, addressLast string) {
	addressFirst = Int64ToString(ip.AddressFirst)
	addressLast = Int64ToString(ip.AddressLast)
	return
}

type QQWry struct {
	Address    []*Address `json:"address"`
	Total      int64      `json:"total"`
	binData    []byte
	offset     int64
	startIndex uint32
	endIndex   uint32
	dataSize   int64
}

func NewQQWry(filename string) (q *QQWry, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	startIndex := binary.LittleEndian.Uint32(data[0:4])
	endIndex := binary.LittleEndian.Uint32(data[4:8])
	return &QQWry{
		binData:    data,
		startIndex: startIndex,
		endIndex:   endIndex,
		Total:      int64((endIndex-startIndex)/indexLen + 1),
		dataSize:   int64(len(data)),
	}, nil
}

func (q *QQWry) IpToUint32(ip string) (long uint32) {
	_ = binary.Read(bytes.NewBuffer(net.ParseIP(ip).To4()), binary.BigEndian, &long)
	return
}

func (q *QQWry) bytesToUint32(data []byte) (b uint32) {
	b = uint32(data[0]) & 0xff
	b |= (uint32(data[1]) << 8) & 0xff00
	b |= (uint32(data[2]) << 16) & 0xff0000
	return
}

func (q *QQWry) setOffset(offset int64) {
	q.offset = offset
}

func (q *QQWry) readBin(byteLen int64, offset int64) (rs []byte) {
	if offset != -1 {
		q.setOffset(offset)
	}
	endLen := q.offset + byteLen
	if q.offset > q.dataSize {
		return nil
	}
	if endLen > q.dataSize {
		endLen = q.dataSize
	}
	rs = q.binData[q.offset:endLen]
	q.setOffset(endLen)
	return
}

func (q *QQWry) getMod(offset uint32) byte {
	mode := q.readBin(1, int64(offset))
	return mode[0]
}

func (q *QQWry) readByte3() uint32 {
	buf := q.readBin(3, -1)
	return q.bytesToUint32(buf)
}

func (q *QQWry) formatString(offset uint32) []byte {
	q.setOffset(int64(offset))
	data := make([]byte, 0, 30)
	for {
		buf := q.readBin(1, -1)
		if buf[0] == 0 {
			break
		}
		data = append(data, buf[0])
	}
	return data
}

func (q *QQWry) formatArea(offset uint32) []byte {
	mode := q.getMod(offset)
	if mode == redirectModeOne || mode == redirectModeTwo {
		areaOffset := q.readByte3()
		if areaOffset != 0 {
			return q.formatString(areaOffset)
		}
	} else {
		return q.formatString(offset)
	}
	return []byte("")
}

func (q *QQWry) ReadPositionInfo(mode byte, offset uint32) (string, string) {
	// 读取地址位置信息，以及结束IP
	var country []byte
	var area []byte
	switch mode {
	case redirectModeOne:
		countryOffset := q.readByte3()
		mode = q.getMod(countryOffset)
		if mode == redirectModeTwo {
			countryOffsetModTwo := q.readByte3()
			country = q.formatString(countryOffsetModTwo)
			countryOffset += 4
		} else {
			country = q.formatString(countryOffset)
			countryOffset += uint32(len(country) + 1)
		}
		area = q.formatArea(countryOffset)
	case redirectModeTwo:
		countryOffset := q.readByte3()
		country = q.formatString(countryOffset)
		area = q.formatArea(offset + 8)
	default:
		country = q.formatString(offset + 4)
		area = q.formatArea(offset + uint32(5+len(country)))
	}
	enc := simplifiedchinese.GBK.NewDecoder()
	s1, _ := enc.String(string(country))
	s2, _ := enc.String(string(area))
	return s1, s2

}

func (q *QQWry) Match(ip string) map[string]string {
	// 二分查找IP地理位置
	parseIP := net.ParseIP(ip)
	if parseIP == nil {
		return map[string]string{}
	}
	ipInt := q.IpToUint32(parseIP.String())
	startIndex := int64(q.startIndex)
	endIndex := int64(q.endIndex)
	for {
		offset := startIndex + ((endIndex-startIndex)/indexLen>>1)*indexLen // 计算偏移
		buf := q.readBin(indexLen, offset)
		startIP := binary.LittleEndian.Uint32(buf[:4]) // 获取开始IP地址
		if endIndex-startIndex == indexLen {
			offsetData := q.bytesToUint32(buf[4:])
			buf = q.readBin(indexLen, -1)
			if ipInt < binary.LittleEndian.Uint32(buf[:4]) {
				country, area := q.ReadPositionInfo(q.getMod(offsetData+4), offsetData)
				return map[string]string{
					"ip":      parseIP.String(),
					"country": country,
					"area":    area,
				}
			} else {
				return map[string]string{}
			}
		}
		if startIP > ipInt {
			endIndex = offset
		} else if startIP < ipInt {
			startIndex = offset
		} else if startIP == ipInt {
			offsetData := q.bytesToUint32(buf[4:])
			country, area := q.ReadPositionInfo(q.getMod(offsetData+4), offsetData)
			return map[string]string{
				"ip":      parseIP.String(),
				"country": country,
				"area":    area,
			}
		}

	}
}

func (q *QQWry) ConvertMap() *QQWry {
	for startIndex := q.startIndex; startIndex <= q.endIndex; startIndex += indexLen {
		buf := q.readBin(indexLen, int64(startIndex))
		startIP := binary.LittleEndian.Uint32(buf[:4]) // 获取开始IP地址
		offset := q.bytesToUint32(buf[4:])
		endIP := binary.LittleEndian.Uint32(q.readBin(4, int64(offset)))
		country, area := q.ReadPositionInfo(q.getMod(offset+4), offset)
		q.Address = append(q.Address, &Address{
			Country:      country,
			Area:         area,
			AddressFirst: int64(startIP),
			AddressLast:  int64(endIP),
		})
	}
	return q
}

func (q *QQWry) DumpToJson(filename string) error {
	// 数据格式化并写入文件
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	jsonEncoder := json.NewEncoder(file)
	err = jsonEncoder.Encode(q)
	if err != nil {
		return err
	}
	return nil
}
