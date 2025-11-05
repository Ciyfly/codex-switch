package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/display"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/spf13/cobra"
)

var (
	showID    string
	showName  string
	showField string
)

func init() {
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "显示指定或当前激活 Key 的详细信息",
		RunE:  runShow,
	}

	showCmd.Flags().StringVar(&showID, "id", "", "指定 Key ID")
	showCmd.Flags().StringVar(&showName, "name", "", "指定 Key 名称")
	showCmd.Flags().StringVar(&showField, "field", "", "仅输出某个字段，如 api_key/base_url/type")

	RootCommand().AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, _ []string) error {
	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	key, err := resolveShowTarget(manager)
	if err != nil {
		return err
	}

	if showField != "" {
		field := strings.ToLower(showField)
		value, err := extractField(field, key)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), value)
		logging.Debugf("输出 Key 字段 %s=%s", field, value)
		return nil
	}

	display.PrintKeyDetail(cmd.OutOrStdout(), key)
	logging.Debugf("展示 Key 详情: %s", key.ID)
	return nil
}

func resolveShowTarget(manager *config.Manager) (config.APIKey, error) {
	if showID != "" {
		return manager.GetKey(showID)
	}
	if showName != "" {
		return manager.GetKeyByName(showName)
	}
	return manager.ActiveKey()
}

func extractField(field string, key config.APIKey) (string, error) {
	switch field {
	case "api_key":
		return key.APIKey, nil
	case "base_url":
		return key.BaseURL, nil
	case "type":
		return key.Type, nil
	case "quota_type":
		return key.QuotaType, nil
	case "quota_limit":
		return fmt.Sprintf("%.2f", key.QuotaLimit), nil
	case "quota_used":
		return fmt.Sprintf("%.2f", key.QuotaUsed), nil
	case "raw_config":
		return key.RawConfig, nil
	case "name":
		return key.Name, nil
	case "id":
		return key.ID, nil
	default:
		return "", errors.New("不支持的字段")
	}
}
