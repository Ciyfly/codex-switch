package cmd

import (
	"fmt"
	"strings"

	"github.com/codex-switch/codex-switch/internal/config"
	codex "github.com/codex-switch/codex-switch/internal/integration/codex"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	switchCmd := &cobra.Command{
		Use:   "switch <ID|NAME>",
		Short: "切换当前激活的 API Key",
		Args:  cobra.ExactArgs(1),
		RunE:  runSwitch,
	}

	RootCommand().AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	target := strings.TrimSpace(args[0])

	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	var key config.APIKey
	key, err = manager.GetKey(target)
	if err != nil {
		key, err = manager.GetKeyByName(target)
		if err != nil {
			return fmt.Errorf("未找到 ID 或名称为 %s 的 Key", target)
		}
	}

	if err := manager.SetActiveKey(key.ID); err != nil {
		return err
	}
	if err := manager.TouchKey(key.ID); err != nil {
		return err
	}
	if err := manager.Save(); err != nil {
		return err
	}

	activeKey, err := manager.GetKey(key.ID)
	if err != nil {
		return err
	}

	configurator, err := codex.NewConfigurator("", "")
	if err != nil {
		return fmt.Errorf("初始化 Codex 配置失败: %w", err)
	}
	if err := configurator.Apply(activeKey); err != nil {
		return fmt.Errorf("同步 Codex 配置失败: %w", err)
	}

	success := color.New(color.FgGreen, color.Bold).Sprint("✓")
	nameText := color.New(color.FgCyan, color.Bold).Sprint(key.Name)
	idText := color.New(color.FgHiBlack).Sprint(key.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "%s 已切换到: %s %s\n", success, nameText, idText)
	fmt.Fprintf(cmd.OutOrStdout(), "  类型: %s\n", color.New(color.FgMagenta).Sprint(strings.ToUpper(key.Type)))
	fmt.Fprintf(cmd.OutOrStdout(), "  剩余额度: %s\n", color.New(color.FgYellow).Sprint(formatRemaining(key)))
	fmt.Fprintf(cmd.OutOrStdout(), "  Codex 配置: %s\n", color.New(color.FgGreen).Sprint("已同步"))

	logging.Infof("切换 Key 至 %s (%s)", key.Name, key.ID)

	return nil
}

func formatRemaining(key config.APIKey) string {
	if key.QuotaLimit <= 0 {
		return fmt.Sprintf("%0.2f / ∞", key.QuotaUsed)
	}
	remaining := key.QuotaLimit - key.QuotaUsed
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("%.2f / %.2f", remaining, key.QuotaLimit)
}
