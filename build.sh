#!/usr/bin/env bash
# 编译 codex-switch 的辅助脚本，包含彩色日志输出，便于快速定位执行状态。
set -euo pipefail

GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RED="\033[1;31m"
BLUE="\033[1;34m"
RESET="\033[0m"

log_info() {
  printf "${BLUE}[信息]${RESET} %s\n" "$1"
}

log_warn() {
  printf "${YELLOW}[提示]${RESET} %s\n" "$1"
}

log_success() {
  printf "${GREEN}[成功]${RESET} %s\n" "$1"
}

log_error() {
  printf "${RED}[错误]${RESET} %s\n" "$1"
}

trap 'log_error "编译过程中发生错误"' ERR

if ! command -v go >/dev/null 2>&1; then
  log_error "未检测到 Go 工具链，请先安装 Go 1.20+"
  exit 1
fi

log_info "当前 Go 版本: $(go version)"
log_info "开始编译 codex-switch..."

mkdir -p bin

log_info "执行 go build ./...，检查所有包是否可编译"
go build ./...

log_info "生成主程序二进制 bin/ckm"
go build -o bin/ckm ./cmd/ckm

log_success "编译完成，二进制已输出至 bin/ckm"

log_warn "如需交叉编译，可通过设置 GOOS/GOARCH 后再运行本脚本"
