package cmd

import "testing"

// TestNormalizeTags 确认标签解析逻辑
func TestNormalizeTags(t *testing.T) {
	tags := normalizeTags("prod, main , ,dev")
	if len(tags) != 3 {
		t.Fatalf("期望 3 个标签，实际 %d", len(tags))
	}
	if tags[0] != "prod" || tags[1] != "main" || tags[2] != "dev" {
		t.Fatalf("解析结果不符合预期: %#v", tags)
	}
}
