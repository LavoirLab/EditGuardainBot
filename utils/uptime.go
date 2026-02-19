package utils

import (
	"fmt"
	"strings"
	"time"
)

func FormatUptime(d time.Duration) string {
	seconds := int(d.Seconds()) % 60
	minutes := int(d.Minutes()) % 60
	hours := int(d.Hours()) % 24
	days := int(d.Hours()) / 24

	var result string
	if days > 0 {
		result += fmt.Sprintf("%dd:", days)
	}

	if hours > 0 {
		result += fmt.Sprintf("%dh:", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%dm:", minutes)
	}
	if contains := strings.Contains(result, "d:"); !contains {
		result += fmt.Sprintf("%ds", seconds)
	}

	return result
}
