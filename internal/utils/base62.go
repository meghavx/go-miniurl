package utils

import "bytes"

var chars = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func Base62Encode(num uint64) string {
	num = 100_000_000 - num

	res := make([]byte, 0)
	for num > 0 {
		res = append(res, chars[num%62])
		num /= 62
	}
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return string(res)
}

func Base62Decode(s string) uint64 {
	var num uint64
	for i := 0; i < len(s); i++ {
		num = num*62 + uint64(bytes.IndexByte(chars, s[i]))
	}
	return 100_000_000 - num
}
