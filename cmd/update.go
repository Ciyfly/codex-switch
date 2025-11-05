package cmd

import (
	"fmt"
	"strings"

	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/spf13/cobra"
)

var (
	updateID         string
	updateName       string
	updateNewName    string
	updateAPIKey     string
	updateQuotaType  string
	updateTags       string
	updateConfigPath string
)

func init() {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "更新指定 Key 的配置",
		RunE:  runUpdate,
	}

	updateCmd.Flags().StringVar(&updateID, "id", "", "目标 Key 的 ID")
	updateCmd.Flags().StringVar(&updateName, "name", "", "目标 Key 的名称")
	updateCmd.Flags().StringVar(&updateNewName, "set-name", "", "更新后的名称")
	updateCmd.Flags().StringVar(&updateAPIKey, "set-key", "", "新的 API Key")
	updateCmd.Flags().StringVar(&updateQuotaType, "set-quota-type", "", "新的额度类型")
	updateCmd.Flags().StringVar(&updateTags, "set-tags", "", "重置标签(逗号分隔)")
	updateCmd.Flags().StringVar(&updateConfigPath, "set-config-file", "", "指定配置文件路径，使用文件内容完整替换 Codex config.toml")

	RootCommand().AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	if updateID == "" && updateName == "" {
		return fmt.Errorf("请通过 --id 或 --name 指定目标 Key")
	}

	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	var key config.APIKey
	if updateID != "" {
		key, err = manager.GetKey(updateID)
	} else {
		key, err = manager.GetKeyByName(updateName)
	}
	if err != nil {
		return err
	}

	updated := key
	if strings.TrimSpace(updateNewName) != "" {
		updated.Name = strings.TrimSpace(updateNewName)
	}
	if strings.TrimSpace(updateAPIKey) != "" {
		updated.APIKey = strings.TrimSpace(updateAPIKey)
	}
	if strings.TrimSpace(updateQuotaType) != "" {
		quotaType := strings.ToLower(strings.TrimSpace(updateQuotaType))
		if !supportedQuotaTypes[quotaType] {
			return fmt.Errorf("不支持的额度类型: %s", quotaType)
		}
		updated.QuotaType = quotaType
	}
	if cmd.Flags().Lookup("set-tags").Changed {
		updated.Tags = normalizeTags(updateTags)
	}
	if cmd.Flags().Lookup("set-config-file").Changed {
		rawConfig, err := loadRawConfigContent(updateConfigPath)
		if err != nil {
			return err
		}
		updated.RawConfig = rawConfig
	}
	if err := manager.UpdateKey(updated); err != nil {
		return err
	}
	if err := manager.Save(); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ 已更新 Key: %s\n", updated.Name)
	logging.Infof("更新 Key: %s (%s)", updated.Name, updated.ID)
	return nil
}
