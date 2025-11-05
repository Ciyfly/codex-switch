package b2

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/codex-switch/codex-switch/internal/config"
)

// Client 封装 Backblaze B2 API 的最小实现，负责上传/下载快照。
type Client struct {
	httpClient *http.Client
	settings   *config.RemoteSettings

	mu            sync.Mutex
	accountID     string
	apiURL        string
	downloadURL   string
	authToken     string
	authExpiresAt time.Time
}

var safeKeyPattern = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
var errObjectNotFound = errors.New("b2: object not found")

// NewClient 根据远程配置创建 B2 客户端。
func NewClient(settings *config.RemoteSettings) (*Client, error) {
	if settings == nil {
		return nil, errors.New("远程配置为空")
	}
	if strings.TrimSpace(settings.KeyID) == "" || strings.TrimSpace(settings.ApplicationKey) == "" {
		return nil, errors.New("缺少 B2 Key 信息")
	}
	if strings.TrimSpace(settings.BucketName) == "" {
		return nil, errors.New("缺少 B2 存储桶名称")
	}
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		settings:   settings,
	}, nil
}

// Upload 将指定对象上传到 B2。
func (c *Client) Upload(ctx context.Context, objectKey string, data []byte) error {
	if len(data) == 0 {
		return errors.New("上传数据为空")
	}
	key := sanitizeObjectKey(objectKey)
	if err := c.ensureAuthorized(ctx); err != nil {
		return err
	}
	if c.settings.BucketID == "" {
		if err := c.fetchBucketID(ctx); err != nil {
			return err
		}
	}
	uploadURL, uploadToken, err := c.getUploadURL(ctx)
	if err != nil {
		return err
	}

	sum := sha1.Sum(data)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", uploadToken)
	req.Header.Set("X-Bz-File-Name", key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bz-Content-Sha1", fmt.Sprintf("%x", sum))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("上传失败: %s", strings.TrimSpace(string(body)))
	}

	c.settings.LastSync = time.Now().UTC()
	return nil
}

// Download 从 B2 拉取对象。
func (c *Client) Download(ctx context.Context, objectKey string) ([]byte, error) {
	key := sanitizeObjectKey(objectKey)
	if err := c.ensureAuthorized(ctx); err != nil {
		return nil, err
	}

	if strings.TrimSpace(c.settings.BucketName) == "" {
		return nil, errors.New("缺少存储桶名称")
	}

	downloadURL := fmt.Sprintf("%s/file/%s/%s", c.downloadURL, url.PathEscape(c.settings.BucketName), key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("下载失败: %s", strings.TrimSpace(string(body)))
	}
	return io.ReadAll(resp.Body)
}

// Delete 删除远端对象，若对象不存在则视为成功。
func (c *Client) Delete(ctx context.Context, objectKey string) error {
	key := sanitizeObjectKey(objectKey)
	if err := c.ensureAuthorized(ctx); err != nil {
		return err
	}
	if c.settings.BucketID == "" {
		if err := c.fetchBucketID(ctx); err != nil {
			return err
		}
	}

	fileID, err := c.findFileID(ctx, key)
	if err != nil {
		if errors.Is(err, errObjectNotFound) {
			return nil
		}
		return err
	}

	endpoint := fmt.Sprintf("%s/b2api/v2/b2_delete_file_version", c.apiURL)
	payload := map[string]string{
		"fileName": key,
		"fileId":   fileID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("删除失败: %s", strings.TrimSpace(string(msg)))
	}
	return nil
}

func (c *Client) ensureAuthorized(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.authToken != "" && time.Until(c.authExpiresAt) > 2*time.Minute {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.settings.KeyID, c.settings.ApplicationKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("授权失败: %s", strings.TrimSpace(string(body)))
	}

	var result struct {
		AccountID          string `json:"accountId"`
		AuthorizationToken string `json:"authorizationToken"`
		APIURL             string `json:"apiUrl"`
		DownloadURL        string `json:"downloadUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.AccountID == "" || result.AuthorizationToken == "" {
		return errors.New("授权响应不完整")
	}

	c.accountID = result.AccountID
	c.authToken = result.AuthorizationToken
	c.apiURL = result.APIURL
	c.downloadURL = result.DownloadURL
	c.authExpiresAt = time.Now().Add(22 * time.Hour)
	return nil
}

func (c *Client) fetchBucketID(ctx context.Context) error {
	c.mu.Lock()
	apiURL := c.apiURL
	authToken := c.authToken
	accountID := c.accountID
	c.mu.Unlock()

	endpoint := fmt.Sprintf("%s/b2api/v2/b2_list_buckets", apiURL)
	payload := map[string]string{
		"accountId":  accountID,
		"bucketName": c.settings.BucketName,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("获取 bucket 失败: %s", strings.TrimSpace(string(msg)))
	}

	var result struct {
		Buckets []struct {
			BucketID   string `json:"bucketId"`
			BucketName string `json:"bucketName"`
		} `json:"buckets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	for _, b := range result.Buckets {
		if strings.EqualFold(b.BucketName, c.settings.BucketName) {
			c.settings.BucketID = b.BucketID
			return nil
		}
	}
	return fmt.Errorf("未找到存储桶 %s", c.settings.BucketName)
}

func (c *Client) getUploadURL(ctx context.Context) (string, string, error) {
	c.mu.Lock()
	apiURL := c.apiURL
	authToken := c.authToken
	bucketID := c.settings.BucketID
	c.mu.Unlock()

	endpoint := fmt.Sprintf("%s/b2api/v2/b2_get_upload_url", apiURL)
	payload := map[string]string{"bucketId": bucketID}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", "", fmt.Errorf("获取上传 URL 失败: %s", strings.TrimSpace(string(msg)))
	}

	var result struct {
		UploadURL          string `json:"uploadUrl"`
		AuthorizationToken string `json:"authorizationToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	if result.UploadURL == "" || result.AuthorizationToken == "" {
		return "", "", errors.New("上传 URL 响应缺失字段")
	}
	return result.UploadURL, result.AuthorizationToken, nil
}

func sanitizeObjectKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		trimmed = "snapshot.json"
	}
	return safeKeyPattern.ReplaceAllString(trimmed, "_")
}

// Prepare 预先完成授权与存储桶校验，适用于初始化流程。
func (c *Client) Prepare(ctx context.Context) error {
	if err := c.ensureAuthorized(ctx); err != nil {
		return err
	}
	if c.settings.BucketID == "" {
		if err := c.fetchBucketID(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) findFileID(ctx context.Context, key string) (string, error) {
	endpoint := fmt.Sprintf("%s/b2api/v2/b2_list_file_names", c.apiURL)
	payload := map[string]any{
		"bucketId":      c.settings.BucketID,
		"startFileName": key,
		"maxFileCount":  1,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("查询对象失败: %s", strings.TrimSpace(string(msg)))
	}

	var result struct {
		Files []struct {
			FileName string `json:"fileName"`
			FileID   string `json:"fileId"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	for _, f := range result.Files {
		if strings.EqualFold(f.FileName, key) {
			return f.FileID, nil
		}
	}
	return "", errObjectNotFound
}
