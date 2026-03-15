# Agent 集成指南 | Agent Integration Guide

本文档指导如何为 cc-connect 添加新的 AI Agent 支持。

---

## 目录

| 中文 | English |
|------|---------|
| [快速开始](#快速开始) | [Quick Start](#quick-start) |
| [架构要求](#架构要求) | [Architecture Requirements](#architecture-requirements) |
| [实现步骤](#实现步骤) | [Implementation Steps](#implementation-steps) |
| [测试验证](#测试验证) | [Testing & Validation](#testing--validation) |
| [常见问题](#常见问题) | [FAQ](#faq) |

---

## 快速开始

### 最小实现模板

```go
// agent/myagent/myagent.go
package myagent

import (
    "context"
    "github.com/chenhg5/cc-connect/core"
)

func init() {
    core.RegisterAgent("myagent", New)
}

func New(opts map[string]any) (core.Agent, error) {
    return &Agent{opts: opts}, nil
}

type Agent struct {
    opts map[string]any
}

func (a *Agent) Name() string {
    return "myagent"
}

func (a *Agent) StartSession(ctx context.Context, sessionID string) (core.AgentSession, error) {
    // 实现会话启动逻辑
    return &Session{}, nil
}

func (a *Agent) ListSessions(ctx context.Context) ([]core.AgentSessionInfo, error) {
    // 实现会话列表获取
    return nil, nil
}

func (a *Agent) Stop() error {
    // 实现资源清理
    return nil
}
```

### 注册 Agent

在 `cmd/cc-connect/main.go` 中添加空导入：

```go
import (
    _ "github.com/chenhg5/cc-connect/agent/myagent"
    // ... 其他导入
)
```

新增 CLI agent 时，优先选择其机器可读协议或结构化事件流接口，例如：

- Claude Code: `--input-format stream-json`
- Codex: `codex exec --json`
- Gemini CLI: `--output-format stream-json`
- GitHub Copilot CLI: `--acp --stdio`

不要优先解析面向人类的 TUI 输出，除非该 CLI 没有更稳定的集成协议。

### 配置示例

在 `config.example.toml` 中添加：

```toml
[[projects]]
name = "my-project"

[projects.agent]
type = "myagent"

[projects.agent.options]
work_dir = "/path/to/project"
# myagent-specific options here
```

---

## 架构要求

### 核心接口

所有 Agent 必须实现 `core.Agent` 接口：

```go
type Agent interface {
    Name() string
    StartSession(ctx context.Context, sessionID string) (AgentSession, error)
    ListSessions(ctx context.Context) ([]AgentSessionInfo, error)
    Stop() error
}
```

### 会话接口

AgentSession 必须实现：

```go
type AgentSession interface {
    Send(prompt string, images []*ImageAttachment) error
    RespondPermission(requestID string, allow bool) error
    Events() <-chan core.Event
    CurrentSessionID() string
    Alive() bool
    Close() error
}
```

---

## 实现步骤

### Step 1: 创建包结构

```
agent/myagent/
├── myagent.go      # Agent 主实现
├── session.go      # 会话实现
└── session_test.go # 单元测试（可选但推荐）
```

### Step 2: 实现 Agent 接口

#### 基础字段

```go
type Agent struct {
    opts     map[string]any
    workDir  string
    mode     string
    apiKeys  map[string]string
}
```

#### 解析配置

```go
func New(opts map[string]any) (core.Agent, error) {
    workDir, ok := opts["work_dir"].(string)
    if !ok || workDir == "" {
        return nil, errors.New("work_dir is required")
    }
    
    mode, _ := opts["mode"].(string)
    
    return &Agent{
        opts:    opts,
        workDir: workDir,
        mode:    mode,
    }, nil
}
```

### Step 3: 实现 Session 接口

#### 启动子进程

```go
func (a *Agent) StartSession(ctx context.Context, sessionID string) (core.AgentSession, error) {
    cmd := exec.CommandContext(ctx, "myagent-cli", "run")
    cmd.Dir = a.workDir
    
    // 设置环境变量
    cmd.Env = os.Environ()
    cmd.Env = append(cmd.Env, fmt.Sprintf("MYAGENT_MODE=%s", a.mode))
    
    // 启动进程
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    return &Session{
        cmd:    cmd,
        sessionID: sessionID,
    }, nil
}
```

#### 发送消息

```go
func (s *Session) Send(prompt string, images []*core.ImageAttachment) error {
    // 将 prompt 写入临时文件或通过 stdin 传递
    input := struct {
        Prompt  string               `json:"prompt"`
        Images  []string             `json:"images"`
        Session string               `json:"session_id"`
    }{
        Prompt:  prompt,
        Session: s.sessionID,
    }
    
    data, _ := json.Marshal(input)
    _, err := s.cmd.StdinPipe.Write(data)
    return err
}
```

#### 事件流处理

```go
func (s *Session) Events() <-chan core.Event {
    ch := make(chan core.Event, 16)
    
    go func() {
        defer close(ch)
        scanner := bufio.NewScanner(s.cmd.Stdout)
        
        for scanner.Scan() {
            line := scanner.Text()
            
            // 解析 Agent 输出
            var evt myagentEvent
            json.Unmarshal([]byte(line), &evt)
            
            // 转换为 cc-connect Event
            switch evt.Type {
            case "text":
                ch <- core.Event{
                    Type:    core.EventTypeText,
                    Content: evt.Content,
                }
            case "tool_use":
                ch <- core.Event{
                    Type:      core.EventTypeToolUse,
                    ToolName:  evt.Tool,
                    ToolInput: evt.Input,
                }
            case "result":
                ch <- core.Event{
                    Type:     core.EventTypeResult,
                    Content:  evt.Content,
                    Done:     true,
                }
            }
        }
    }()
    
    return ch
}
```

### Step 4: 可选接口实现

根据 Agent 能力，实现以下可选接口以增强功能：

#### ProviderSwitcher - API Provider 切换

```go
func (a *Agent) SupportsProviderSwitching() bool {
    return true
}

func (a *Agent) SetProvider(env map[string]string) error {
    a.apiKeys = env
    // 通知正在运行的 session 更新
    return nil
}
```

#### ModeSwitcher - 权限模式切换

```go
func (a *Agent) SupportsModeSwitching() bool {
    return true
}

func (a *Agent) SetMode(mode string) error {
    a.mode = mode
    return nil
}
```

#### ModelSwitcher - 模型切换

```go
func (a *Agent) SupportsModelSwitching() bool {
    return true
}

func (a *Agent) SetModel(model string) error {
    a.opts["model"] = model
    return nil
}

func (a *Agent) AvailableModels() []string {
    return []string{"myagent-1", "myagent-2", "myagent-vision"}
}
```

#### CommandProvider - 自定义命令

```go
func (a *Agent) CommandDirs() []string {
    return []string{
        filepath.Join(a.workDir, ".myagent", "commands"),
        filepath.Join(xdg.ConfigHome, "myagent", "commands"),
        filepath.Join(os.Getenv("HOME"), ".config", "myagent", "commands"),
    }
}
```

#### SkillProvider - 技能发现

```go
func (a *Agent) SkillDirs() []string {
    return []string{
        filepath.Join(a.workDir, ".myagent", "skills"),
        filepath.Join(xdg.ConfigHome, "myagent", "skills"),
        filepath.Join(os.Getenv("HOME"), ".config", "myagent", "skills"),
    }
}
```

#### MemoryFileProvider - 记忆文件

```go
func (a *Agent) GetMemoryFile() (string, error) {
    return filepath.Join(a.workDir, ".myagent", "memory.md"), nil
}
```

---

## 测试验证

### 单元测试

```go
// agent/myagent/myagent_test.go
package myagent

import (
    "context"
    "testing"
)

func TestAgentNew(t *testing.T) {
    opts := map[string]any{
        "work_dir": "/tmp/test",
        "mode":     "default",
    }
    
    agent, err := New(opts)
    if err != nil {
        t.Fatalf("New() failed: %v", err)
    }
    
    if agent.Name() != "myagent" {
        t.Errorf("expected name 'myagent', got %s", agent.Name())
    }
}

func TestAgentStartSession(t *testing.T) {
    agent := &Agent{workDir: "/tmp/test"}
    
    ctx := context.Background()
    session, err := agent.StartSession(ctx, "test-session")
    if err != nil {
        t.Fatalf("StartSession() failed: %v", err)
    }
    
    if session == nil {
        t.Fatal("StartSession() returned nil session")
    }
    
    // 清理
    session.Close()
}
```

### 集成测试

```go
// agent/myagent/session_test.go
package myagent

import (
    "context"
    "testing"
    "time"
)

func TestSessionEventFlow(t *testing.T) {
    agent := &Agent{workDir: "/tmp/test"}
    ctx := context.Background()
    
    session, err := agent.StartSession(ctx, "test")
    if err != nil {
        t.Fatalf("Failed to start session: %v", err)
    }
    defer session.Close()
    
    events := session.Events()
    
    // 发送消息
    err = session.Send("hello", nil)
    if err != nil {
        t.Fatalf("Send() failed: %v", err)
    }
    
    // 等待事件
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    select {
    case evt, ok := <-events:
        if !ok {
            t.Fatal("events channel closed unexpectedly")
        }
        t.Logf("Received event: %+v", evt)
    case <-ctx.Done():
        t.Fatal("Timeout waiting for events")
    }
}
```

### 验证命令

```bash
# 运行 Agent 测试
go test ./agent/myagent/... -v

# 运行所有测试
go test ./... -v

# 构建项目
make build

# 检查注册
make check-harness
```

---

## 配置文档

### 更新 config.example.toml

```toml
[[projects]]
name = "my-agent-project"

[projects.agent]
type = "myagent"

[projects.agent.options]
work_dir = "/path/to/project"
mode = "default"
# model = "myagent-1"  # 可选

# Provider 配置
[[projects.agent.providers]]
name = "myagent"
api_key = "${MYAGENT_API_KEY}"
```

### 更新 INSTALL.md

在 `INSTALL.md` 中添加 Agent 安装说明：

```markdown
### MyAgent CLI

```bash
# 安装 MyAgent CLI
npm install -g myagent-cli

# 验证安装
myagent --version
```

**Config:**

```toml
[[projects]]
name = "my-agent-project"

[projects.agent]
type = "myagent"

[projects.agent.options]
work_dir = "/absolute/path/to/your/project"
mode = "default"

[[projects.agent.providers]]
name = "myagent"
api_key = "your-api-key"
```
```

---

## 常见问题

### Q1: Agent 没有响应？

**检查项**：
1. Agent CLI 是否安装并可执行
2. `work_dir` 路径是否存在
3. 查看 cc-connect 日志：`cc-connect daemon logs -f`

### Q2: 事件流不工作？

**检查项**：
1. Agent 子进程的 stdout 是否正确输出
2. 事件格式是否为 JSON（或按预期解析）
3. 流式输出是否正确处理

### Q3: 权限模式不生效？

**解决方法**：
1. 确保实现了 `ModeSwitcher` 接口
2. 验证 `SetMode()` 方法正确应用到子进程
3. 检查 Agent CLI 文档支持的模式参数

### Q4: 如何调试 Agent 集成？

**方法**：
```go
// 启用 debug 日志
[log]
level = "debug"

# 在代码中添加日志
log.Printf("Starting session with work_dir: %s", a.workDir)
```

### Q5: 支持多会话吗？

是的，只要每次 `StartSession()` 创建独立的子进程和会话状态即可。

---

## 模板仓库

可以参考以下实现：

- [Claude Code](https://github.com/chenhg5/cc-connect/tree/main/agent/claudecode)
- [Codex](https://github.com/chenhg5/cc-connect/tree/main/agent/codex)
- [Gemini](https://github.com/chenhg5/cc-connect/tree/main/agent/gemini)

---

## 下一步

- [ ] 创建 PR 提交 Agent 实现
- [ ] 更新 README.md 添加支持状态
- [ ] 添加平台测试（如果适用）

---

**维护者**: cc-connect Team

**最后更新**: 2026-03-14
