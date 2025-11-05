# codex-switch

codex-switch 是一个命令行工具，用于集中管理多个 Codex/OpenAI API Key，并提供配置导入导出以及一键切换能力，帮助团队和个人快速切换工作环境并保障密钥安全。

## 功能亮点
- 多密钥生命周期管理：支持新增、查看、切换、删除密钥，并自动记录激活状态。
- 配置文件统一存放：默认保存在 `~/.codex-switch/config.json`，支持自定义路径覆盖。
- 标签管理：可为密钥打标签，便于按业务或环境进行筛选。
- 配置导入导出：可导入现有 Codex 配置文件或导出为其他环境复用。
- 彩色终端体验：命令结果采用彩色表格与状态提示，关键信息一目了然。

## 环境要求
- Go 1.21 及以上版本
- 可访问 Codex/OpenAI API 的网络环境
- 已准备好至少一个可用的 API Key

## 安装方式

### 通过 go install
```bash
go install github.com/codex-switch/codex-switch/cmd/ckm@latest
```

该方式会直接在 GOPATH/bin 下生成 `ckm` 可执行文件，确保 Go 环境已配置 `GOBIN` 或 `GOPATH/bin` 已加入 `PATH`。

### 本地源码构建
```bash
git clone https://github.com/codex-switch/codex-switch.git
cd codex-switch
go build -o bin/ckm ./...
```

构建完成后，将 `bin/ckm` 添加到 `PATH`，或直接在仓库根目录运行。

## 快速开始
```bash
# 1. 添加一个新的 API Key，并导入 Codex 原始配置
ckm add \
  --name "生产环境 Key" \
  --key "sk-xxxx" \
  --config-file ~/.config/codex/config.toml

# 2. 查看当前所有密钥
ckm list

# 3. 切换当前激活密钥
ckm switch <密钥ID>

# 4. 查看当前密钥详情
ckm show --id <密钥ID>
```

首次运行会自动在用户目录创建 `~/.codex-switch` 文件夹，并持久化所有操作结果。若希望使用自定义配置文件，可通过 `ckm --config /path/to/config.json` 指定加载位置。

## 常用命令速查
| 命令 | 作用 |
| ---- | ---- |
| `ckm add` | 添加新的 API Key，并导入对应配置文件内容 |
| `ckm list` | 以表格形式列出所有已管理的密钥 |
| `ckm switch <id>` | 将指定密钥设置为当前激活密钥 |
| `ckm show --id <id>` | 查看单个密钥的详细信息 |
| `ckm remove <id>` | 删除不再使用的密钥记录 |
| `ckm export --format json` | 导出全部密钥配置，便于备份或迁移 |
| `ckm import --file <path>` | 从已有备份中恢复密钥信息 |
| `ckm remote push` / `pull` | 推送或拉取远端备份（当前基于 Backblaze B2） |
| `ckm update --id <id>` | 更新指定密钥的名称、标签或配置文件 |

运行任意命令时可附加 `-h/--help` 获取详细参数说明。

## 配置与安全
- 所有配置默认为 JSON 格式存放在 `~/.codex-switch/config.json`，文件权限将自动设置为 `0600`，避免敏感信息泄露。
- API Key 在输出时会自动脱敏，仅在必要场景下展示完整值。
- 可通过 `CKM_CONFIG` 环境变量或 `--config` 参数覆盖配置文件路径，方便在 CI 或多账户环境中使用。

## 贡献指南
1. Fork 本仓库并创建功能分支。
2. 确保通过 `go test ./...` 与静态检查。
3. 提交前运行 `golangci-lint run` 保持代码风格一致。
4. 在 Pull Request 中说明变更动机、影响范围以及验证方式。

欢迎通过 Issue 或 PR 提交改进建议，帮助我们共同完善 codex-switch。

更多细节请参见 `CONTRIBUTING.md` 与 `CODE_OF_CONDUCT.md`。

## 发布脚本

项目提供 `scripts/release.sh` 便于本地快速发布：

```bash
# 假设已登录 GitHub CLI，并完成 git tag
./scripts/release.sh v0.1.0 CHANGELOG.md
```

脚本会在 `dist/` 目录生成 Linux/amd64 可执行文件、压缩包与 SHA256 校验文件，并自动创建/更新同名 GitHub Release。需要预先安装 GitHub CLI (`gh`) 并完成 `gh auth login`，同时确保本机具备 `sha256sum`（macOS 可通过 `brew install coreutils` 获取 `gsha256sum`）。若未提供发布说明文件，将使用默认说明。

## 延伸阅读

- `docs/设计方案.md`：包含完整的架构设计与实现细节。

## 许可证

本项目采用 MIT License，详见仓库中的 `LICENSE` 文件。
