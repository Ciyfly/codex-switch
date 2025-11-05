package config

import (
	"testing"
	"time"
)

// TestManagerAddAndActive 验证添加 Key 并自动激活的逻辑
func TestManagerAddAndActive(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewManager(storage)

	if _, err := manager.Load(); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	key := APIKey{
		Name:       "主力账号",
		APIKey:     "sk-test",
		BaseURL:    "https://api.openai.com/v1",
		Type:       TypeOpenAI,
		QuotaType:  QuotaMonthly,
		QuotaLimit: 100,
		Active:     true,
	}

	created, err := manager.AddKey(key)
	if err != nil {
		t.Fatalf("添加 Key 失败: %v", err)
	}

	if created.ID == "" {
		t.Fatalf("期待生成 ID")
	}

	active, err := manager.ActiveKey()
	if err != nil {
		t.Fatalf("获取激活 Key 失败: %v", err)
	}

	if active.ID != created.ID {
		t.Fatalf("激活 Key 不匹配，got=%s want=%s", active.ID, created.ID)
	}
}

// TestManagerUpdate 验证更新逻辑
func TestManagerUpdate(t *testing.T) {
	storage := NewMemoryStorage()
	manager := NewManager(storage)
	if _, err := manager.Load(); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	key := APIKey{
		Name:       "测试账号",
		APIKey:     "sk-1",
		BaseURL:    "https://example.com",
		Type:       TypeCRS,
		QuotaType:  QuotaUnlimited,
		QuotaLimit: 0,
	}

	added, err := manager.AddKey(key)
	if err != nil {
		t.Fatalf("添加 Key 失败: %v", err)
	}

	added.Description = "更新描述"
	added.QuotaUsed = 42
	if err := manager.UpdateKey(added); err != nil {
		t.Fatalf("更新 Key 失败: %v", err)
	}

	got, err := manager.GetKey(added.ID)
	if err != nil {
		t.Fatalf("读取 Key 失败: %v", err)
	}
	if got.Description != "更新描述" {
		t.Fatalf("描述未更新: %s", got.Description)
	}
	if got.QuotaUsed != 42 {
		t.Fatalf("额度未更新: %f", got.QuotaUsed)
	}
}

// TestCalculateRemaining 验证额度计算
func TestCalculateRemaining(t *testing.T) {
	now := time.Now()
	key := APIKey{
		QuotaLimit:  100,
		QuotaUsed:   40,
		QuotaType:   QuotaMonthly,
		LastChecked: now.AddDate(0, -1, 0),
	}

	remain := CalculateRemaining(key)
	if remain != 100 {
		t.Fatalf("月度额度应在新周期重置: %f", remain)
	}

	key.QuotaType = QuotaDaily
	key.LastChecked = now
	key.QuotaUsed = 70
	remain = CalculateRemaining(key)
	if remain != 30 {
		t.Fatalf("剩余额度计算错误: %f", remain)
	}

	key.QuotaType = QuotaYearly
	key.LastChecked = now.AddDate(-1, 0, 0)
	key.QuotaUsed = 10
	remain = CalculateRemaining(key)
	if remain != 100 {
		t.Fatalf("年度额度应在新周期重置: %f", remain)
	}
}
