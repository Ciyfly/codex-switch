package cmd

import (
	"fmt"

	"github.com/codex-switch/codex-switch/internal/config"
	codex "github.com/codex-switch/codex-switch/internal/integration/codex"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/spf13/cobra"
)

var (
	codexConfigPath string
	codexAuthPath   string
	codexKeyID      string
)

func init() {
	codexCmd := &cobra.Command{
		Use:   "codex-config",
		Short: "同步 Codex 配置文件和认证文件",
		RunE:  runCodexConfig,
	}

	codexCmd.Flags().StringVar(&codexConfigPath, "config", "", "Codex 配置文件路径，默认 ~/.codex/config.toml")
	codexCmd.Flags().StringVar(&codexAuthPath, "auth", "", "Codex 认证文件路径，默认 ~/.codex/auth.json")
	codexCmd.Flags().StringVar(&codexKeyID, "id", "", "指定使用的 Key ID，不填则使用当前激活 Key")

	RootCommand().AddCommand(codexCmd)
}

func runCodexConfig(cmd *cobra.Command, _ []string) error {
	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	var key config.APIKey
	if codexKeyID != "" {
		key, err = manager.GetKey(codexKeyID)
		if err != nil {
			return err
		}
	} else {
		key, err = manager.ActiveKey()
		if err != nil {
			return fmt.Errorf("未找到激活的 Key，请先使用 ckm switch 设置激活 Key: %w", err)
		}
	}

	configurator, err := codex.NewConfigurator(codexConfigPath, codexAuthPath)
	if err != nil {
		return err
	}

	if err := configurator.Apply(key); err != nil {
		return err
	}

	logging.Infof("已同步 Codex 配置，时间: %s", codex.Timestamp())
	fmt.Fprintf(cmd.OutOrStdout(), "✓ 已更新 Codex 配置，使用 Key: %s (%s)\n", key.Name, key.ID)
	return nil
}
