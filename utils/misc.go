package utils

import (
	"strconv"
	"strings"
)

func LoadTyped[T any](cache interface {
	Get(string) (any, bool)
}, key string) (T, bool) {
	var zero T
	if val, ok := cache.Get(key); ok {
		if typed, ok := val.(T); ok {
			return typed, true
		}
	}
	return zero, false
}

const digitMap = "adefjtghkz" // 0=a, 1=d, 2=e, 3=f, 4=j, 5=t, 6=g, 7=h, 8=k, 9=z
func EncodeDigits(num int64) string {
	s := strconv.FormatInt(num, 10)
	var sb strings.Builder
	for _, ch := range s {
		d := ch - '0'
		if d < 0 || d > 9 {
			continue
		}
		sb.WriteByte(digitMap[d])
	}
	return sb.String()
}
