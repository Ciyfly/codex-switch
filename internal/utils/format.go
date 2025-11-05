package utils

import (
	"fmt"
	"math"
	"time"
)

// FormatCurrency 将金额格式化为美元字符串
func FormatCurrency(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

// FormatRelativeTime 输出相对时间描述
func FormatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "未记录"
	}
	now := time.Now()
	diff := now.Sub(t)
	if diff < 0 {
		diff = -diff
	}

	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d 天前", int(diff.Hours()/24))
	default:
		days := int(math.Round(diff.Hours() / 24))
		return fmt.Sprintf("%d 天前", days)
	}
}
