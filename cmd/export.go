package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	exportOutput string
	exportFormat string
)

func init() {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "导出配置文件",
		RunE:  runExport,
	}

	exportCmd.Flags().StringVar(&exportOutput, "output", "", "导出文件路径，默认输出到标准输出")
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "导出格式: json/yaml/toml")

	RootCommand().AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, _ []string) error {
	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	cfg, err := manager.Config()
	if err != nil {
		return err
	}

	data, err := encodeConfig(cfg, exportFormat)
	if err != nil {
		return err
	}

	if exportOutput == "" {
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		logging.Debugf("导出配置到标准输出，格式: %s", exportFormat)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(exportOutput), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(exportOutput, data, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ 已导出配置至 %s\n", exportOutput)
	logging.Infof("导出配置: %s 格式: %s", exportOutput, exportFormat)
	return nil
}

func encodeConfig(cfg *config.Config, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(cfg, "", "  ")
	case "yaml":
		return yaml.Marshal(cfg)
	case "toml":
		return toml.Marshal(cfg)
	default:
		return nil, fmt.Errorf("不支持的格式: %s", format)
	}
}
