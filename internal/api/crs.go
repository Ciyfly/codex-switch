package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/codex-switch/codex-switch/internal/config"
)

// CRSClient 用于自定义 CRS API 查询额度
type CRSClient struct {
	httpClient *http.Client
}

// NewCRSClient 构造函数
func NewCRSClient(client *http.Client) *CRSClient {
	return &CRSClient{httpClient: client}
}

// FetchUsage 根据配置执行额度查询
func (c *CRSClient) FetchUsage(key config.APIKey) (*UsageResult, error) {
	if c.httpClient == nil {
		c.httpClient = &http.Client{}
	}
	if key.ManualTrack {
		return &UsageResult{
			Used:        key.QuotaUsed,
			Limit:       key.QuotaLimit,
			DailyTotal:  0,
			WeeklyTotal: 0,
			Monthly:     key.QuotaUsed,
		}, nil
	}

	endpoint := strings.TrimRight(key.BaseURL, "/") + "/v1/dashboard/billing/usage"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CRS API 响应错误: %s", strings.TrimSpace(string(body)))
	}

	var payload crsUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return &UsageResult{
		Used:        payload.TotalUsage,
		Limit:       key.QuotaLimit,
		DailyTotal:  payload.TodayUsage,
		WeeklyTotal: payload.WeekUsage,
		Monthly:     payload.MonthUsage,
		Raw:         map[string]any{"response": payload},
	}, nil
}

type crsUsageResponse struct {
	TotalUsage float64 `json:"total_usage"`
	TodayUsage float64 `json:"today_usage"`
	WeekUsage  float64 `json:"week_usage"`
	MonthUsage float64 `json:"month_usage"`
}
