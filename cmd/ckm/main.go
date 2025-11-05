package main

import (
	"log"

	"github.com/codex-switch/codex-switch/cmd"
	"github.com/codex-switch/codex-switch/internal/logging"
)

// main 入口函数，初始化并执行根命令
func main() {
	defer logging.Close()
	if err := cmd.Execute(); err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
}
