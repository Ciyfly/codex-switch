package display

import (
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
