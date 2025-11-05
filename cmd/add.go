package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/spf13/cobra"
)

var (
	addName       string
	addAPIKey     string
	addQuotaType  string
	addTags       string
	addConfigPath string
)

var supportedQuotaTypes = map[string]bool{
	config.QuotaDaily:     true,
	config.QuotaWeekly:    true,
	config.QuotaMonthly:   true,
	config.QuotaYearly:    true,
	config.QuotaUnlimited: true,
}

func init() {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "添加一个新的 API Key",
		RunE:  runAdd,
	}

	addCmd.Flags().StringVar(&addName, "name", "", "Key 名称")
	addCmd.Flags().StringVar(&addAPIKey, "key", "", "API Key 内容")
	addCmd.Flags().StringVar(&addQuotaType, "quota-type", config.QuotaMonthly, "额度类型(daily/weekly/monthly/yearly/unlimited)")
	addCmd.Flags().StringVar(&addTags, "tags", "", "标签，逗号分隔")
	addCmd.Flags().StringVar(&addConfigPath, "config-file", "", "配置文件路径，使用文件内容完整替换 Codex config.toml")

	RootCommand().AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, _ []string) error {
	name := strings.TrimSpace(addName)
	if name == "" {
		return fmt.Errorf("必须提供名称")
	}

	apiKey := strings.TrimSpace(addAPIKey)
	if apiKey == "" {
		return fmt.Errorf("必须提供 API Key")
	}

	configPath := strings.TrimSpace(addConfigPath)
	if configPath == "" {
		return fmt.Errorf("必须通过 --config-file 指定配置文件路径")
	}

	rawConfig, err := loadRawConfigContent(configPath)
	if err != nil {
		return err
	}

	quotaType := strings.ToLower(strings.TrimSpace(addQuotaType))
	if quotaType == "" {
		quotaType = config.QuotaMonthly
	}
	if !supportedQuotaTypes[quotaType] {
		return fmt.Errorf("不支持的额度类型: %s", quotaType)
	}

	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	newKey := config.APIKey{
		Name:       name,
		APIKey:     apiKey,
		QuotaType:  quotaType,
		QuotaLimit: 0,
		QuotaUsed:  0,
		Tags:       normalizeTags(addTags),
		RawConfig:  rawConfig,
	}

	created, err := manager.AddKey(newKey)
	if err != nil {
		return err
	}

	if err := manager.Save(); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ 成功添加 API Key: %s (%s)\n", created.Name, created.ID)
	logging.Infof("添加 Key: %s (%s)", created.Name, created.ID)
	return nil
}

func loadRawConfigContent(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("配置文件路径不能为空")
	}
	trimmed := strings.TrimSpace(path)

	var finalPath string
	if strings.HasPrefix(trimmed, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("解析配置文件路径失败: %w", err)
		}
		rel := strings.TrimPrefix(trimmed, "~")
		rel = strings.TrimPrefix(rel, string(os.PathSeparator))
		finalPath = filepath.Join(home, rel)
	} else {
		abs, err := filepath.Abs(trimmed)
		if err != nil {
			return "", fmt.Errorf("解析配置文件路径失败: %w", err)
		}
		finalPath = abs
	}

	data, err := os.ReadFile(finalPath)
	if err != nil {
		return "", fmt.Errorf("读取配置文件失败: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("配置文件内容为空: %s", finalPath)
	}
	return string(data), nil
}
