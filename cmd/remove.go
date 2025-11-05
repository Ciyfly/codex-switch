package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	removeYes bool
)

func init() {
	removeCmd := &cobra.Command{
		Use:   "remove <ID|NAME>",
		Short: "删除指定的 API Key",
		Args:  cobra.ExactArgs(1),
		RunE:  runRemove,
	}

	removeCmd.Flags().BoolVarP(&removeYes, "yes", "y", false, "无需确认直接删除")

	RootCommand().AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	target := strings.TrimSpace(args[0])

	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	if !removeYes {
		fmt.Fprintf(cmd.OutOrStdout(), "%s 确认删除 %s? 输入 yes 继续: ",
			color.New(color.FgYellow, color.Bold).Sprint("⚠"),
			color.New(color.FgHiBlack).Sprint(target))
		reader := bufio.NewReader(cmd.InOrStdin())
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(confirm) != "yes" {
			fmt.Fprintln(cmd.OutOrStdout(), color.New(color.FgHiBlack).Sprint("操作已取消"))
			return nil
		}
	}

	key, err := manager.GetKey(target)
	if err != nil {
		key, err = manager.GetKeyByName(target)
		if err != nil {
			return fmt.Errorf("未找到 ID 或名称为 %s 的 Key", target)
		}
	}

	if err := manager.RemoveKey(key.ID); err != nil {
		return err
	}
	if err := manager.Save(); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s 已删除 Key: %s (%s)\n",
		color.New(color.FgRed, color.Bold).Sprint("−"),
		color.New(color.FgCyan, color.Bold).Sprint(key.Name),
		color.New(color.FgHiBlack).Sprint(key.ID))
	logging.Warnf("删除 Key: %s (%s)", key.Name, key.ID)
	return nil
}
