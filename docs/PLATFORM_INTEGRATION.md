# 平台接入指南 | Platform Integration Guide

本文档指导如何为 cc-connect 添加新的聊天平台支持。

---

## 目录

| 中文 | English |
|------|---------|
| [快速开始](#快速开始) | [Quick Start](#quick-start) |
| [架构要求](#架构要求) | [Architecture Requirements](#architecture-requirements) |
| [实现步骤](#实现步骤) | [Implementation Steps](#implementation-steps) |
| [测试验证](#测试验证) | [Testing & Validation](#testing--validation) |
| [常见平台类型](#常见平台类型) | [Common Platform Types](#common-platform-types) |

---

## 快速开始

### 最小实现模板

```go
// platform/myplatform/myplatform.go
package myplatform

import (
    "context"
    "github.com/chenhg5/cc-connect/core"
)

func init() {
    core.RegisterPlatform("myplatform", New)
}

func New(opts map[string]any) (core.Platform, error) {
    return &Platform{opts: opts}, nil
}

type Platform struct {
    opts map[string]any
}

func (p *Platform) Name() string {
    return "myplatform"
}

func (p *Platform) Start(handler core.MessageHandler) error {
    // 实现启动逻辑
    return nil
}

func (p *Platform) Reply(ctx context.Context, replyCtx any, content string) error {
    // 实现回复消息
    return nil
}

func (p *Platform) Send(ctx context.Context, replyCtx any, content string) error {
    // 实现发送消息
    return nil
}

func (p *Platform) Stop() error {
    // 实现资源清理
    return nil
}
```

### 注册平台

在 `cmd/cc-connect/main.go` 中添加空导入：

```go
import (
    _ "github.com/chenhg5/cc-connect/platform/myplatform"
    // ... 其他导入
)
```

### 配置示例

在 `config.example.toml` 中添加：

```toml
[[projects]]
name = "my-project"

[[projects.platforms]]
type = "myplatform"

[projects.platforms.options]
# platform-specific options here
```

---

## 架构要求

### 核心接口

所有平台必须实现 `core.Platform` 接口：

```go
type Platform interface {
    Name() string
    Start(handler core.MessageHandler) error
    Reply(ctx context.Context, replyCtx any, content string) error
    Send(ctx context.Context, replyCtx any, content string) error
    Stop() error
}
```

###MessageHandler 签名

```go
type MessageHandler func(p Platform, msg *core.Message)
```

### 可选接口

根据平台能力，实现以下可选接口：

```go
// 重建 replyCtx（用于 cron 等场景）
type ReplyContextReconstructor interface {
    ReconstructReplyCtx(sessionKey string) (any, error)
}

// 输入指示器（展示"正在输入"状态）
type TypingIndicator interface {
    Typing(ctx context.Context, replyCtx any) error
}

// 消息更新（用于流式预览）
type MessageUpdater interface {
    UpdateMessage(ctx context.Context, replyCtx any, content string) error
}

// 内联按钮
type InlineButtonSender interface {
    SendInlineButtons(ctx context.Context, replyCtx any, content string, buttons [][]InlineButton) error
}

// 命令注册（用于设置 bot menu）
type CommandRegistrar interface {
    RegisterCommands(ctx context.Context, commands []Command) error
}
```

---

## 实现步骤

### Step 1: 创建包结构

```
platform/myplatform/
├── myplatform.go   # Platform 主实现
├── platform_test.go # 单元测试（可选）
```

### Step 2: 解析配置

```go
type Platform struct {
    opts        map[string]any
    token       string
    webhookURL  string
    port        int
    messageBus  map[string]chan *core.Message
}

func New(opts map[string]any) (core.Platform, error) {
    token, ok := opts["token"].(string)
    if !ok || token == "" {
        return nil, errors.New("token is required")
    }
    
    port, _ := opts["port"].(int)
    if port == 0 {
        port = 8080
    }
    
    return &Platform{
        opts:     opts,
        token:    token,
        port:     port,
        messageBus: make(map[string]chan *core.Message),
    }, nil
}
```

### Step 3: 实现 Start()

#### Webhook 平台（需要公网 URL）

```go
func (p *Platform) Start(handler core.MessageHandler) error {
    // 创建 HTTP 服务器
    http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost {
            // 解析 webhook payload
            body, _ := io.ReadAll(r.Body)
            var payload webHookPayload
            json.Unmarshal(body, &payload)
            
            // 构建 cc-connect Message
            msg := &core.Message{
                Platform:  p.Name(),
                SessionKey: fmt.Sprintf("%s:%s", p.Name(), payload.UserID),
                UserID:    payload.UserID,
                Content:   payload.Text,
                ReplyCtx:  payload.ReplyContext,
            }
            
            // 调用 handler
            handler(p, msg)
        }
    })
    
    // 启动服务器
    addr := fmt.Sprintf(":%d", p.port)
    return http.ListenAndServe(addr, nil)
}
```

#### WebSocket 平台（长连接）

```go
func (p *Platform) Start(handler core.MessageHandler) error {
    // 连接到 WebSocket
    conn, _, err := websocket.DefaultDialer.Dial(p.wsURL, nil)
    if err != nil {
        return err
    }
    
    go func() {
        defer conn.Close()
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                log.Printf("WebSocket error: %v", err)
                return
            }
            
            // 解析消息
            var msg websocketMessage
            json.Unmarshal(message, &msg)
            
            // 构建 cc-connect Message
            ccMsg := &core.Message{
                Platform:   p.Name(),
                SessionKey: fmt.Sprintf("%s:%s", p.Name(), msg.FromID),
                UserID:     msg.FromID,
                Content:    msg.Text,
                ReplyCtx:   msg.ChatID,
            }
            
            handler(p, ccMsg)
        }
    }()
    
    return nil
}
```

### Step 4: 实现 Reply()

```go
func (p *Platform) Reply(ctx context.Context, replyCtx any, content string) error {
    // 构建回复 payload
    payload := map[string]any{
        "to":      replyCtx,
        "message": content,
        "token":   p.token,
    }
    
    data, _ := json.Marshal(payload)
    
    // 发送 HTTP 请求
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewBuffer(data))
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("platform API returned %d", resp.StatusCode)
    }
    
    return nil
}
```

### Step 5: 实现 Send()

```go
func (p *Platform) Send(ctx context.Context, replyCtx any, content string) error {
    // Send 通常用于主动消息（非回复）
    // 实现逻辑可以类似 Reply，但无需 replyCtx
    
    payload := map[string]any{
        "chat_id": replyCtx,
        "text":    content,
        "token":   p.token,
    }
    
    data, _ := json.Marshal(payload)
    
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL+"/sendMessage", bytes.NewBuffer(data))
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

### Step 6: 实现 Stop()

```go
func (p *Platform) Stop() error {
    // 清理资源
    // 关闭 WebSocket 连接
    // 关闭 HTTP 服务器
    // 清理 Goroutines
    
    return nil
}
```

---

## 常见平台类型

### 类型 1: Webhook 平台（需要公网 IP）

**代表**: LINE, WeChat Work

**特点**：
- 平台向你的服务器发送 HTTP POST
- 需要公网可访问的 URL
- 通常需要 HTTPS

**实现要点**：
```go
// 必须
type Platform struct {
    port         int
    callbackPath string
    token        string  # 验证 token
}

// 启动 HTTP 服务器
func (p *Platform) Start(handler core.MessageHandler) error {
    http.HandleFunc(p.callbackPath, p.handleWebhook)
    return http.ListenAndServe(fmt.Sprintf(":%d", p.port), nil)
}

// 处理 webhook
func (p *Platform) handleWebhook(w http.ResponseWriter, r *http.Request) {
    # 验证签名/token
    # 解析 payload
    # 调用 handler
}
```

### 类型 2: WebSocket 长连接（无需公网 IP）

**代表**: Feishu, DingTalk, Telegram (Long Polling)

**特点**：
- 主动连接平台 WebSocket
- 无需公网 IP
- 适合本地开发

**实现要点**：
```go
// WebSocket 连接
func (p *Platform) Start(handler core.MessageHandler) error {
    conn, _, err := websocket.DefaultDialer.Dial(p.wsURL, nil)
    
    go p.receiveLoop(conn, handler)
    go p.sendLoop(conn)
    
    return nil
}

func (p *Platform) receiveLoop(conn *websocket.Conn, handler core.MessageHandler) {
    for {
        _, msg, err := conn.ReadMessage()
        # 处理消息
    }
}
```

### 类型 3: Long Polling（无需公网 IP）

**代表**: Telegram

**特点**：
- 定期轮询平台 API
- 实现简单
- 有一定延迟

**实现要点**：
```go
func (p *Platform) Start(handler core.MessageHandler) error {
    go func() {
        offset := 0
        for {
            url := fmt.Sprintf("%s/getUpdates?offset=%d", p.apiURL, offset)
            resp, _ := http.Get(url)
            
            var result struct {
                Result []Update `json:"result"`
            }
            json.NewDecoder(resp.Body).Decode(&result)
            
            for _, update := range result.Result {
                offset = update.UpdateID + 1
                handler(p, p.parseMessage(update.Message))
            }
            
            time.Sleep(1 * time.Second)
        }
    }()
    
    return nil
}
```

---

## 测试验证

### 单元测试

```go
// platform/myplatform/platform_test.go
package myplatform

import (
    "testing"
)

func TestPlatformNew(t *testing.T) {
    opts := map[string]any{
        "token": "test-token",
        "port":  8080,
    }
    
    p, err := New(opts)
    if err != nil {
        t.Fatalf("New() failed: %v", err)
    }
    
    if p.Name() != "myplatform" {
        t.Errorf("expected name 'myplatform', got %s", p.Name())
    }
}

func TestPlatformStartStop(t *testing.T) {
    opts := map[string]any{
        "token": "test-token",
        "port":  8080,
    }
    
    p, err := New(opts)
    if err != nil {
        t.Fatalf("New() failed: %v", err)
    }
    
    err = p.Start(func(p core.Platform, msg *core.Message) {})
    if err != nil {
        t.Fatalf("Start() failed: %v", err)
    }
    
    err = p.Stop()
    if err != nil {
        t.Fatalf("Stop() failed: %v", err)
    }
}
```

### 集成测试

```bash
# 1. 启动平台模拟器（如 Docker）
docker run -d --name myplatform -p 8080:8080 myplatform/mock

# 2. 运行 cc-connect
cc-connect -config test-config.toml

# 3. 发送测试消息
curl -X POST http://localhost:8080/test-message \
  -H "Content-Type: application/json" \
  -d '{"text": "test", "user_id": "user123"}'

# 4. 验证响应
```

### 验证命令

```bash
# 测试平台包
go test ./platform/myplatform/... -v

# 整体测试
go test ./... -v

# 构建
make build

# 检查注册
make check-harness
```

---

## 配置文档

### 更新 config.example.toml

```toml
[[projects]]
name = "my-project"

[[projects.platforms]]
type = "myplatform"

[projects.platforms.options]
token = "your-token"
# platform-specific options
port = 8080
callback_path = "/callback"
```

### 更新 INSTALL.md

```markdown
### MyPlatform

**Setup steps:**
1. Create app at MyPlatform developer console
2. Copy credential tokens
3. Configure cc-connect

**Config:**

```toml
[[projects.platforms]]
type = "myplatform"

[projects.platforms.options]
token = "your-token"
port = 8080
```
```

---

## 常见问题

### Q1: 如何选择平台类型？

| 因素 | 推荐类型 |
|------|---------|
| 有 WebSocket API | WebSocket |
| 有 Long Polling API | Long Polling |
| 只有 Webhook | Webhook |
| 需要公网 IP | Webhook |
| 本地开发优先 | WebSocket/Long Polling |

### Q2: 如何处理消息重复？

```go
// 使用 message ID 去重
seen := make(map[string]bool)

func (p *Platform) handleWebhook(w http.ResponseWriter, r *http.Request) {
    msgID := payload.MessageID
    
    if seen[msgID] {
        return // 重复消息
    }
    seen[msgID] = true
    
    # 处理消息
}
```

### Q3: 如何处理Rate Limiting？

```go
// 使用时间窗口限速
var (
    lastRequest time.Time
    rateLimit   = 100 * time.Millisecond
)

func (p *Platform) sendMessage(msg string) error {
    elapsed := time.Since(lastRequest)
    if elapsed < rateLimit {
        time.Sleep(rateLimit - elapsed)
    }
    
    # 发送消息
    lastRequest = time.Now()
    return nil
}
```

---

## 模板仓库

参考实现：

- [Feishu](https://github.com/chenhg5/cc-connect/tree/main/platform/feishu) (WebSocket)
- [Telegram](https://github.com/chenhg5/cc-connect/tree/main/platform/telegram) (Long Polling)
- [LINE](https://github.com/chenhg5/cc-connect/tree/main/platform/line) (Webhook)
- [WeChat Work](https://github.com/chenhg5/cc-connect/tree/main/platform/wecom) (Webhook)

---

## 下一步

- [ ] 创建 PR 提交平台实现
- [ ] 更新 README.md 添加支持状态
- [ ] 编写平台专用文档（参考 docs/feishu.md）

---

**维护者**: cc-connect Team

**最后更新**: 2026-03-14
