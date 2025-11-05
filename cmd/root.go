package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codex-switch/codex-switch/internal/logging"
	"github.com/codex-switch/codex-switch/internal/version"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgOverride string
	rootCmd     = &cobra.Command{
		Use:     "ckm",
		Short:   "codex-switch - 多 Key 管理工具",
		Long:    color.New(color.FgCyan).Sprintf("codex-switch\n一个用于管理 Codex/OpenAI API Key 的命令行工具"),
		Version: version.Version,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgOverride, "config", "c", "", "指定配置文件路径(默认位于用户目录)")
	cobra.OnInitialize(initConfig, initLogging)
}

// Execute 执行根命令，作为程序入口
func Execute() error {
	return rootCmd.Execute()
}

// initConfig 初始化配置文件位置，支持通过环境变量或参数覆盖
func initConfig() {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法获取用户目录: %v\n", err)
		os.Exit(1)
	}

	defaultDir := filepath.Join(userHome, ".codex-switch")
	defaultCfgPath := filepath.Join(defaultDir, "config.json")

	if cfgOverride != "" {
		viper.SetConfigFile(cfgOverride)
	} else {
		viper.SetConfigFile(defaultCfgPath)
	}

	viper.SetConfigType("json")
}

// initLogging 初始化日志系统，记录运行行为
func initLogging() {
	if err := logging.Init(""); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
	}
}

// RootCommand 返回根命令实例，方便其他包注册子命令
func RootCommand() *cobra.Command {
	return rootCmd
}
