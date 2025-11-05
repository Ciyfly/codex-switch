package config

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// RemoteSettings 描述远程同步所需的完整配置
//
// 开发者可以通过 remote push/pull 将本地 Key 列表推送到
// Backblaze B2 等对象存储；该结构保存所需的鉴权信息、
// 对象定位以及同步辅助元数据。
type RemoteSettings struct {
	Provider       string    `json:"provider"`
	BucketName     string    `json:"bucket_name"`
	BucketID       string    `json:"bucket_id"`
	ObjectKey      string    `json:"object_key"`
	KeyID          string    `json:"key_id"`
	ApplicationKey string    `json:"application_key"`
	APIURL         string    `json:"api_url,omitempty"`
	DownloadURL    string    `json:"download_url,omitempty"`
	SyncToken      string    `json:"sync_token"`
	LastSync       time.Time `json:"last_sync,omitempty"`
	Enabled        bool      `json:"enabled"`
}

// normalizeRemoteSettings 确保配置结构中包含有效的远程设置占位。
//
// 若配置文件中尚未初始化远程字段，则创建默认实例并自动生成
// 用于数据加密/鉴权的 SyncToken，避免后续逻辑访问空指针。
// 返回值表示是否修改了配置内容，便于调用方决定是否持久化。
func normalizeRemoteSettings(cfg *Config) bool {
	changed := false
	if cfg.Remote == nil {
		cfg.Remote = &RemoteSettings{}
		changed = true
	}
	if cfg.Remote.SyncToken == "" {
		cfg.Remote.SyncToken = generateSyncToken()
		changed = true
	}
	return changed
}

// generateSyncToken 生成 32 字节随机数并使用 Base64 编码。
func generateSyncToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return base64.RawURLEncoding.EncodeToString([]byte("codex-sync-fallback"))
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
