package config

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 支持的 API 类型
const (
	TypeOpenAI = "openai"
	TypeCRS    = "crs"
)

// 默认配置版本
const defaultVersion = "1.0.0"
const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// APIKey 表示单个 Key 的完整信息
type APIKey struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	APIKey              string    `json:"api_key"`
	BaseURL             string    `json:"base_url"`
	Type                string    `json:"type"`
	Description         string    `json:"description"`
	CreatedAt           time.Time `json:"created_at"`
	LastChecked         time.Time `json:"last_checked"`
	LastUsed            time.Time `json:"last_used"`
	Active              bool      `json:"active"`
	Tags                []string  `json:"tags"`
	Provider            string    `json:"provider,omitempty"`
	PreferredAuthMethod string    `json:"preferred_auth_method,omitempty"`
	WireAPI             string    `json:"wire_api,omitempty"`
	EnvKey              string    `json:"env_key,omitempty"`
	RequiresOpenAIAuth  *bool     `json:"requires_openai_auth,omitempty"`
	RawConfig           string    `json:"raw_config,omitempty"`
}

// Config 表示配置文件的顶层结构
type Config struct {
	Version     string          `json:"version"`
	ActiveKeyID string          `json:"active_key_id"`
	Keys        []APIKey        `json:"keys"`
	LastUpdated time.Time       `json:"last_updated"`
	NextID      int             `json:"next_id,omitempty"`
	Remote      *RemoteSettings `json:"remote,omitempty"`
}

// Manager 负责管理配置的读写及业务逻辑
type Manager struct {
	storage Storage
	mu      sync.RWMutex
	cfg     *Config
	loaded  bool
}

// NewDefaultManager 根据路径创建默认文件存储的管理器
func NewDefaultManager(path string) (*Manager, error) {
	storage, err := NewFileStorage(path)
	if err != nil {
		return nil, err
	}
	return NewManager(storage), nil
}

// NewManager 创建 Manager 实例
func NewManager(storage Storage) *Manager {
	return &Manager{storage: storage}
}

// Load 加载配置文件，如果不存在则创建空配置
func (m *Manager) Load() (*Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loaded {
		return m.cfg, nil
	}

	cfg, err := m.storage.Load()
	if err != nil {
		return nil, err
	}

	if cfg.Version == "" {
		cfg.Version = defaultVersion
	}
	if cfg.Keys == nil {
		cfg.Keys = []APIKey{}
	}
	remoteChanged := normalizeRemoteSettings(cfg)
	changed := ensureNextID(cfg)
	if remoteChanged {
		changed = true
	}
	for i := range cfg.Keys {
		setIntegrationDefaults(&cfg.Keys[i])
	}

	m.cfg = cfg
	m.loaded = true
	if changed {
		cfg.LastUpdated = time.Now().UTC()
		_ = m.storage.Save(cfg)
	}
	return cfg, nil
}

// Save 将当前配置持久化到磁盘
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cfg == nil {
		return errors.New("配置尚未加载")
	}

	m.cfg.LastUpdated = time.Now().UTC()
	return m.storage.Save(m.cfg)
}

// ReplaceConfig 完整替换当前配置，常用于导入场景
func (m *Manager) ReplaceConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("传入配置为空")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.Version == "" {
		cfg.Version = defaultVersion
	}
	if cfg.Keys == nil {
		cfg.Keys = []APIKey{}
	}
	remoteChanged := normalizeRemoteSettings(cfg)
	changed := ensureNextID(cfg)
	if remoteChanged {
		changed = true
	}
	for i := range cfg.Keys {
		setIntegrationDefaults(&cfg.Keys[i])
	}

	activeID := cfg.ActiveKeyID
	activeCount := 0
	for i := range cfg.Keys {
		if cfg.Keys[i].ID == "" {
			cfg.Keys[i].ID = uuid.NewString()
		}
		if cfg.Keys[i].Active {
			activeCount++
			activeID = cfg.Keys[i].ID
		}
	}

	if activeCount == 0 && len(cfg.Keys) > 0 {
		cfg.Keys[0].Active = true
		activeID = cfg.Keys[0].ID
	}
	cfg.ActiveKeyID = activeID

	for i := range cfg.Keys {
		cfg.Keys[i].Active = cfg.Keys[i].ID == activeID
	}

	cfg.LastUpdated = time.Now().UTC()
	m.cfg = cfg
	m.loaded = true
	if changed {
		_ = m.storage.Save(cfg)
	}
	return nil
}

// Config 返回当前配置副本
func (m *Manager) Config() (*Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return nil, errors.New("配置尚未加载")
	}

	cfgCopy := *m.cfg
	cfgCopy.Keys = append([]APIKey(nil), m.cfg.Keys...)
	return &cfgCopy, nil
}

// ConfigPath 返回当前存储实现使用的配置文件路径，方便外部功能定位目录。
func (m *Manager) ConfigPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storage.Path()
}

// AddKey 添加新的 API Key 记录
func (m *Manager) AddKey(input APIKey) (APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.loaded {
		return APIKey{}, errors.New("配置尚未加载")
	}

	if strings.TrimSpace(input.Name) == "" {
		return APIKey{}, errors.New("名称不能为空")
	}

	if strings.TrimSpace(input.APIKey) == "" {
		return APIKey{}, errors.New("API Key 不能为空")
	}

	if input.Type == "" {
		input.Type = TypeOpenAI
	}

	if input.ID == "" {
		input.ID = m.generateID()
	}

	setIntegrationDefaults(&input)

	now := time.Now().UTC()
	if input.CreatedAt.IsZero() {
		input.CreatedAt = now
	}
	if input.LastChecked.IsZero() {
		input.LastChecked = now
	}
	if input.LastUsed.IsZero() {
		input.LastUsed = now
	}

	for _, k := range m.cfg.Keys {
		if strings.EqualFold(k.Name, input.Name) {
			return APIKey{}, fmt.Errorf("名称 %s 已存在", input.Name)
		}
		if k.ID == input.ID {
			return APIKey{}, fmt.Errorf("ID %s 已存在", input.ID)
		}
	}

	m.cfg.Keys = append(m.cfg.Keys, input)
	if input.Active || m.cfg.ActiveKeyID == "" {
		m.setActiveKeyLocked(input.ID)
	}

	return input, nil
}

// UpdateKey 更新既有 Key 信息
func (m *Manager) UpdateKey(updated APIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.loaded {
		return errors.New("配置尚未加载")
	}

	idx := -1
	for i, k := range m.cfg.Keys {
		if k.ID == updated.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("未找到 ID %s", updated.ID)
	}

	existing := m.cfg.Keys[idx]
	if updated.Name == "" {
		updated.Name = existing.Name
	}
	if updated.APIKey == "" {
		updated.APIKey = existing.APIKey
	}
	if updated.BaseURL == "" {
		updated.BaseURL = existing.BaseURL
	}
	if updated.Type == "" {
		updated.Type = existing.Type
	}
	if updated.CreatedAt.IsZero() {
		updated.CreatedAt = existing.CreatedAt
	}
	if updated.LastChecked.IsZero() {
		updated.LastChecked = existing.LastChecked
	}
	if updated.LastUsed.IsZero() {
		updated.LastUsed = existing.LastUsed
	}

	mergeIntegrationDefaults(&updated, existing)

	m.cfg.Keys[idx] = updated
	if updated.Active {
		m.setActiveKeyLocked(updated.ID)
	}
	return nil
}

// RemoveKey 根据 ID 删除 Key
func (m *Manager) RemoveKey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.loaded {
		return errors.New("配置尚未加载")
	}

	idx := -1
	for i, k := range m.cfg.Keys {
		if k.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("未找到 ID %s", id)
	}

	m.cfg.Keys = append(m.cfg.Keys[:idx], m.cfg.Keys[idx+1:]...)
	if m.cfg.ActiveKeyID == id {
		m.cfg.ActiveKeyID = ""
		if len(m.cfg.Keys) > 0 {
			m.setActiveKeyLocked(m.cfg.Keys[0].ID)
		}
	}
	return nil
}

// SetActiveKey 设置当前激活 Key
func (m *Manager) SetActiveKey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.setActiveKeyLocked(id)
}

func (m *Manager) setActiveKeyLocked(id string) error {
	if !m.loaded {
		return errors.New("配置尚未加载")
	}

	found := false
	for i, k := range m.cfg.Keys {
		if k.ID == id {
			found = true
			m.cfg.Keys[i].Active = true
			m.cfg.ActiveKeyID = id
		} else {
			m.cfg.Keys[i].Active = false
		}
	}

	if !found {
		return fmt.Errorf("未找到 ID %s", id)
	}
	return nil
}

// GetKey 返回指定 ID 的 Key
func (m *Manager) GetKey(id string) (APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return APIKey{}, errors.New("配置尚未加载")
	}

	for _, k := range m.cfg.Keys {
		if k.ID == id {
			return k, nil
		}
	}
	return APIKey{}, fmt.Errorf("未找到 ID %s", id)
}

// GetKeyByName 通过名称查找 Key
func (m *Manager) GetKeyByName(name string) (APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return APIKey{}, errors.New("配置尚未加载")
	}

	for _, k := range m.cfg.Keys {
		if strings.EqualFold(k.Name, name) {
			return k, nil
		}
	}
	return APIKey{}, fmt.Errorf("未找到名称 %s", name)
}

// ActiveKey 返回当前激活的 Key
func (m *Manager) ActiveKey() (APIKey, error) {
	if !m.loaded {
		return APIKey{}, errors.New("配置尚未加载")
	}
	return m.GetKey(m.cfg.ActiveKeyID)
}

// ListKeys 返回排序后的 Key 列表
func (m *Manager) ListKeys(sortBy string) ([]APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return nil, errors.New("配置尚未加载")
	}

	items := append([]APIKey(nil), m.cfg.Keys...)

	switch sortBy {
	case "name":
		sort.Slice(items, func(i, j int) bool {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			if items[i].Active == items[j].Active {
				return items[i].CreatedAt.Before(items[j].CreatedAt)
			}
			return items[i].Active
		})
	}

	return items, nil
}

// TouchKey 更新最后使用时间
func (m *Manager) TouchKey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.loaded {
		return errors.New("配置尚未加载")
	}

	for i, k := range m.cfg.Keys {
		if k.ID == id {
			m.cfg.Keys[i].LastUsed = time.Now().UTC()
			return nil
		}
	}

	return fmt.Errorf("未找到 ID %s", id)
}

func (m *Manager) generateID() string {
	if m.cfg.NextID <= 0 {
		ensureNextID(m.cfg)
	}
	id := strconv.Itoa(m.cfg.NextID)
	m.cfg.NextID++
	return id
}

func ensureNextID(cfg *Config) bool {
	if cfg.Keys == nil {
		cfg.Keys = []APIKey{}
	}

	needsNormalize := false
	maxID := 0
	for _, k := range cfg.Keys {
		if n, err := strconv.Atoi(k.ID); err == nil {
			if n > maxID {
				maxID = n
			}
		} else {
			needsNormalize = true
		}
	}

	if needsNormalize {
		mapping := make(map[string]string, len(cfg.Keys))
		for i := range cfg.Keys {
			old := cfg.Keys[i].ID
			newID := strconv.Itoa(i + 1)
			cfg.Keys[i].ID = newID
			mapping[old] = newID
		}
		if updated, ok := mapping[cfg.ActiveKeyID]; ok {
			cfg.ActiveKeyID = updated
		}
		maxID = len(cfg.Keys)
	}

	if cfg.NextID <= 0 || cfg.NextID <= maxID {
		cfg.NextID = maxID + 1
		if cfg.NextID <= 0 {
			cfg.NextID = len(cfg.Keys) + 1
		}
	}
	return needsNormalize
}

func setIntegrationDefaults(key *APIKey) {
	if key.PreferredAuthMethod == "" {
		key.PreferredAuthMethod = "apikey"
	}
	if key.WireAPI == "" {
		key.WireAPI = "responses"
	}
	if key.RequiresOpenAIAuth == nil {
		defaultTrue := true
		key.RequiresOpenAIAuth = &defaultTrue
	}
	if strings.TrimSpace(key.BaseURL) == "" && strings.ToLower(key.Type) == TypeOpenAI {
		key.BaseURL = defaultOpenAIBaseURL
	}
}

func mergeIntegrationDefaults(updated *APIKey, existing APIKey) {
	if updated.Provider == "" {
		updated.Provider = existing.Provider
	}
	if updated.PreferredAuthMethod == "" {
		updated.PreferredAuthMethod = existing.PreferredAuthMethod
	}
	if updated.WireAPI == "" {
		updated.WireAPI = existing.WireAPI
	}
	if updated.EnvKey == "" {
		updated.EnvKey = existing.EnvKey
	}
	if updated.RequiresOpenAIAuth == nil {
		updated.RequiresOpenAIAuth = existing.RequiresOpenAIAuth
	}
	if strings.TrimSpace(updated.RawConfig) == "" {
		updated.RawConfig = existing.RawConfig
	}
	setIntegrationDefaults(updated)
}
