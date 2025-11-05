package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Storage 抽象化配置持久化行为，便于测试替换
type Storage interface {
	Load() (*Config, error)
	Save(*Config) error
	Path() string
}

// FileStorage 基于本地文件系统的实现
type FileStorage struct {
	path string
}

// NewFileStorage 创建文件存储实例，自动定位默认目录
func NewFileStorage(path string) (*FileStorage, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		base := filepath.Join(home, ".codex-manager")
		path = filepath.Join(base, "config.json")
	}

	return &FileStorage{path: path}, nil
}

// Path 返回配置文件路径
func (f *FileStorage) Path() string {
	return f.path
}

// Load 从文件读取配置，若文件不存在则返回默认结构
func (f *FileStorage) Load() (*Config, error) {
	if err := ensureConfigDir(f.path); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := &Config{
			Version:     defaultVersion,
			ActiveKeyID: "",
			Keys:        []APIKey{},
			LastUpdated: time.Now().UTC(),
		}
		if err := f.Save(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &Config{Version: defaultVersion, Keys: []APIKey{}}, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Keys == nil {
		cfg.Keys = []APIKey{}
	}
	return &cfg, nil
}

// Save 将配置写入磁盘，并严格控制权限
func (f *FileStorage) Save(cfg *Config) error {
	if err := ensureConfigDir(f.path); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := f.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}

	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, f.path); err != nil {
		return err
	}

	return os.Chmod(f.path, 0o600)
}

// ensureConfigDir 确保目录存在并具有安全权限
func ensureConfigDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}

// MemoryStorage 仅用于测试场景的内存实现
type MemoryStorage struct {
	cfg *Config
}

// NewMemoryStorage 创建内存存储，方便单元测试隔离文件系统
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{cfg: &Config{Version: defaultVersion, Keys: []APIKey{}}}
}

// Load 返回内存中的配置副本
func (m *MemoryStorage) Load() (*Config, error) {
	copy := *m.cfg
	copy.Keys = append([]APIKey(nil), m.cfg.Keys...)
	return &copy, nil
}

// Save 更新内存中的配置内容
func (m *MemoryStorage) Save(cfg *Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	// 重新反序列化，确保深拷贝
	var copy Config
	if err := json.Unmarshal(data, &copy); err != nil {
		return err
	}
	m.cfg = &copy
	return nil
}

// Path 返回逻辑路径，用于调试
func (m *MemoryStorage) Path() string { return "memory://config" }

var _ Storage = (*FileStorage)(nil)
var _ Storage = (*MemoryStorage)(nil)
