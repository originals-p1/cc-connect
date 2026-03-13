# 贡献者指南 | CONTRIBUTING

感谢你对 cc-connect 项目的贡献！本文档介绍如何贡献代码、报告问题和参与社区。

---

## 表格目录

| 中文 | English |
|------|---------|
| [代码贡献](#代码贡献) | [Code Contribution](#code-contribution) |
| [问题报告](#问题报告) | [Issue Reporting](#issue-reporting) |
| [文档贡献](#文档贡献) | [Documentation Contribution](#documentation-contribution) |
| [社区参与](#社区参与) | [Community Participation](#community-participation) |

---

## 代码贡献

### 贡献流程

```
1. Fork 仓库
2. 创建功能分支 (git checkout -b feature/your-feature)
3. 提交更改 (git commit -m 'Add some feature')
4. 推送分支 (git push origin feature/your-feature)
5. Open a Pull Request
```

### 开发环境

#### 前置要求

- Go 1.22+
-_make (GNU Make)
- Git

#### 设置开发环境

```bash
# 1. Fork 仓库并克隆
git clone https://github.com/your-username/cc-connect.git
cd cc-connect

# 2. 安装依赖
go mod download

# 3. 运行测试
make test

# 4. 构建项目
make build

# 5. 运行 lint
make lint
```

### 代码规范

#### Go 格式

```bash
# 格式化代码
gofmt -w $(find . -name "*.go" -not -path "./vendor/*")

# 检查格式
gofmt -l $(find . -name "*.go" -not -path "./vendor/*")
```

#### Linting

```bash
# 运行 linter
make lint

# 安装 linter（如果 Needed）
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### 命名规范

- **包名**: 小写，单数形式（`agent`, `platform`, `core`）
- **类型名**: 大驼峰（`Agent`, `Platform`, `Engine`）
- **变量/函数**: 小驼峰（`agentSession`, `startPlatform`）
- **常量**: 全大写（`DefaultPort`, `MaxRetries`）

#### 注释规范

```go
// Package xxx 描述包的用途
package xxx

// StructName 描述结构体的用途
type StructName struct {
    FieldName string // FieldName 描述字段用途
}

// FuncName 描述函数的用途、参数和返回值
func FuncName(param string) error {
    // 实现
}
```

---

## 问题报告

### 报告 Bug

使用 GitHub Issues 报告 bug。在提交前请检查：

1. [x] 问题在最新版本中仍存在
2. [x] 搜索现有 Issue 避免重复
3. [x] 提供清晰的复现步骤

#### Bug Report 模板

```
**简短描述**
简明描述问题

**复现步骤**
1. xxx
2. xxx
3. xxx

**预期行为**
描述预期结果

**实际行为**
描述实际结果

**环境信息**
- cc-connect version: v1.2.1
- Go version: go1.22.0
- Platform: Linux/macOS/Windows
- Agent: Claude Code/Codex/...

**日志**
```

### 功能请求

使用 GitHub Issues 提出新功能。

#### Feature Request 模板

```
**简短描述**
简明描述功能

**动机**
为什么需要这个功能？解决什么问题？

**实现建议**
如果有的话，描述你设想的实现方式

**替代方案**
描述已考虑的替代方案
```

---

## 文档贡献

### 文档结构

```
docs/
├── AGENTS.md                 # 代理操作指南
├── AI_CONTEXT.md            # AI 上下文
├── VALIDATION.md            # 验证指南
├── TASK_RECIPES.md          # 任务配方
├── PLATFORM_INTEGRATION.md  # 平台接入指南（新增）
├── AGENT_INTEGRATION.md     # Agent 接入指南（新增）
├── SECURITY.md              # 安全指南（新增）
├── feishu.md                # 飞书接入
├── telegram.md              # Telegram 接入
└── ...                      # 其他平台文档
```

### 文档规范

#### Markdown 格式

- 使用 2 空格缩进
- 行长不超过 120 字符
- 使用 emoji 表情增强可读性（可选）

#### 中英文对照

中文文档应包含英文术语对照：

```markdown
| 中文 | English |
|------|---------|
| 配置文件 | Configuration File |
| 会话 | Session |
```

#### 代码示例

```toml
# 配置示例
[[projects]]
name = "my-project"

[projects.agent]
type = "claudecode"
```

```go
// Go 代码示例
func Example() {
    agent := &Agent{}
}
```

### 文档检查

```bash
# 检查文档链接
find docs -name "*.md" -exec grep -l "\[.*\](.*)" {} \; | xargs -I {} sh -c 'echo "Checking {}"; grep -oP "\[.*\]\(([^)]+)\)" {} | sed "s/\[.*\](//;s/)//" | while read link; do if [[ ! "$link" =~ ^http ]]; then echo "  $link"; fi; done'

# 检查拼写
pip install codespell
codespell docs/
```

---

## 测试贡献

### 单元测试

#### 测试文件命名

```
source_file.go  →  source_file_test.go
```

#### 测试示例

```go
package mypackage

import "testing"

func TestMyFunction(t *testing.T) {
    // Arrange
    input := "test"
    
    // Act
    result, err := MyFunction(input)
    
    // Assert
    if err != nil {
        t.Errorf("expected no error, got %v", err)
    }
    if result != "expected" {
        t.Errorf("expected 'expected', got '%s'", result)
    }
}

func TestMyFunction_InvalidInput(t *testing.T) {
    // 测试无效输入
    _, err := MyFunction("")
    if err == nil {
        t.Error("expected error for empty input")
    }
}
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定包测试
go test ./agent/claudecode/... -v

# 运行特定测试
go test -run TestMyFunction ./agent/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 社区参与

### 社区渠道

- **Discord**: https://discord.gg/kHpwgaM4kq
- **Telegram**: https://t.me/+odGNDhCjbjdmMmZl
- **GitHub Issues**: https://github.com/chenhg5/cc-connect/issues

### 快速开始任务

标签 `good first issue` 标记了适合新手的入门任务：

- [Good First Issues](https://github.com/chenhg5/cc-connect/labels/good%20first%20issue)

### code Review 流程

1. 提交 PR
2. CI 检查自动运行
3. 维护者 review
4. 根据反馈修改
5. 合并

#### PR 标题格式

```
type: description

# type: feat | fix | chore | docs | test | refactor
```

示例：

```
feat: add support for new agent
fix: handle nil pointer in session
docs: update configuration guide
test: add session tests
```

---

## 许可证

本项目采用 MIT 许可证。贡献者代码将自动按相同条款授权。

见 `LICENSE` 文件了解详情。

---

## 致谢

感谢所有贡献者！

<a href="https://github.com/chenhg5/cc-connect/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=chenhg5/cc-connect" />
</a>

---

**最后更新**: 2026-03-14
