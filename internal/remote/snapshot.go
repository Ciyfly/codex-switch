package remote

import (
	"encoding/json"
	"errors"
	"github.com/codex-switch/codex-switch/internal/config"
	"os"
	"path/filepath"
	"time"
)

const snapshotSchemaVersion = "1.0"

// Snapshot 描述一次远程同步的完整数据快照。
//
// 该结构会被序列化为 JSON，包含当前激活 Key、所有 Key 列表
// 以及生成时间，保证远端与本地的数据一致性。
type Snapshot struct {
	SchemaVersion string          `json:"schema_version"`
	GeneratedAt   time.Time       `json:"generated_at"`
	ActiveKeyID   string          `json:"active_key_id"`
	Keys          []config.APIKey `json:"keys"`
}

// BuildSnapshot 根据当前配置构建快照，供上传或备份使用。
func BuildSnapshot(cfg *config.Config) *Snapshot {
	if cfg == nil {
		return &Snapshot{SchemaVersion: snapshotSchemaVersion, GeneratedAt: time.Now().UTC()}
	}
	keys := make([]config.APIKey, len(cfg.Keys))
	copy(keys, cfg.Keys)
	return &Snapshot{
		SchemaVersion: snapshotSchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		ActiveKeyID:   cfg.ActiveKeyID,
		Keys:          keys,
	}
}

// Marshal 序列化快照为 JSON，默认使用缩进格式，便于审计与版本管理。
func (s *Snapshot) Marshal() ([]byte, error) {
	if s == nil {
		return nil, errors.New("快照为空")
	}
	return json.MarshalIndent(s, "", "  ")
}

// UnmarshalSnapshot 反序列化 JSON 快照。
func UnmarshalSnapshot(data []byte) (*Snapshot, error) {
	if len(data) == 0 {
		return nil, errors.New("快照数据为空")
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	if snap.SchemaVersion == "" {
		snap.SchemaVersion = snapshotSchemaVersion
	}
	return &snap, nil
}

// SaveSnapshotFile 将快照写入指定路径，权限保持为 0600。
func SaveSnapshotFile(path string, snap *Snapshot) error {
	if snap == nil {
		return errors.New("快照对象为空")
	}
	data, err := snap.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadSnapshotFile 从文件加载快照，常用于离线备份或远程拉取后落地。
func LoadSnapshotFile(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return UnmarshalSnapshot(data)
}
