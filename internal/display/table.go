package display

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/utils"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

// PrintKeyTable 以表格形式输出 Key 列表
func PrintKeyTable(out io.Writer, keys []config.APIKey) {
	writer := prettytable.NewWriter()
	writer.SetOutputMirror(out)

	width := 120
	if f, ok := out.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
			width = w
		}
	}
	writer.SetAllowedRowLength(width)

	writer.Style().Options = prettytable.Options{
		DrawBorder:      true,
		SeparateHeader:  true,
		SeparateColumns: true,
		SeparateRows:    false,
	}
	writer.Style().Box = prettytable.StyleBoxLight

	writer.AppendHeader(prettytable.Row{
		ColorPrimary.Sprint("状态"),
		ColorPrimary.Sprint("名称"),
		ColorPrimary.Sprint("ID"),
		ColorPrimary.Sprint("类型"),
		ColorPrimary.Sprint("额度类型"),
		ColorPrimary.Sprint("使用情况"),
		ColorPrimary.Sprint("最后使用"),
	})

	writer.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 1, Align: text.AlignCenter, WidthMax: 6},
		{Number: 2, Align: text.AlignLeft, WidthMin: 12},
		{Number: 3, Align: text.AlignCenter, WidthMax: 6},
		{Number: 4, Align: text.AlignCenter, WidthMax: 10},
		{Number: 5, Align: text.AlignCenter, WidthMax: 14},
		{Number: 6, Align: text.AlignLeft, WidthMin: 22},
		{Number: 7, Align: text.AlignLeft, WidthMin: 16},
	})

	for _, key := range keys {
		status := "○"
		if key.Active {
			status = ColorActive.Sprint("● 激活")
		}

		writer.AppendRow(prettytable.Row{
			status,
			key.Name,
			key.ID,
			strings.ToUpper(key.Type),
			translateQuotaType(key.QuotaType),
			buildQuotaInfo(key),
			utils.FormatRelativeTime(key.LastUsed),
		})
	}

	writer.Render()
}

// PrintKeyDetail 输出指定 Key 的详细信息
func PrintKeyDetail(out io.Writer, key config.APIKey) {
	fmt.Fprintln(out, ColorPrimary.Sprint("╔═══════════════════════════════════════════════════════════════════╗"))
	fmt.Fprintln(out, ColorPrimary.Sprint("║                        Key 详细信息                                ║"))
	fmt.Fprintln(out, ColorPrimary.Sprint("╚═══════════════════════════════════════════════════════════════════╝"))

	fmt.Fprintf(out, "\n  基本信息\n  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(out, "  ID:            %s\n", key.ID)
	fmt.Fprintf(out, "  名称:          %s\n", key.Name)
	fmt.Fprintf(out, "  类型:          %s\n", strings.ToUpper(key.Type))
	state := "未激活"
	if key.Active {
		state = "✓ 激活中"
	}
	fmt.Fprintf(out, "  状态:          %s\n", state)

	fmt.Fprintf(out, "\n  API 配置\n  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(out, "  Base URL:      %s\n", key.BaseURL)
	fmt.Fprintf(out, "  API Key:       %s\n", MaskAPIKey(key.APIKey))
	if strings.TrimSpace(key.RawConfig) != "" {
		fmt.Fprintf(out, "  配置片段:      已提供\n")
	}

	fmt.Fprintf(out, "\n  额度信息\n  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(out, "  类型:          %s\n", translateQuotaType(key.QuotaType))
	if key.QuotaLimit > 0 {
		fmt.Fprintf(out, "  限额:          %s\n", utils.FormatCurrency(key.QuotaLimit))
		fmt.Fprintf(out, "  已使用:        %s (%.1f%%)\n", utils.FormatCurrency(key.QuotaUsed), usagePercent(key.QuotaUsed, key.QuotaLimit))
		fmt.Fprintf(out, "  剩余:          %s\n", utils.FormatCurrency(key.QuotaLimit-key.QuotaUsed))
		fmt.Fprintf(out, "\n  进度: %s %.1f%%\n", RenderProgressBar(key.QuotaUsed, key.QuotaLimit, 20), usagePercent(key.QuotaUsed, key.QuotaLimit))
	} else {
		fmt.Fprintf(out, "  限额:          不限\n")
		fmt.Fprintf(out, "  已使用:        %s\n", utils.FormatCurrency(key.QuotaUsed))
	}

	fmt.Fprintf(out, "\n  使用统计\n  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(out, "  创建时间:      %s\n", key.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(out, "  最后检查:      %s\n", key.LastChecked.Format(time.RFC3339))
	fmt.Fprintf(out, "  最后使用:      %s\n", key.LastUsed.Format(time.RFC3339))

	if len(key.Tags) > 0 {
		fmt.Fprintf(out, "\n  标签:          %s\n", strings.Join(key.Tags, ", "))
	}
	if strings.TrimSpace(key.Description) != "" {
		fmt.Fprintf(out, "  备注:          %s\n", key.Description)
	}
}

func buildQuotaInfo(key config.APIKey) string {
	if key.QuotaLimit <= 0 {
		return fmt.Sprintf("%s / ∞", utils.FormatCurrency(key.QuotaUsed))
	}
	percentage := usagePercent(key.QuotaUsed, key.QuotaLimit)
	bar := RenderProgressBar(key.QuotaUsed, key.QuotaLimit, 10)
	return fmt.Sprintf("%s/%s  %s %.1f%%", utils.FormatCurrency(key.QuotaUsed), utils.FormatCurrency(key.QuotaLimit), bar, percentage)
}

func translateQuotaType(q string) string {
	switch strings.ToLower(q) {
	case config.QuotaDaily:
		return "天卡"
	case config.QuotaWeekly:
		return "周卡"
	case config.QuotaMonthly:
		return "月卡"
	case config.QuotaYearly:
		return "年卡"
	case config.QuotaUnlimited:
		return "不限制"
	default:
		return q
	}
}

func usagePercent(used, limit float64) float64 {
	if limit <= 0 {
		return 0
	}
	return used / limit * 100
}
