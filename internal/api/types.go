package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/codex-switch/codex-switch/internal/config"
)

// UsageResult 表示额度查询结果
type UsageResult struct {
	Used        float64
	Limit       float64
	DailyTotal  float64
	WeeklyTotal float64
	Monthly     float64
	Raw         map[string]any
}

// UsageClient 定义额度查询客户端接口
type UsageClient interface {
	FetchUsage(key config.APIKey) (*UsageResult, error)
}

// NewClient 根据 Key 类型返回对应客户端
func NewClient(key config.APIKey) (UsageClient, error) {
	switch key.Type {
	case config.TypeOpenAI:
		return NewOpenAIClient(&http.Client{Timeout: 10 * time.Second}), nil
	case config.TypeCRS:
		return NewCRSClient(&http.Client{Timeout: 10 * time.Second}), nil
	default:
		return nil, fmt.Errorf("不支持的 API 类型: %s", key.Type)
	}
}
