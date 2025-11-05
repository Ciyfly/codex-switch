package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codex-switch/codex-switch/internal/config"
)

func TestDeriveProviderName(t *testing.T) {
	cases := map[string]string{
		"https://jp.duckcoding.com/v1": "duckcoding",
		"https://api.openai.com":       "openai",
		"invalid-url":                  "custom",
	}
	for input, want := range cases {
		if got := deriveProviderName(input); got != want {
			t.Fatalf("deriveProviderName(%s)=%s, want %s", input, got, want)
		}
	}
}

func TestConfiguratorApply(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}
	key := config.APIKey{
		ID:      "k1",
		Name:    "测试",
		APIKey:  "sk-test",
		BaseURL: "https://jp.duckcoding.com/v1",
		Type:    "openai",
	}

	if err := conf.Apply(key); err != nil {
		t.Fatalf("Apply 返回错误: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `model_provider = "duckcoding"`) {
		t.Fatalf("配置片段未包含推导的提供商名称, got: %s", content)
	}

	auth, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("读取认证文件失败: %v", err)
	}
	if string(auth) == "" {
		t.Fatalf("认证文件应有内容")
	}
}

func TestConfiguratorCustomProvider(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}
	requires := true
	key := config.APIKey{
		ID:                  "ctl",
		Name:                "CTL",
		APIKey:              "sk-ctl",
		BaseURL:             "http://218.249.73.247:3000/openai",
		Type:                "openai",
		Provider:            "ctl",
		PreferredAuthMethod: "apikey",
		WireAPI:             "responses",
		EnvKey:              "CTL_OAI_KEY",
		RequiresOpenAIAuth:  &requires,
	}

	if err := conf.Apply(key); err != nil {
		t.Fatalf("Apply 返回错误: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `model_provider = "ctl"`) {
		t.Fatalf("未找到自定义 provider: %s", content)
	}
	if !strings.Contains(content, `env_key = "CTL_OAI_KEY"`) {
		t.Fatalf("未找到自定义 env_key: %s", content)
	}
}

func TestConfiguratorRawConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}
	raw := "model_provider = \"custom\"\n[model_providers.custom]\nname = \"custom\"\nbase_url = \"https://example.com\"\n"
	key := config.APIKey{
		ID:        "raw",
		Name:      "原始配置",
		APIKey:    "sk-raw",
		RawConfig: raw,
	}

	if err := conf.Apply(key); err != nil {
		t.Fatalf("Apply 返回错误: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	if strings.TrimSpace(string(data)) != strings.TrimSpace(raw) {
		t.Fatalf("配置内容未按原样保存: %q", string(data))
	}

	auth, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("读取认证文件失败: %v", err)
	}
	if !strings.Contains(string(auth), "sk-raw") {
		t.Fatalf("认证文件未同步 Key: %s", string(auth))
	}
}

func TestConfiguratorOpenAIDefaultTemplate(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}
	key := config.APIKey{
		ID:      "openai",
		Name:    "OpenAI 默认",
		APIKey:  "sk-openai",
		BaseURL: "https://jp.duckcoding.com/v1",
		Type:    config.TypeOpenAI,
	}

	if err := conf.Apply(key); err != nil {
		t.Fatalf("Apply 返回错误: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	content := string(data)
	expectedLines := []string{
		`model_provider = "duckcoding"`,
		`model = "gpt-5-codex"`,
		`network_access = "enabled"`,
		`disable_response_storage = true`,
		`[model_providers.duckcoding]`,
		`base_url = "https://jp.duckcoding.com/v1"`,
	}
	for _, line := range expectedLines {
		if !strings.Contains(content, line) {
			t.Fatalf("缺少默认 OpenAI 模板行 %q，实际内容: %s", line, content)
		}
	}
}

func TestConfiguratorCRSTemplateDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}
	key := config.APIKey{
		ID:     "crs",
		Name:   "CRS 默认",
		APIKey: "sk-crs",
		Type:   config.TypeCRS,
	}

	if err := conf.Apply(key); err != nil {
		t.Fatalf("Apply 返回错误: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	content := string(data)
	expected := map[string]string{
		`model_provider = "crs"`:             "",
		`preferred_auth_method = "apikey"`:   "",
		`base_url = "https://ki1.me/openai"`: "",
		`env_key = "CRS_OAI_KEY"`:            "",
		`wire_api = "responses"`:             "",
		`requires_openai_auth = true`:        "",
		`model_reasoning_effort = "high"`:    "",
		`disable_response_storage = true`:    "",
		`[model_providers.crs]`:              "",
	}
	for line := range expected {
		if !strings.Contains(content, line) {
			t.Fatalf("缺少默认 CRS 模板行 %q，实际内容: %s", line, content)
		}
	}
}

func TestConfiguratorSwitchFromRawToGenerated(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}

	raw := `# 自定义段落
model_provider = "custom"
model = "gpt-4"

[model_providers.custom]
name = "custom"
base_url = "https://example.com"

[mcp_servers.cli]
transport = "stdio"
command = "/usr/bin/ckm"
`
	rawKey := config.APIKey{
		ID:        "raw",
		Name:      "原始配置",
		APIKey:    "sk-raw",
		RawConfig: raw,
	}
	if err := conf.Apply(rawKey); err != nil {
		t.Fatalf("首次 Apply 返回错误: %v", err)
	}

	nextKey := config.APIKey{
		ID:      "next",
		Name:    "后续配置",
		APIKey:  "sk-next",
		BaseURL: "https://jp.duckcoding.com/v1",
		Type:    config.TypeOpenAI,
	}
	if err := conf.Apply(nextKey); err != nil {
		t.Fatalf("二次 Apply 返回错误: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `model_provider = "duckcoding"`) {
		t.Fatalf("未写入新的 model_provider，文件内容: %s", content)
	}
	if strings.Contains(content, `model_provider = "custom"`) {
		t.Fatalf("旧的 RawConfig 仍然存在，文件内容: %s", content)
	}
	if !strings.Contains(content, "[mcp_servers.cli]") {
		t.Fatalf("原有 mcp_servers 段落未保留: %s", content)
	}

	auth, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("读取认证文件失败: %v", err)
	}
	if !strings.Contains(string(auth), "sk-next") {
		t.Fatalf("认证文件未更新 Key，内容: %s", string(auth))
	}
}

func TestConfiguratorIntegrationWithManagerSwitch(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}

	storage := config.NewMemoryStorage()
	manager := config.NewManager(storage)

	if _, err := manager.Load(); err != nil {
		t.Fatalf("初始化配置失败: %v", err)
	}

	rawKey := config.APIKey{
		ID:     "1",
		Name:   "Key-1",
		APIKey: "sk-raw",
		RawConfig: "model_provider = \"custom\"\n[model_providers.custom]\nname = \"custom\"\nbase_url = \"https://example.com\"\n\n" +
			"[mcp_servers.cli]\ntransport = \"stdio\"\ncommand = \"ckm\"\n",
		Active: true,
	}
	if _, err := manager.AddKey(rawKey); err != nil {
		t.Fatalf("添加 RawConfig Key 失败: %v", err)
	}

	nextKey := config.APIKey{
		ID:      "2",
		Name:    "Key-2",
		APIKey:  "sk-next",
		BaseURL: "https://jp.duckcoding.com/v1",
		Type:    config.TypeOpenAI,
	}
	if _, err := manager.AddKey(nextKey); err != nil {
		t.Fatalf("添加第二个 Key 失败: %v", err)
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("保存初始配置失败: %v", err)
	}

	active, err := manager.ActiveKey()
	if err != nil {
		t.Fatalf("读取激活 Key 失败: %v", err)
	}
	if err := conf.Apply(active); err != nil {
		t.Fatalf("首次同步失败: %v", err)
	}

	if err := manager.SetActiveKey("2"); err != nil {
		t.Fatalf("切换 Key 失败: %v", err)
	}
	if err := manager.TouchKey("2"); err != nil {
		t.Fatalf("更新最后使用时间失败: %v", err)
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("保存切换后的配置失败: %v", err)
	}

	// 模拟新进程重新加载配置
	manager2 := config.NewManager(storage)
	if _, err := manager2.Load(); err != nil {
		t.Fatalf("重新加载配置失败: %v", err)
	}
	active2, err := manager2.ActiveKey()
	if err != nil {
		t.Fatalf("读取切换后的激活 Key 失败: %v", err)
	}
	if active2.ID != "2" {
		t.Fatalf("激活 Key 应为 2，实际为 %s", active2.ID)
	}

	if err := conf.Apply(active2); err != nil {
		t.Fatalf("切换后同步失败: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `model_provider = "duckcoding"`) {
		t.Fatalf("未生成新的配置片段: %s", content)
	}
	if strings.Contains(content, `model_provider = "custom"`) {
		t.Fatalf("旧配置片段仍然存在: %s", content)
	}
	if !strings.Contains(content, "[mcp_servers.cli]") {
		t.Fatalf("原有 mcp_servers 段落未保留: %s", content)
	}
	authData, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("读取认证文件失败: %v", err)
	}
	if !strings.Contains(string(authData), "sk-next") {
		t.Fatalf("认证文件 Key 未更新: %s", string(authData))
	}
}

func TestConfiguratorPreserveUppercaseMCPServers(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}

	initial := `model_provider = "custom"
model = "gpt-4"

[MODEL_PROVIDERS.CUSTOM]
name = "custom"
base_url = "https://example.com"

[MCP_SERVERS.CLI]
transport = "stdio"
command = "ckm"
`
	rawKey := config.APIKey{
		ID:        "raw-upper",
		Name:      "原始配置大写",
		APIKey:    "sk-raw-upper",
		RawConfig: initial,
	}
	if err := conf.Apply(rawKey); err != nil {
		t.Fatalf("初始化 RawConfig 失败: %v", err)
	}

	nextKey := config.APIKey{
		ID:      "auto",
		Name:    "自动生成",
		APIKey:  "sk-auto",
		BaseURL: "https://api.openai.com/v1",
		Type:    config.TypeOpenAI,
	}
	if err := conf.Apply(nextKey); err != nil {
		t.Fatalf("应用自动配置失败: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[MCP_SERVERS.CLI]") {
		t.Fatalf("未保留原有大写 MCP_SERVERS 段落: %s", content)
	}
	if !strings.Contains(content, `model_provider = "duckcoding"`) {
		t.Fatalf("未生成新的 model_provider，文件内容: %s", content)
	}
}

func TestSanitizeRawConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	authPath := filepath.Join(dir, "auth.json")

	conf := &Configurator{ConfigPath: cfgPath, AuthPath: authPath}
	raw := "model_provider = \"crs\"\nmodel = \"gpt-5\"\x1b[31m\n[model_providers.crs]\nname = \"crs\"\nbase_url = \"https://example.com\"\nrequires_openai_auth = true\n\x1bend\b\b"
	key := config.APIKey{
		ID:        "clr",
		Name:      "带控制字符",
		APIKey:    "sk-clear",
		RawConfig: raw,
	}

	if err := conf.Apply(key); err != nil {
		t.Fatalf("Apply 返回错误: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	content := string(data)
	if strings.ContainsRune(content, '\x1b') || strings.ContainsRune(content, '\b') || strings.ContainsRune(content, '\r') {
		t.Fatalf("清理后仍包含控制字符: %q", content)
	}
	if !strings.Contains(content, `model_provider = "crs"`) {
		t.Fatalf("核心内容被意外清理: %s", content)
	}
	if !strings.HasSuffix(content, "\n") {
		t.Fatalf("清理后应保留换行结尾")
	}
	if !strings.HasSuffix(strings.TrimSpace(content), "requires_openai_auth = true") {
		t.Fatalf("尾部应保留配置主体，实际为: %s", content)
	}
}
