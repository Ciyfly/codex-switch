package cmd

import (
	"fmt"
	"strings"

	"github.com/codex-switch/codex-switch/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// init 中注册子命令使用的工具函数

// mustLoadManager 根据当前配置文件路径加载配置管理器
func mustLoadManager(cmd *cobra.Command) (*config.Manager, error) {
	cfgPath := viper.ConfigFileUsed()
	manager, err := config.NewDefaultManager(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("创建配置管理器失败: %w", err)
	}
	if _, err := manager.Load(); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}
	return manager, nil
}

// normalizeTags 将逗号分隔字符串拆分为标签列表
func normalizeTags(input string) []string {
	if strings.TrimSpace(input) == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
