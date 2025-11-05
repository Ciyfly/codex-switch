package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/codex-switch/codex-switch/internal/display"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/spf13/cobra"
)

var (
	listSort       string
	listFormat     string
	listFilterType string
)

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有已配置的 API Key",
		RunE:  runList,
	}

	listCmd.Flags().StringVar(&listSort, "sort", "default", "排序字段: default/name")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "输出格式: table/json")
	listCmd.Flags().StringVar(&listFilterType, "filter-type", "", "按类型筛选: openai/crs")

	RootCommand().AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	keys, err := manager.ListKeys(listSort)
	if err != nil {
		return err
	}

	if listFilterType != "" {
		filtered := keys[:0]
		target := strings.ToLower(listFilterType)
		for _, k := range keys {
			if strings.ToLower(k.Type) == target {
				filtered = append(filtered, k)
			}
		}
		keys = filtered
	}

	if listFormat == "json" {
		data, err := json.MarshalIndent(keys, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	display.PrintKeyTable(cmd.OutOrStdout(), keys)

	cfg, err := manager.Config()
	if err != nil {
		return err
	}

	activeName := "无"
	for _, k := range cfg.Keys {
		if k.Active {
			activeName = k.Name
			break
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n总计: %d 个 Key  |  当前激活: %s\n", len(keys), activeName)
	logging.Debugf("列出 %d 个 Key，激活: %s", len(keys), activeName)
	return nil
}
