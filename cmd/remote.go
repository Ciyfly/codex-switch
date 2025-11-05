package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/logging"
	"github.com/codex-switch/codex-switch/internal/remote"
	b2 "github.com/codex-switch/codex-switch/internal/remote/b2"

	"github.com/spf13/cobra"
)

var (
	remoteKeyID         string
	remoteAppKey        string
	remoteBucketName    string
	remoteInitProfile   string
	remotePushProfile   string
	remotePullProfile   string
	remoteDeleteProfile string
)

func init() {
	remoteCmd := &cobra.Command{
		Use:   "remote",
		Short: "远程配置同步管理",
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "配置 Backblaze B2 同步",
		RunE:  runRemoteInit,
	}
	initCmd.Flags().StringVar(&remoteKeyID, "key-id", "", "Backblaze B2 Key ID")
	initCmd.Flags().StringVar(&remoteAppKey, "app-key", "", "Backblaze B2 Application Key")
	initCmd.Flags().StringVar(&remoteBucketName, "bucket", "", "Backblaze B2 存储桶名称")
	initCmd.Flags().StringVar(&remoteInitProfile, "profile", "default", "远程配置档案名，用于区分不同机器/环境")
	initCmd.Flags().StringVar(&remoteInitProfile, "storage-key", "default", "(已弃用) 远程存储标识")
	_ = initCmd.Flags().MarkHidden("storage-key")

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "将本地 Key 列表上传到远程存储",
		RunE:  runRemotePush,
	}
	pushCmd.Flags().StringVar(&remotePushProfile, "profile", "", "指定要上传的配置档案名")
	pushCmd.Flags().StringVar(&remotePushProfile, "storage-key", "", "(已弃用) 存储标识")
	_ = pushCmd.Flags().MarkHidden("storage-key")

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "从远程存储拉取最新 Key 列表",
		RunE:  runRemotePull,
	}
	pullCmd.Flags().StringVar(&remotePullProfile, "profile", "", "指定要拉取的配置档案名")
	pullCmd.Flags().StringVar(&remotePullProfile, "storage-key", "", "(已弃用) 存储标识")
	_ = pullCmd.Flags().MarkHidden("storage-key")

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "删除远程快照并清理本地备份",
		RunE:  runRemoteDelete,
	}
	deleteCmd.Flags().StringVar(&remoteDeleteProfile, "profile", "", "指定要删除的配置档案名")
	deleteCmd.Flags().StringVar(&remoteDeleteProfile, "storage-key", "", "(已弃用) 存储标识")
	_ = deleteCmd.Flags().MarkHidden("storage-key")

	remoteCmd.AddCommand(initCmd, pushCmd, pullCmd, deleteCmd)
	RootCommand().AddCommand(remoteCmd)
}

// runRemoteInit 负责校验凭据并写入远程存储配置。
func runRemoteInit(cmd *cobra.Command, _ []string) error {
	if strings.TrimSpace(remoteKeyID) == "" || strings.TrimSpace(remoteAppKey) == "" {
		return errors.New("必须提供 key-id 与 app-key")
	}
	if strings.TrimSpace(remoteBucketName) == "" {
		return errors.New("必须提供 bucket 名称")
	}

	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	cfg, err := manager.Config()
	if err != nil {
		return err
	}

	settings := cfg.Remote
	if settings == nil {
		return errors.New("配置缺失远程字段，请重新初始化配置")
	}
	keyID := strings.TrimSpace(remoteKeyID)
	if keyID == "" {
		keyID = strings.TrimSpace(settings.KeyID)
	}
	appKey := strings.TrimSpace(remoteAppKey)
	if appKey == "" {
		appKey = strings.TrimSpace(settings.ApplicationKey)
	}
	if keyID == "" || appKey == "" {
		return errors.New("必须提供 key-id 与 app-key")
	}

	settings.Provider = "b2"
	settings.KeyID = keyID
	settings.ApplicationKey = appKey
	bucket := strings.TrimSpace(remoteBucketName)
	if bucket == "" {
		bucket = strings.TrimSpace(settings.BucketName)
	}
	if bucket == "" {
		return errors.New("必须提供 bucket 名称")
	}

	settings.BucketName = bucket
	profile := normalizeProfile(remoteInitProfile, "default")
	settings.ObjectKey = profile
	settings.Enabled = true

	client, err := b2.NewClient(settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	if err := client.Prepare(ctx); err != nil {
		return err
	}

	// 准备完成后写回配置文件，包含 bucketID 与同步 token。
	if err := manager.ReplaceConfig(cfg); err != nil {
		return err
	}
	if err := manager.Save(); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ 已完成远程配置，目标桶: %s\n", settings.BucketName)
	fmt.Fprintf(cmd.OutOrStdout(), "接下来可执行: ckm remote push --profile %s\n", profile)
	fmt.Fprintf(cmd.OutOrStdout(), "其他机器执行: ckm remote pull --profile %s\n", profile)
	logging.Infof("初始化远程同步: bucket=%s profile=%s", settings.BucketName, profile)
	return nil
}

// runRemotePush 将当前配置快照保存为 JSON 并同步到 B2。
func runRemotePush(cmd *cobra.Command, _ []string) error {
	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	cfg, err := manager.Config()
	if err != nil {
		return err
	}

	settings := cfg.Remote
	if settings == nil || !settings.Enabled {
		return errors.New("未配置远程同步，请先执行 ckm remote init")
	}

	profile := normalizeProfile(remotePushProfile, settings.ObjectKey)
	if profile == "" {
		return errors.New("未指定有效的配置档案名")
	}

	objectName := buildRemoteObjectName(settings, profile)

	snapshot := remote.BuildSnapshot(cfg)
	data, err := snapshot.Marshal()
	if err != nil {
		return err
	}

	localPath := buildSnapshotPath(manager.ConfigPath(), profile)
	if err := remote.SaveSnapshotFile(localPath, snapshot); err != nil {
		return err
	}

	client, err := b2.NewClient(settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
	defer cancel()

	if err := client.Prepare(ctx); err != nil {
		return err
	}
	if err := client.Upload(ctx, objectName, data); err != nil {
		return err
	}

	settings.ObjectKey = profile
	settings.Enabled = true
	settings.LastSync = time.Now().UTC()

	if err := manager.ReplaceConfig(cfg); err != nil {
		return err
	}
	if err := manager.Save(); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ 已上传快照至 B2 对象: %s\n", objectName)
	fmt.Fprintf(cmd.OutOrStdout(), "本地快照路径: %s\n", localPath)
	logging.Infof("推送远程快照: object=%s profile=%s", objectName, profile)
	return nil
}

// runRemotePull 下载远端快照并覆盖本地配置，同时生成备份。
func runRemotePull(cmd *cobra.Command, _ []string) error {
	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	cfg, err := manager.Config()
	if err != nil {
		return err
	}

	settings := cfg.Remote
	if settings == nil || !settings.Enabled {
		return errors.New("未配置远程同步，请先执行 ckm remote init")
	}

	profile := normalizeProfile(remotePullProfile, settings.ObjectKey)
	if profile == "" {
		return errors.New("未指定有效的配置档案名")
	}

	objectName := buildRemoteObjectName(settings, profile)

	client, err := b2.NewClient(settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
	defer cancel()

	if err := client.Prepare(ctx); err != nil {
		return err
	}
	data, err := client.Download(ctx, objectName)
	if err != nil {
		return err
	}

	snap, err := remote.UnmarshalSnapshot(data)
	if err != nil {
		return err
	}

	localPath := buildSnapshotPath(manager.ConfigPath(), profile)
	if err := remote.SaveSnapshotFile(localPath, snap); err != nil {
		return err
	}

	cfg.Keys = snap.Keys
	cfg.ActiveKeyID = snap.ActiveKeyID
	settings.ObjectKey = profile
	settings.Enabled = true
	settings.LastSync = time.Now().UTC()

	if err := manager.ReplaceConfig(cfg); err != nil {
		return err
	}
	if err := manager.Save(); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ 已从 B2 拉取快照并更新本地配置\n")
	fmt.Fprintf(cmd.OutOrStdout(), "本地快照: %s\n", localPath)
	logging.Infof("拉取远程快照: object=%s profile=%s", objectName, profile)
	return nil
}

// runRemoteDelete 删除远程 B2 快照并清理本地备份。
func runRemoteDelete(cmd *cobra.Command, _ []string) error {
	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	cfg, err := manager.Config()
	if err != nil {
		return err
	}

	settings := cfg.Remote
	if settings == nil || !settings.Enabled {
		return errors.New("当前未启用远程同步，无需删除")
	}

	profile := normalizeProfile(remoteDeleteProfile, settings.ObjectKey)
	if profile == "" {
		return errors.New("未指定有效的配置档案名")
	}

	objectName := buildRemoteObjectName(settings, profile)

	client, err := b2.NewClient(settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	if err := client.Prepare(ctx); err != nil {
		return err
	}

	if err := client.Delete(ctx, objectName); err != nil {
		return err
	}

	localPath := buildSnapshotPath(manager.ConfigPath(), profile)
	removeLocalSnapshot(localPath)

	if settings.ObjectKey == profile {
		settings.Enabled = false
		settings.ObjectKey = ""
		settings.LastSync = time.Time{}
	}

	if err := manager.ReplaceConfig(cfg); err != nil {
		return err
	}
	if err := manager.Save(); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ 已删除远程快照并清理本地备份\n")
	logging.Infof("删除远程快照: object=%s profile=%s", objectName, profile)
	return nil
}

// normalizeProfile 统一 profile 的命名，过滤非法字符。
func normalizeProfile(input string, fallback string) string {
	candidate := strings.TrimSpace(input)
	if candidate == "" {
		candidate = fallback
	}
	if candidate == "" {
		candidate = "default"
	}
	var builder strings.Builder
	for _, r := range strings.ToLower(candidate) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			continue
		}
		switch r {
		case '-', '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		result = "default"
	}
	return result
}

// buildRemoteObjectName 根据同步令牌与 profile 生成远端对象名。
func buildRemoteObjectName(settings *config.RemoteSettings, profile string) string {
	return fmt.Sprintf("%s.json", profile)
}

// buildSnapshotPath 返回本地快照的存储路径，便于审计与备份。
func buildSnapshotPath(cfgPath string, profile string) string {
	base := filepath.Dir(cfgPath)
	dir := filepath.Join(base, "snapshots")
	return filepath.Join(dir, fmt.Sprintf("%s.json", profile))
}

func removeLocalSnapshot(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		logging.Warnf("删除本地快照失败: %v", err)
	}
}
