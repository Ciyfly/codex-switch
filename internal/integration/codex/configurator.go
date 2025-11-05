package codex

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/logging"
)

// Configurator 负责根据 API Key 信息同步外部 Codex 配置
// 包含配置文件与认证文件的全量更新逻辑
type Configurator struct {
	ConfigPath string
	AuthPath   string
}

// Apply 根据指定 Key 更新配置文件与认证文件
func (c *Configurator) Apply(key config.APIKey) error {
	usingRaw := strings.TrimSpace(key.RawConfig) != ""
	logging.Infof("开始同步 Codex 配置: key=%s(%s), raw_config=%t, config_path=%s, auth_path=%s",
		key.Name, key.ID, usingRaw, c.ConfigPath, c.AuthPath)
	if err := c.updateConfigToml(key); err != nil {
		logging.Errorf("更新 config.toml 失败: %v", err)
		return err
	}
	if err := c.updateAuthJSON(key); err != nil {
		logging.Errorf("更新 auth.json 失败: %v", err)
		return err
	}
	logging.Infof("完成同步 Codex 配置: key=%s(%s)", key.Name, key.ID)
	return nil
}

// updateConfigToml 生成核心段落并与原文件内容合并
func (c *Configurator) updateConfigToml(key config.APIKey) error {
	if trimmed := strings.TrimSpace(key.RawConfig); trimmed != "" {
		content := sanitizeRawConfig(key.RawConfig)
		if strings.TrimSpace(content) == "" {
			logging.Warnf("原始配置经清理后为空，将回退为自动生成片段")
		} else {
			logging.Debugf("使用原始配置写入 config.toml，长度=%d", len(content))
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			if err := os.MkdirAll(filepath.Dir(c.ConfigPath), 0o755); err != nil {
				return fmt.Errorf("创建配置目录失败: %w", err)
			}
			temp := c.ConfigPath + ".tmp"
			if err := os.WriteFile(temp, []byte(content), 0o600); err != nil {
				return fmt.Errorf("写入临时文件失败: %w", err)
			}
			if err := os.Rename(temp, c.ConfigPath); err != nil {
				return fmt.Errorf("替换配置文件失败: %w", err)
			}
			return nil
		}
	}

	snippet := c.buildCoreSnippet(key)

	var rest string
	existing, err := os.ReadFile(c.ConfigPath)
	if err == nil {
		content := string(existing)
		lower := strings.ToLower(content)
		idx := strings.Index(lower, "\n[mcp_servers")
		if idx == -1 {
			idx = strings.Index(lower, "[mcp_servers")
		}
		if idx != -1 {
			rest = strings.TrimLeft(content[idx:], "\r\n")
			logging.Debugf("检测到已有 [mcp_servers] 段落，长度=%d", len(rest))
		} else {
			logging.Debugf("未在现有配置中找到 [mcp_servers] 段落，将直接覆盖核心片段")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	builder := &strings.Builder{}
	builder.WriteString(snippet)
	builder.WriteString("\n")
	if rest != "" {
		builder.WriteString("\n")
		builder.WriteString(rest)
		builder.WriteString("\n")
	}

	if err := os.MkdirAll(filepath.Dir(c.ConfigPath), 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	temp := c.ConfigPath + ".tmp"
	if err := os.WriteFile(temp, []byte(builder.String()), 0o600); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := os.Rename(temp, c.ConfigPath); err != nil {
		return fmt.Errorf("替换配置文件失败: %w", err)
	}
	return nil
}

// updateAuthJSON 更新认证文件中的 API Key
func (c *Configurator) updateAuthJSON(key config.APIKey) error {
	payload := map[string]string{
		"OPENAI_API_KEY": key.APIKey,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.AuthPath), 0o755); err != nil {
		return fmt.Errorf("创建认证目录失败: %w", err)
	}

	temp := c.AuthPath + ".tmp"
	if err := os.WriteFile(temp, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("写入认证临时文件失败: %w", err)
	}
	if err := os.Rename(temp, c.AuthPath); err != nil {
		return fmt.Errorf("替换认证文件失败: %w", err)
	}
	return nil
}

// buildCoreSnippet 根据 Key 类型生成核心配置段
func (c *Configurator) buildCoreSnippet(key config.APIKey) string {
	if trimmed := strings.TrimSpace(key.RawConfig); trimmed != "" {
		return sanitizeRawConfig(key.RawConfig)
	}

	provider := strings.TrimSpace(key.Provider)
	baseURL := strings.TrimSpace(key.BaseURL)
	envKey := strings.TrimSpace(key.EnvKey)

	template := classifyProvider(provider, key.Type, baseURL)
	switch template {
	case "crs":
		if baseURL == "" {
			baseURL = "https://ki1.me/openai"
		}
		if envKey == "" {
			envKey = "CRS_OAI_KEY"
		}
		return fmt.Sprintf(`model_provider = "crs"
model = "gpt-5-codex"
model_reasoning_effort = "high"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.crs]
name = "crs"
base_url = "%s"
wire_api = "responses"
requires_openai_auth = true
env_key = "%s"
`, baseURL, envKey)
	case "duckcoding":
		if baseURL == "" {
			baseURL = "https://jp.duckcoding.com/v1"
		}
		return fmt.Sprintf(`model_provider = "duckcoding"
model = "gpt-5-codex"
model_reasoning_effort = "high"
network_access = "enabled"
disable_response_storage = true

[model_providers.duckcoding]
name = "duckcoding"
base_url = "%s"
wire_api = "responses"
requires_openai_auth = true
`, baseURL)
	default:
		return buildGenericSnippet(key)
	}
}

// deriveProviderName 根据 BaseURL 自动推导提供商名称
func deriveProviderName(base string) string {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed.Host == "" {
		return "custom"
	}
	host := strings.TrimPrefix(parsed.Host, "www.")
	parts := strings.Split(host, ".")
	if len(parts) > 2 {
		parts = parts[1:]
	}
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[len(parts)-2])
	}
	return strings.TrimSpace(parts[0])
}

// NewConfigurator 创建配置器实例，提供默认路径
func NewConfigurator(configPath, authPath string) (*Configurator, error) {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(home, ".codex", "config.toml")
	}
	if authPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		authPath = filepath.Join(home, ".codex", "auth.json")
	}
	return &Configurator{ConfigPath: configPath, AuthPath: authPath}, nil
}

// Timestamp 输出当前时间，方便记录同步行为
func Timestamp() string {
	return time.Now().Format(time.RFC3339)
}

// sanitizeRawConfig 清理 ANSI 控制符、回车与退格，避免污染生成的配置文件
func sanitizeRawConfig(content string) string {
	result := make([]rune, 0, len(content))

	inEscape := false
	for _, r := range content {
		switch {
		case r == '\x1b':
			inEscape = true
			continue
		case inEscape:
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		case r == '\r':
			continue
		case r == '\b':
			if n := len(result); n > 0 {
				result = result[:n-1]
			}
			continue
		default:
			result = append(result, r)
		}
	}
	clean := string(result)
	return trimInteractiveSentinel(clean)
}

func trimInteractiveSentinel(content string) string {
	lines := strings.Split(content, "\n")
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "" {
			lines = lines[:len(lines)-1]
			continue
		}
		if strings.EqualFold(last, "end") {
			lines = lines[:len(lines)-1]
		}
		break
	}
	return strings.Join(lines, "\n")
}

func classifyProvider(provider, keyType, baseURL string) string {
	lowerProvider := strings.ToLower(strings.TrimSpace(provider))
	if lowerProvider != "" {
		if lowerProvider == "crs" {
			return "crs"
		}
		if lowerProvider == "duckcoding" || lowerProvider == "openai" {
			return "duckcoding"
		}
		return ""
	}

	if strings.EqualFold(strings.TrimSpace(keyType), config.TypeCRS) {
		return "crs"
	}

	lowerURL := strings.ToLower(strings.TrimSpace(baseURL))
	switch {
	case strings.Contains(lowerURL, "ki1.me"), strings.Contains(lowerURL, "crs"):
		return "crs"
	case strings.Contains(lowerURL, "duckcoding"), strings.Contains(lowerURL, "openai"):
		return "duckcoding"
	default:
		return ""
	}
}

func buildGenericSnippet(key config.APIKey) string {
	provider := strings.TrimSpace(key.Provider)
	if provider == "" {
		if strings.EqualFold(key.Type, config.TypeCRS) {
			provider = "crs"
		} else {
			provider = deriveProviderName(key.BaseURL)
		}
	}

	preferredAuth := strings.TrimSpace(key.PreferredAuthMethod)
	if preferredAuth == "" {
		preferredAuth = "apikey"
	}

	wireAPI := strings.TrimSpace(key.WireAPI)
	if wireAPI == "" {
		wireAPI = "responses"
	}

	requiresAuth := true
	if key.RequiresOpenAIAuth != nil {
		requiresAuth = *key.RequiresOpenAIAuth
	}

	baseURL := strings.TrimSpace(key.BaseURL)

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`model_provider = "%s"
model = "gpt-5-codex"
model_reasoning_effort = "high"
disable_response_storage = true
preferred_auth_method = "%s"
`, provider, preferredAuth))

	if strings.TrimSpace(key.Provider) == "" && !strings.EqualFold(provider, "crs") {
		builder.WriteString("network_access = \"enabled\"\n")
	}

	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf(`[model_providers.%s]
name = "%s"
base_url = "%s"
wire_api = "%s"
requires_openai_auth = %t
`, provider, provider, baseURL, wireAPI, requiresAuth))

	if strings.TrimSpace(key.EnvKey) != "" {
		builder.WriteString(fmt.Sprintf("env_key = \"%s\"\n", strings.TrimSpace(key.EnvKey)))
	}

	return builder.String()
}
