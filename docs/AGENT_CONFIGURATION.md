# Agent 配置指南 | Agent Configuration Guide

这篇文档已经存在在 INSTALL.md 中，我来创建一份更详细的专用配置指南。

---

## 目录

| 中文 | English |
|------|---------|
| [Agent 支持矩阵](#agent-支持矩阵) | [Agent Support Matrix](#agent-support-matrix) |
| [配置选项](#配置选项) | [Configuration Options](#configuration-options) |
| [权限模式详解](#权限模式详解) | [Permission Modes](#permission-modes) |
| [API Provider 配置](#api-provider-配置) | [API Provider Configuration](#api-provider-configuration) |
| [模型配置](#模型配置) | [Model Configuration](#model-configuration) |

---

## Agent 支持矩阵

| Agent | 权限模式 | Provider 切换 | 模型切换 | 命令发现 | 技能发现 |
|-------|---------|--------------|----------|----------|----------|
| Claude Code | ✅ | ✅ | ✅ | ✅ | ✅ |
| Codex | ✅ | ✅ | ✅ | ❌ | ❌ |
| Cursor Agent | ✅ | ❌ | ❌ | ❌ | ❌ |
| Gemini CLI | ✅ | ❌ | ❌ | ❌ | ❌ |
| Qoder CLI | ✅ | ❌ | ❌ | ❌ | ❌ |
| OpenCode | ✅ | ✅ | ❌ | ✅ | ✅ |
| iFlow CLI | ✅ | ✅ | ❌ | ❌ | ❌ |

---

## 配置选项

### 通用选项

```toml
[projects.agent.options]
work_dir = "/path/to/project"  # 工作目录（必需）
mode = "default"               # 权限模式
# model = "claude-sonnet-4"    # 模型（可选）
# provider = "anthropic"       # 默认 provider（可选）
```

### Agent-specific 选项

#### Claude Code

```toml
[projects.agent.options]
work_dir = "/path/to/project"
mode = "default"
# allowed_tools = ["Read", "Grep", "Glob"]  # 允许的工具列表
```

#### Codex

```toml
[projects.agent.options]
work_dir = "/path/to/project"
mode = "full-auto"
model = "o3"  # 或 o1, o1-mini
```

#### Cursor Agent

```toml
[projects.agent.options]
work_dir = "/path/to/project"
mode = "default"
```

#### Gemini CLI

```toml
[projects.agent.options]
work_dir = "/path/to/project"
mode = "default"
```

#### Qoder CLI

```toml
[projects.agent.options]
work_dir = "/path/to/project"
mode = "default"
model = "auto"  # auto, ultimate, performance, efficient, lite
```

#### OpenCode

```toml
[projects.agent.options]
work_dir = "/path/to/project"
mode = "default"
```

#### iFlow CLI

```toml
[projects.agent.options]
work_dir = "/path/to/project"
mode = "default"
model = "Qwen3-Coder"  # 可选模型
```

---

## 权限模式详解

### Default Mode（默认模式）

**适合场景**: 所有场景，默认推荐

```toml
[projects.agent.options]
mode = "default"
```

**行为**:
- 每次工具调用都要求用户批准
- 最安全的选项
- 适用于所有工作流

### Accept Edits Mode（接受编辑）

**适合场景**: 信任环境，希望提高效率

```toml
[projects.agent.options]
mode = "acceptEdits"  # 或 "edit"
```

**行为**:
- 文件编辑工具自动批准
- 其他工具仍需批准
- 适用于频繁编辑的项目

### Plan Mode（计划模式）

**适合场景**: 代码审查、只读分析

```toml
[projects.agent.options]
mode = "plan"
```

**行为**:
- 只允许读取操作
- 不允许任何修改
- 适用于安全审计

### YOLO Mode（无限制模式）

**⚠️ 警告**: 仅在完全信任的沙箱环境使用

```toml
[projects.agent.options]
mode = "yolo"
```

**行为**:
- 所有工具调用自动批准
- 无需用户干预
- 风险较高

**建议**:
```toml
# 配合 allowed_tools 限制风险
[projects.agent.options]
mode = "yolo"
allowed_tools = ["Read", "Glob", "Grep"]  # 仅允许读取工具
```

---

## API Provider 配置

### 基础配置

```toml
[projects.agent.options]
work_dir = "/path/to/project"
provider = "anthropic"  # 默认 provider

[[projects.agent.providers]]
name = "anthropic"
api_key = "${ANTHROPIC_API_KEY}"

[[projects.agent.providers]]
name = "relay"
api_key = "${RELAY_API_KEY}"
base_url = "https://api.relay.com"
```

### 云服务商配置

#### AWS Bedrock

```toml
[[projects.agent.providers]]
name = "bedrock"
env = {
  CLAUDE_CODE_USE_BEDROCK = "1",
  AWS_PROFILE = "bedrock",
  AWS_REGION = "us-east-1"
}
```

#### Azure OpenAI

```toml
[[projects.agent.providers]]
name = "azure"
env = {
  ANTHROPIC_API_KEY = "${AZURE_API_KEY}",
  ANTHROPIC_BASE_URL = "${AZURE_ENDPOINT}"
}
```

#### 自定义 Relay

```toml
[[projects.agent.providers]]
name = "custom-relay"
api_key = "${CUSTOM_KEY}"
base_url = "https://custom-relay.example.com"
model = "claude-sonnet-4"
```

### Provider 切换

在聊天中使用：

```
/provider              # 查看当前 provider
/provider list         # 列出所有 providers
/provider switch <name> # 切换到指定 provider
/provider add <name> <key> <url> # 添加 provider
```

---

## 模型配置

### Claude Code 模型

```toml
[[projects.agent.providers]]
name = "anthropic"
model = "claude-sonnet-4"  # 默认模型
```

### Codex 模型

```toml
[projects.agent.options]
model = "o3"  # 或 o1, o1-mini
```

### Qoder CLI 模型

```toml
[projects.agent.options]
model = "auto"  # auto, ultimate, performance, efficient, lite
```

### iFlow CLI 模型

```toml
[projects.agent.options]
model = "Qwen3-Coder"
```

---

## 高级配置

### 路由集成（Claude Code Router）

```toml
[projects.agent.options]
work_dir = "/path/to/project"
router_url = "http://127.0.0.1:3456"
router_api_key = "your-secret-key"
```

### 环境变量映射

| Agent | api_key → | base_url → |
|-------|-----------|------------|
| Claude Code | `ANTHROPIC_API_KEY` | `ANTHROPIC_BASE_URL` |
| Codex | `OPENAI_API_KEY` | `OPENAI_BASE_URL` |
| Gemini CLI | `GEMINI_API_KEY` | N/A |
| OpenCode | `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY` 等 | `AZURE_OPENAI_ENDPOINT` / `LOCAL_ENDPOINT` |
| iFlow CLI | `IFLOW_API_KEY` / `IFLOW_apiKey` | `IFLOW_BASE_URL` / `IFLOW_baseUrl` |

### 自定义环境变量

```toml
[[projects.agent.providers]]
name = "custom"
env = {
  CUSTOM_VAR1 = "value1",
  CUSTOM_VAR2 = "value2"
}
```

---

## 最佳实践

### 1. 分离生产与开发配置

```toml
# 生产环境配置
[projects.agent.options]
work_dir = "/production/project"
mode = "default"
provider = "production-provider"

# 开发环境配置（使用不同 provider）
[[projects]]
name = "dev-project"
[projects.agent.options]
work_dir = "/development/project"
mode = "yolo"
provider = "dev-provider"
```

### 2. 使用环境变量

```bash
# .env 文件（添加到 .gitignore）
export ANTHROPIC_API_KEY="sk-xxx"
export OPENAI_API_KEY="sk-xxx"

# 在 config.toml 中引用
[[projects.agent.providers]]
name = "anthropic"
api_key = "${ANTHROPIC_API_KEY}"
```

### 3. 多账号管理

```toml
# 工作账号
[[projects]]
name = "work-project"
[projects.agent.options]
work_dir = "/work/project"
provider = "work-account"

[[projects.agent.providers]]
name = "work-account"
api_key = "${WORK_API_KEY}"

# 个人账号
[[projects]]
name = "personal-project"
[projects.agent.options]
work_dir = "/personal/project"
provider = "personal-account"

[[projects.agent.providers]]
name = "personal-account"
api_key = "${PERSONAL_API_KEY}"
```

---

## 故障排查

### 问题: Provider 切换失败

**检查**:
```bash
# 查看当前 provider
/provider

# 检查 providers 配置
cat ~/.cc-connect/config.toml | grep -A 5 "providers"
```

**解决**:
```bash
# 重启会话
/new

# 或重启服务
cc-connect daemon restart
```

### 问题: 模型配置不生效

**检查**:
```bash
/model
```

**解决**: 确保配置格式正确，查看 Agent 文档确认支持的模型

---

**最后更新**: 2026-03-14
