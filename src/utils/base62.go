package utils

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
)

var b62_dict = []byte("0emqTk6bL1RNGStXV8ECzyKxMignYWrJ47ZsPoBpU2udOHwI3fDva9F5clQjhA")

const b62_dict_len = 62

func sign(num int64) uint32 {
	numS := fmt.Sprintf("%d", num)
	hash := crc32.ChecksumIEEE([]byte(numS))
	var sum uint32 = 0
	var mask uint32 = 0xff
	for i := 0; i < 4; i++ {
		sum ^= ^((hash >> uint(i*8)) & mask)
	}
	return sum
}

func base62Encode(num int64) []byte {
	div := num & 0xffffffff
	str := ""
	for div >= b62_dict_len {
		pos := div % b62_dict_len
		div = (div - pos) / b62_dict_len
		str = fmt.Sprintf("%c%s", b62_dict[pos], str)
	}
	return []byte(fmt.Sprintf("%c%s", b62_dict[div], str))
}

func base62Decode(raw []byte) (int64, error) {
	var num int64 = 0
	for i := 0; i < len(raw); i++ {
		num *= b62_dict_len
		pos := bytes.IndexByte(b62_dict, raw[i])
		if pos == -1 {
			return 0, errors.New(fmt.Sprintf("invalid char:%c", raw[i]))
		}
		num += int64(pos)
	}
	return num, nil
}

func Base62Encode(num int64) []byte {
	low32 := num & 0xffffffff
	hieght32 := num >> 32 & 0xffffffff
	signRaw := sign(num)
	b62Low := base62Encode(low32)
	b62hieght := []byte{}
	if hieght32 != 0 {
		b62hieght = base62Encode(hieght32)
	}
	b62LowLen := len(b62Low) & 0x7
	headerRaw := uint32(b62LowLen<<8) | signRaw
	header := base62Encode(int64(headerRaw))
	return bytes.Join([][]byte{b62hieght, b62Low, header}, []byte{})
}

func Base62Decode(raw []byte) (int64, error) {
	rawLen := len(raw)
	if rawLen <= 2 {
		return 0, errors.New(fmt.Sprintf("raw is too short:%s", raw))
	}
	header := raw[rawLen-2:]
	headerRaw, err := base62Decode(header)
	if err != nil {
		return 0, err
	}
	signRaw := headerRaw & 0xff
	b62LowLen := (headerRaw >> 8) & 0x7
	if rawLen < int(b62LowLen+2) {
		return 0, errors.New(fmt.Sprintf("invalid raw:%s", raw))
	}
	headerStart := int(rawLen - 2)
	b62LowStart := headerStart - int(b62LowLen)
	b62Low := raw[b62LowStart:headerStart]
	b62Hieght := raw[0:b62LowStart]
	low32, err := base62Decode(b62Low)
	if err != nil {
		return 0, err
	}
	var hieght32 int64 = 0
	if len(b62Hieght) > 0 {
		hieght32, err = base62Decode(b62Hieght)
		if err != nil {
			return 0, err
		}
	}
	num := (hieght32 << 32) | low32
	if sign(num) == uint32(signRaw) {
		return num, nil
	}
	return 0, errors.New(fmt.Sprintf("invalid raw:%s", raw))
}
