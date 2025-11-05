#!/usr/bin/env bash
# 该脚本用于快速发布 codex-switch 版本：
# 1. 校验环境与依赖；
# 2. 运行一次交叉编译确保二进制可用；
# 3. 创建 Git tag 并推送触发 GitHub Actions；
# 4. 可选附带发布说明（文件或单行文本）。

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "用法：$0 <版本号或标签> [发布说明]" >&2
  echo "发布说明可以是文件路径或直接的一行文本" >&2
  exit 1
fi

INPUT_TAG="$1"
RAW_NOTES="${2-}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"
BINARY_NAME="ckm"
TEMP_NOTES=""

# 统一标签格式
if [[ "${INPUT_TAG}" == v* ]]; then
  TAG="${INPUT_TAG}"
else
  TAG="v${INPUT_TAG}"
fi

declare -a NOTES_ARGS=()

log() {
  printf "\033[1;34m[发布]\033[0m %s\n" "$1"
}

warn() {
  printf "\033[1;33m[提示]\033[0m %s\n" "$1"
}

error() {
  printf "\033[1;31m[错误]\033[0m %s\n" "$1" >&2
  exit 1
}

ensure_cleanup() {
  if [[ -n "${TEMP_NOTES}" && -f "${TEMP_NOTES}" ]]; then
    rm -f "${TEMP_NOTES}"
  fi
}
trap ensure_cleanup EXIT

if ! command -v go >/dev/null 2>&1; then
  error "未检测到 Go，请先安装 Go 1.21+ 工具链。"
fi

if [[ -n "${RAW_NOTES}" ]]; then
  if [[ -f "${RAW_NOTES}" ]]; then
    NOTES_ARGS=( -F "${RAW_NOTES}" )
  else
    TEMP_NOTES="$(mktemp)"
    printf "%s\n" "${RAW_NOTES}" > "${TEMP_NOTES}"
    NOTES_ARGS=( -F "${TEMP_NOTES}" )
  fi
fi

if [[ -n "$(git status --porcelain)" ]]; then
  error "当前工作区存在未提交的修改，请先提交或清理后再执行。"
fi

if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
  error "标签 ${TAG} 已存在，如需重新发布请先删除该标签。"
fi

log "执行交叉编译验证 (linux/amd64)..."
mkdir -p "${DIST_DIR}"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIST_DIR}/${BINARY_NAME}" "${PROJECT_ROOT}/cmd/ckm"
rm -f "${DIST_DIR}/${BINARY_NAME}"

log "创建 Git 标签 ${TAG}..."
if [[ ${#NOTES_ARGS[@]} -gt 0 ]]; then
  git tag -a "${TAG}" "${NOTES_ARGS[@]}"
else
  git tag -a "${TAG}" -m "Release ${TAG}"
fi

log "推送标签到远程..."
git push origin "${TAG}"

log "完成：GitHub Actions 将基于标签 ${TAG} 自动构建并发布资产。"

if command -v gh >/dev/null 2>&1; then
  if gh auth status >/dev/null 2>&1; then
    warn "已检测到 GitHub CLI，可在工作流完成后执行 'gh release view ${TAG} --web' 检查发布内容。"
  else
    warn "检测到 GitHub CLI 但尚未登录，可运行 'gh auth login' 以便后续管理 Release。"
  fi
else
  warn "未安装 GitHub CLI，流程仍已完成。GitHub Actions 将创建 Release 并上传制品。"
fi

log "发布流程结束。"
