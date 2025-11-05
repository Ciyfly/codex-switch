# 贡献指南

感谢你对 codex-switch 的关注！为了让协作更加顺畅，请在提交 Issue 或 Pull Request 前阅读以下说明。

## 环境准备
- Go 1.21 及以上版本
- 已配置 `golangci-lint`（可选，用于静态检查）
- 建议安装 GitHub CLI (`gh`) 以便测试发布流程

克隆仓库后执行：

```bash
go mod download
go test ./...
```

## 分支与提交流程
1. 从 `master` 创建功能分支，例如 `feature/add-command-foo`。
2. 开发过程中保持提交语义化，格式建议为 `type: 描述`（如 `feat: 支持批量导入`）。
3. 完成后运行以下命令确保状态正常：
   ```bash
   go test ./...
   golangci-lint run   # 若已安装
   ```
4. 提交 Pull Request 时，请描述：
   - 改动动机与目标
   - 主要变更点
   - 验证方式（测试、人工验证等）

## Issue 指南
- **Bug**：提供复现步骤、期望行为与实际行为，若可能附加日志或截图。
- **功能需求**：说明使用场景、期望收益与可能的替代方案。

## 发布流程
- 本地可通过 `scripts/release.sh <tag>` 进行验收，与 GitHub Actions 的流程保持一致。
- 发布前请确保更新 `CHANGELOG`（若存在）并同步 `README` 中的版本信息。

欢迎提交改进建议或文档更新，感谢你的贡献！
