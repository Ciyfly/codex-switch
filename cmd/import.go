package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	importInput  string
	importFormat string
	importMerge  bool
)

func init() {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "从文件导入配置",
		RunE:  runImport,
	}

	importCmd.Flags().StringVar(&importInput, "input", "", "待导入的配置文件路径")
	importCmd.Flags().StringVar(&importFormat, "format", "", "配置格式，默认根据扩展名推断")
	importCmd.Flags().BoolVar(&importMerge, "merge", false, "是否与现有配置合并")

	RootCommand().AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, _ []string) error {
	if strings.TrimSpace(importInput) == "" {
		return errors.New("请通过 --input 指定配置文件")
	}

	data, err := os.ReadFile(importInput)
	if err != nil {
		return err
	}

	format := importFormat
	if format == "" {
		format = inferFormat(importInput)
	}

	cfg, err := decodeConfig(data, format)
	if err != nil {
		return err
	}

	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	if importMerge {
		current, err := manager.Config()
		if err != nil {
			return err
		}
		merged := mergeConfig(current, cfg)
		if err := manager.ReplaceConfig(merged); err != nil {
			return err
		}
		logging.Infof("合并导入配置: %s", importInput)
	} else {
		if err := manager.ReplaceConfig(cfg); err != nil {
			return err
		}
		logging.Warnf("覆盖导入配置: %s", importInput)
	}

	if err := manager.Save(); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ 已导入配置，共 %d 个 Key\n", len(cfg.Keys))
	logging.Infof("导入完成，共 %d 个 Key", len(cfg.Keys))
	return nil
}

func decodeConfig(data []byte, format string) (*config.Config, error) {
	var cfg config.Config
	switch strings.ToLower(format) {
	case "json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	case "toml":
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("不支持的格式: %s", format)
	}
	return &cfg, nil
}

func inferFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	default:
		return "json"
	}
}

func mergeConfig(base *config.Config, incoming *config.Config) *config.Config {
	result := *base
	result.Keys = append([]config.APIKey(nil), base.Keys...)

	byID := make(map[string]int)
	byName := make(map[string]int)
	for i, k := range result.Keys {
		byID[k.ID] = i
		byName[strings.ToLower(k.Name)] = i
	}

	for _, newKey := range incoming.Keys {
		if idx, ok := byID[newKey.ID]; ok && newKey.ID != "" {
			result.Keys[idx] = newKey
			continue
		}
		if idx, ok := byName[strings.ToLower(newKey.Name)]; ok && newKey.Name != "" {
			result.Keys[idx] = newKey
			continue
		}
		result.Keys = append(result.Keys, newKey)
	}

	if incoming.ActiveKeyID != "" {
		result.ActiveKeyID = incoming.ActiveKeyID
	}
	if incoming.Version != "" {
		result.Version = incoming.Version
	}
	return &result
}
