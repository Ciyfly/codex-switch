package display

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/codex-switch/codex-switch/internal/api"
	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/utils"
)

// PrintUsageReport 输出额度检查结果
func PrintUsageReport(out io.Writer, key config.APIKey, usage *api.UsageResult) {
	fmt.Fprintln(out, ColorPrimary.Sprint("╔═══════════════════════════════════════════════════════════════╗"))
	fmt.Fprintln(out, ColorPrimary.Sprint("║                       额度检查报告                             ║"))
	fmt.Fprintln(out, ColorPrimary.Sprint("╚═══════════════════════════════════════════════════════════════╝"))

	fmt.Fprintf(out, "\n  当前 Key: %s (%s)\n", key.Name, key.ID)
	fmt.Fprintf(out, "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Fprintf(out, "  ✓ API 类型: %s\n", key.Type)

	used := key.QuotaUsed
	limit := key.QuotaLimit
	daily := 0.0
	weekly := 0.0
	monthly := key.QuotaUsed

	if usage != nil {
		used = usage.Used
		if usage.Limit > 0 {
			limit = usage.Limit
		}
		daily = usage.DailyTotal
		weekly = usage.WeeklyTotal
		if usage.Monthly > 0 {
			monthly = usage.Monthly
		} else {
			monthly = usage.Used
		}
	}

	fmt.Fprintf(out, "  额度状态:\n")
	if limit > 0 {
		fmt.Fprintf(out, "    本期限额:   %s\n", utils.FormatCurrency(limit))
		fmt.Fprintf(out, "    已使用:     %s\n", utils.FormatCurrency(used))
		remaining := limit - used
		if remaining < 0 {
			remaining = 0
		}
		fmt.Fprintf(out, "    剩余:       %s\n", utils.FormatCurrency(remaining))
		fmt.Fprintf(out, "\n    进度: %s %.1f%%\n", RenderProgressBar(used, limit, 25), used/limit*100)
	} else {
		fmt.Fprintf(out, "    已使用:     %s\n", utils.FormatCurrency(used))
	}

	fmt.Fprintf(out, "\n  使用趋势:\n")
	fmt.Fprintf(out, "    今日:       %s\n", utils.FormatCurrency(daily))
	fmt.Fprintf(out, "    本周:       %s\n", utils.FormatCurrency(weekly))
	fmt.Fprintf(out, "    本月:       %s\n", utils.FormatCurrency(monthly))

	fmt.Fprintf(out, "\n  更新时间:    %s\n", time.Now().Format(time.RFC3339))

	if advice := buildAdvice(limit, used, daily); advice != "" {
		fmt.Fprintf(out, "\n  建议: %s\n", advice)
	}
}

func buildAdvice(limit, used, daily float64) string {
	if limit <= 0 {
		return "当前配置为不限额，请关注实际账单"
	}
	remaining := limit - used
	if remaining <= 0 {
		return ColorError.Sprint("额度已耗尽，建议立即切换到备用 Key")
	}
	if daily <= 0 {
		return "当前使用正常，保持关注即可"
	}
	days := remaining / daily
	if days < 1 {
		return ColorWarning.Sprint("预计 24 小时内耗尽额度，建议准备备用 Key")
	}
	if days < 7 {
		return ColorWarning.Sprintf("预计 %.0f 天后额度用尽，建议提前规划", math.Floor(days))
	}
	return "额度充足，无需额外操作"
}
