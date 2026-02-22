package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func LoadTyped[T any](cache interface {
	Get(string) (any, bool)
}, key string,
) (T, bool) {
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

func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration")
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, err
	}

	switch unit {
	case 'm':
		return time.Duration(num) * time.Minute, nil
	case 'h':
		return time.Duration(num) * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid unit (use m or h)")
	}
}

func ParseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "enable", "enabled", "true", "1", "yes", "y":
		return true, nil
	case "off", "disable", "disabled", "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean string: %q", s)
	}
}

func FormatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
