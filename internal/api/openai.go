package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/codex-switch/codex-switch/internal/config"
)

// OpenAIClient 调用 OpenAI 额度接口
type OpenAIClient struct {
	httpClient *http.Client
}

// NewOpenAIClient 构造函数
func NewOpenAIClient(client *http.Client) *OpenAIClient {
	return &OpenAIClient{httpClient: client}
}

// FetchUsage 调用 OpenAI Usage API
func (c *OpenAIClient) FetchUsage(key config.APIKey) (*UsageResult, error) {
	if c.httpClient == nil {
		return nil, errors.New("HTTP 客户端未初始化")
	}

	endpoint := strings.TrimRight(key.BaseURL, "/") + "/usage"
	today := time.Now().Format("2006-01-02")
	req, err := http.NewRequest(http.MethodGet, endpoint+"?date="+today, nil)
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
		return nil, fmt.Errorf("OpenAI API 响应错误: %s", strings.TrimSpace(string(body)))
	}

	var payload openAIUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	used := payload.TotalUsage
	daily := payload.SumRecent(1)
	weekly := payload.SumRecent(7)
	monthly := payload.SumRecent(30)

	result := &UsageResult{
		Used:        used,
		Limit:       key.QuotaLimit,
		DailyTotal:  daily,
		WeeklyTotal: weekly,
		Monthly:     monthly,
		Raw:         map[string]any{"response": payload},
	}
	return result, nil
}

type openAIUsageResponse struct {
	Object      string             `json:"object"`
	DailyCosts  []openAIDailyCost  `json:"daily_costs"`
	TotalUsage  float64            `json:"total_usage"`
	Aggregation map[string]float64 `json:"aggregation"`
}

type openAIDailyCost struct {
	Timestamp int64            `json:"timestamp"`
	LineItems []openAILineItem `json:"line_items"`
}

type openAILineItem struct {
	Name string  `json:"name"`
	Cost float64 `json:"cost"`
}

// SumRecent 计算最近 n 天的消费
func (r openAIUsageResponse) SumRecent(days int) float64 {
	if days <= 0 {
		return 0
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	total := 0.0
	for _, dc := range r.DailyCosts {
		ts := time.Unix(dc.Timestamp, 0)
		if ts.Before(cutoff) {
			continue
		}
		for _, li := range dc.LineItems {
			total += li.Cost
		}
	}
	return total
}
