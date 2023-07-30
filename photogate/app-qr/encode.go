package appqr

import "errors"

const b62chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const b62len = uint64(len(b62chars))

var ErrInvalidBase62String = errors.New("invalid base62 string")

var b62reverseMap = map[byte]uint64{}

func init() {
	for i, c := range b62chars {
		b62reverseMap[byte(c)] = uint64(i)
	}
}

func chunkEncode(n uint64) string {
	var b [11]byte
	for i := 10; i >= 0; i-- {
		b[i] = b62chars[n%b62len]
		n /= b62len
		if n == 0 {
			return string(b[i:])
		}
	}
	return ""
}

func chunkDecode(s string) (uint64, error) {
	n := uint64(0)
	for i := range s {
		c := s[i]
		if x, ok := b62reverseMap[c]; ok {
			n = n*62 + x
		} else {
			return 0, ErrInvalidBase62String
		}
	}
	return n, nil
}
