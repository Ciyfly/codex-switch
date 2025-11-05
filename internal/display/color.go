package display

import (
	"math"
	"strings"

	"github.com/fatih/color"
)

// 定义终端颜色主题，统一控制输出风格
var (
	ColorActive   = color.New(color.FgGreen, color.Bold)
	ColorInactive = color.New(color.FgWhite)
	ColorWarning  = color.New(color.FgYellow, color.Bold)
	ColorError    = color.New(color.FgRed, color.Bold)
	ColorSuccess  = color.New(color.FgGreen)
	ColorInfo     = color.New(color.FgCyan)
	ColorPrimary  = color.New(color.FgBlue, color.Bold)
)

// QuotaColor 根据使用百分比返回不同颜色
func QuotaColor(percentage float64) *color.Color {
	switch {
	case percentage < 50:
		return color.New(color.FgGreen)
	case percentage < 80:
		return color.New(color.FgYellow)
	default:
		return color.New(color.FgRed)
	}
}

// RenderProgressBar 按照使用比例绘制彩色进度条
func RenderProgressBar(used, limit float64, width int) string {
	if width <= 0 {
		width = 20
	}
	if limit <= 0 {
		bar := strings.Repeat("━", width)
		return ColorInfo.Sprint(bar)
	}

	percentage := used / limit * 100
	if percentage < 0 {
		percentage = 0
	}
	filledWidth := int(math.Round((used / limit) * float64(width)))
	if filledWidth > width {
		filledWidth = width
	}
	if filledWidth < 0 {
		filledWidth = 0
	}

	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", width-filledWidth)

	return QuotaColor(percentage).Sprintf("%s%s", filled, empty)
}

// MaskAPIKey 对 API Key 做脱敏处理
func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 6 {
		return "****"
	}
	prefix := key[:4]
	suffix := key[len(key)-3:]
	return prefix + strings.Repeat("*", len(key)-7) + suffix
}
