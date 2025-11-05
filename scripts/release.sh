#!/usr/bin/env bash
# 该脚本用于本地快速发布 Codex Key Manager 版本：
# 1. 校验环境变量与工具依赖；
# 2. 构建 Linux/amd64 无 CGO 可执行文件；
# 3. 打包压缩并生成 SHA256 校验文件；
# 4. 使用 GitHub CLI 创建/更新 Release 并上传制品。

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "用法：$0 <tag> [发布说明文件]" >&2
  exit 1
fi

TAG="$1"
NOTES_FILE="${2:-}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"
BINARY_NAME="ckm"
ARCHIVE_NAME="ckm-linux-amd64.tar.gz"
CHECKSUM_FILE="ckm-linux-amd64.sha256"

log() {
  printf "\033[1;34m[发布]\033[0m %s\n" "$1"
}

error() {
  printf "\033[1;31m[错误]\033[0m %s\n" "$1" >&2
  exit 1
}

if ! command -v go >/dev/null 2>&1; then
  error "未检测到 Go，请先安装 Go 1.21+ 工具链。"
fi

if ! command -v gh >/dev/null 2>&1; then
  error "未检测到 GitHub CLI (gh)，请安装后再执行。"
fi

if [[ -n "$NOTES_FILE" && ! -f "$NOTES_FILE" ]]; then
  error "发布说明文件不存在：${NOTES_FILE}"
fi

log "开始构建 Linux/amd64 可执行文件..."
mkdir -p "${DIST_DIR}"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIST_DIR}/${BINARY_NAME}" "${PROJECT_ROOT}/cmd/ckm"

log "生成压缩包与校验文件..."
tar -czf "${DIST_DIR}/${ARCHIVE_NAME}" -C "${DIST_DIR}" "${BINARY_NAME}"
(cd "${DIST_DIR}" && sha256sum "${ARCHIVE_NAME}" > "${CHECKSUM_FILE}")

log "准备提交到 GitHub Release..."
if ! gh auth status >/dev/null 2>&1; then
  error "GitHub CLI 未完成登录，请执行 gh auth login 后再试。"
fi

GH_ARGS=(
  "${TAG}"
  "${DIST_DIR}/${ARCHIVE_NAME}"
  "${DIST_DIR}/${CHECKSUM_FILE}"
  "--title" "${TAG}"
)

if [[ -n "$NOTES_FILE" ]]; then
  GH_ARGS+=("--notes-file" "$NOTES_FILE")
else
  GH_ARGS+=("--notes" "自动发布 ${TAG}")
fi

if gh release view "${TAG}" >/dev/null 2>&1; then
  log "检测到 Release 已存在，将执行更新。"
  gh release upload "${TAG}" "${DIST_DIR}/${ARCHIVE_NAME}" "${DIST_DIR}/${CHECKSUM_FILE}" --clobber
  gh release edit "${GH_ARGS[@]:2}"
else
  log "创建新的 Release。"
  gh release create "${GH_ARGS[@]}"
fi

log "发布完成，制品已上传。"
