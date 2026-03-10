# cc-connect — AI 友好项目文档

本文档面向 **AI 助手**与代码贡献者，提供结构化、可检索的项目上下文，便于理解架构、定位代码和扩展功能。

**使用建议（给 AI）**：  
- 做功能/扩展前先看 **§3 目录结构** 和 **§8 文件速查** 定位相关文件。  
- 改配置或加配置项时结合 **§5 配置结构** 与 `config/config.go`。  
- 新增平台/Agent 时严格按 **§7 扩展指南** 的步骤（含在 `main.go` 中空导入）。

---

## 1. 项目概览

- **名称**: cc-connect  
- **作用**: 将本地运行的 AI 编程 Agent（Claude Code、Codex、Cursor、Gemini CLI 等）桥接到即时通讯平台（飞书、钉钉、Slack、Telegram、Discord、企业微信、LINE、QQ 等），使用户可在聊天应用中远程操控 Agent。
- **语言**: Go 1.22+  
- **配置**: TOML（`config.toml`），支持多项目、多平台、多 Agent 类型。  
- **入口**: 主程序为 `cmd/cc-connect/main.go`，通过 `core.NewEngine` 为每个 `[[projects]]` 创建一个 `Engine`，负责消息路由与斜杠命令。

**核心数据流**：  
平台收到消息 → `MessageHandler(Platform, *Message)` → Engine 解析斜杠命令或转发给 Agent → Agent 输出通过 `Event` 流回 Engine → Engine 调用 `Platform.Reply/Send` 回传用户。

---

## 2. 术语表（中英一致）

| 中文     | 英文 / 代码中用法        | 说明 |
|----------|---------------------------|------|
| 项目     | project                    | 配置中的一个 `[[projects]]`，绑定一个 Agent + 若干 Platform。 |
| 平台     | platform                   | 即时通讯端（飞书、Telegram 等），实现 `core.Platform`。 |
| 引擎     | engine                     | `core.Engine`，单项目消息路由与命令处理。 |
| 会话     | session                    | 用户与 Agent 的对话会话，由 `core.SessionManager` 管理，持久化到 JSON。 |
| Agent 会话 | agent session            | `core.AgentSession`，与单个 Agent 进程的双向交互。 |
| 斜杠命令 | slash command              | 以 `/` 开头的命令，如 `/help`、`/new`，由 Engine 处理。 |
| 回复上下文 | reply context / replyCtx | 平台相关、用于回消息的上下文（如 chat_id），类型为 `any`。 |
| 会话键   | session key                | 唯一标识用户上下文，如 `feishu:{chatID}:{userID}`。 |

---

## 3. 目录与模块结构

```
cc-connect/
├── cmd/cc-connect/          # 入口与子命令
│   ├── main.go               # 解析 config、创建 Engine、注册 platform/agent 包
│   ├── daemon.go             # daemon install/start/stop/...
│   ├── send.go               # cc-connect send
│   ├── cron.go               # cc-connect cron add/list/del
│   ├── provider.go           # cc-connect provider add/list/remove/import
│   ├── relay.go              # cc-connect relay send
│   └── update.go             # cc-connect update / check-update
├── config/
│   └── config.go             # 配置加载、校验、TOML 结构与持久化（provider/command/alias 等）
├── core/                     # 核心抽象与引擎
│   ├── interfaces.go         # Platform、Agent、AgentSession、Message、Event 及可选接口
│   ├── registry.go           # RegisterPlatform / RegisterAgent 工厂注册
│   ├── message.go            # Message、Event、EventType、ImageAttachment、HistoryEntry 等
│   ├── session.go            # Session、SessionManager（多会话与持久化）
│   ├── engine.go             # Engine：消息路由、斜杠命令、流式输出、权限请求
│   ├── command.go            # CustomCommand、CommandRegistry（斜杠命令与 agent 命令目录）
│   ├── skill.go              # SkillRegistry（Agent 提供的 SKILL.md 目录）
│   ├── i18n.go               # 多语言文案（en/zh/zh-TW/ja/es）
│   ├── speech.go             # 语音转文字（STT：Whisper / Groq / Qwen）
│   ├── tts.go                # 文字转语音（TTS：Qwen / OpenAI）
│   ├── cron.go               # 定时任务调度与存储
│   ├── relay.go              # 多机器人中继（RelayManager）
│   ├── api.go                # 内部 API（send、relay 等）
│   ├── providerproxy.go      # 第三方 API 代理
│   └── ...                   # 其他（markdown、ratelimit、dedup、updater、doctor 等）
├── platform/                 # 平台实现（均含 init 中 RegisterPlatform）
│   ├── feishu/feishu.go
│   ├── dingtalk/dingtalk.go
│   ├── telegram/telegram.go
│   ├── slack/slack.go
│   ├── discord/discord.go
│   ├── line/line.go
│   ├── wecom/wecom.go
│   ├── qq/qq.go
│   └── qqbot/qqbot.go
├── agent/                    # Agent 实现（均含 init 中 RegisterAgent）
│   ├── claudecode/claudecode.go  # Claude Code CLI
│   ├── codex/codex.go
│   ├── cursor/cursor.go
│   ├── gemini/gemini.go
│   ├── qoder/qoder.go
│   ├── opencode/opencode.go
│   └── iflow/iflow.go
├── daemon/                   # 守护进程（systemd / launchd）
├── config.example.toml      # 配置示例
├── INSTALL.md               # 安装与配置指南（面向 AI Agent）
└── docs/                    # 文档（平台接入等）
```

---

## 4. 核心接口与类型（代码锚点）

### 4.1 平台：`core.Platform`（`core/interfaces.go`）

```go
type Platform interface {
    Name() string
    Start(handler MessageHandler) error
    Reply(ctx context.Context, replyCtx any, content string) error
    Send(ctx context.Context, replyCtx any, content string) error
    Stop() error
}
```

- **MessageHandler**: `func(p Platform, msg *Message)`，平台收到消息时调用。  
- 可选接口：`ReplyContextReconstructor`（cron 等按 session key 重建 replyCtx）、`TypingIndicator`、`MessageUpdater`、`InlineButtonSender`、`CommandRegistrar`。

### 4.2 消息：`core.Message`（`core/message.go`）

- `SessionKey`、`Platform`、`MessageID`、`UserID`、`UserName`、`Content`  
- `Images []ImageAttachment`、`Audio *AudioAttachment`、`ReplyCtx any`、`FromVoice bool`

### 4.3 Agent：`core.Agent`（`core/interfaces.go`）

```go
type Agent interface {
    Name() string
    StartSession(ctx context.Context, sessionID string) (AgentSession, error)
    ListSessions(ctx context.Context) ([]AgentSessionInfo, error)
    Stop() error
}
```

- **AgentSession**: `Send(prompt, images)`、`RespondPermission`、`Events() <-chan Event`、`CurrentSessionID`、`Alive`、`Close`。  
- 可选接口：`ProviderSwitcher`、`ModelSwitcher`、`ModeSwitcher`、`MemoryFileProvider`、`SystemPromptSupporter`、`SessionEnvInjector`、`ToolAuthorizer`、`HistoryProvider`、`ContextCompressor`、`CommandProvider`、`SkillProvider`、`SessionDeleter`。

### 4.4 事件流：`core.Event`（`core/message.go`）

- `EventType`: `text`、`tool_use`、`tool_result`、`result`、`error`、`permission_request`、`thinking`  
- `Event` 含 `Type`、`Content`、`ToolName`、`ToolInput`、`ToolInputRaw`、`RequestID`、`Done`、`Error` 等。

### 4.5 注册表（`core/registry.go`）

- `RegisterPlatform(name string, factory PlatformFactory)`  
- `RegisterAgent(name string, factory AgentFactory)`  
- `CreatePlatform(name, opts)`、`CreateAgent(name, opts)`  
- 所有 `platform/*` 与 `agent/*` 在 `cmd/cc-connect/main.go` 中通过 `_ "github.com/chenhg5/cc-connect/platform/xxx"` 空导入完成注册。

---

## 5. 配置结构（TOML → config 包）

- **全局**: `config.Config` — `DataDir`、`Projects`、`Commands`、`Aliases`、`BannedWords`、`Log`、`Language`、`Speech`、`TTS`、`Display`、`StreamPreview`、`RateLimit`、`Quiet`、`Cron`、`IdleTimeoutMins`。  
- **项目**: `config.ProjectConfig` — `Name`、`Agent`（`Type` + `Options` + `Providers`）、`Platforms`（`Type` + `Options`）、`Quiet`、`DisabledCommands`。  
- **Agent 选项** 常见键：`work_dir`、`mode`、`model`、`provider`、`router_url`、`router_api_key`；各 Agent 类型在对应 `agent/*/xxx.go` 的 `New(opts)` 中解析。  
- **平台选项** 由各 `platform/*/xxx.go` 的 `New(opts)` 自行解析（如 `token`、`app_id`、`app_secret` 等）。  
- 配置路径：显式 `-config` → `./config.toml` → `~/.cc-connect/config.toml`（见 `main.resolveConfigPath`）。

---

## 6. 引擎与斜杠命令（Engine）

- **Engine**（`core/engine.go`）：每个项目一个实例，持有 `Agent`、`[]Platform`、`SessionManager`、`CommandRegistry`、`SkillRegistry`、cron/relay 等。  
- 用户消息先经 `Engine` 判断是否为斜杠命令；若是则 `handleCommand` 处理并返回，否则作为普通消息发给当前会话的 Agent。  
- 内置命令在 `core/engine.go` 与 `core/i18n.go` 中定义，例如：`/new`、`/list`、`/switch`、`/current`、`/history`、`/mode`、`/model`、`/provider`、`/memory`、`/cron`、`/bind`、`/quiet`、`/stop`、`/help`、`/restart`、`/upgrade`、`/version` 等。  
- 自定义斜杠命令来源：  
  1. 配置 `[commands]`（config 包加载）；  
  2. Agent 的 `CommandProvider.CommandDirs()` 返回目录下的 `*.md` 文件（`core/command.go` 中按名称解析）。  
- 别名：`config.aliases` 与运行时 `AddAlias`，将触发词映射到已有命令（如 `帮助` → `/help`）。

---

## 7. 扩展指南（面向 AI 的修改要点）

### 7.1 新增平台

1. 在 `platform/` 下新建包，实现 `core.Platform` 五件套。  
2. 在 `init()` 中调用 `core.RegisterPlatform("myplatform", New)`。  
3. `New(opts map[string]any)` 从 `opts` 读取所需配置并返回实现。  
4. 在 `cmd/cc-connect/main.go` 中添加：`_ "github.com/chenhg5/cc-connect/platform/myplatform"`。  
5. 在 `config.example.toml` 与文档中补充该平台示例。

### 7.2 新增 Agent

1. 在 `agent/` 下新建包，实现 `core.Agent`（及 `StartSession` 返回的 `AgentSession`）。  
2. 在 `init()` 中调用 `core.RegisterAgent("myagent", New)`。  
3. `New(opts map[string]any)` 从 `opts` 读取 `work_dir`、`mode` 等。  
4. 在 `cmd/cc-connect/main.go` 中添加：`_ "github.com/chenhg5/cc-connect/agent/myagent"`。  
5. 若需权限/模式/模型/Provider，实现对应可选接口（如 `ModeSwitcher`、`ModelSwitcher`、`ProviderSwitcher`）。

### 7.3 新增斜杠命令

- **仅本项目使用**：在 `core/engine.go` 的 `handleCommand` 中增加分支，必要时在 `core/i18n.go` 增加文案。  
- **用户可配置**：在 `config.CommandConfig` 与 `config.Load` 中已支持；通过 `engine.AddCommand(...)` 在 main 中注册。  
- **由 Agent 提供**：实现 `CommandProvider.CommandDirs()`，在指定目录下放置 `命令名.md`（内容为 prompt 或说明），Engine 会扫描并注册。

---

## 8. 关键文件速查

| 目的                     | 文件路径 |
|--------------------------|----------|
| 程序入口与 Engine 创建   | `cmd/cc-connect/main.go` |
| 平台/Agent 接口定义      | `core/interfaces.go` |
| 消息与事件类型           | `core/message.go` |
| 平台/Agent 注册与创建    | `core/registry.go` |
| 消息路由与斜杠命令       | `core/engine.go` |
| 自定义命令与 agent 命令  | `core/command.go` |
| 会话管理                 | `core/session.go` |
| 配置加载与结构           | `config/config.go` |
| 示例平台实现             | `platform/telegram/telegram.go` |
| 示例 Agent 实现         | `agent/claudecode/claudecode.go` |
| 多语言文案               | `core/i18n.go` |
| 安装与配置（给 AI 用）   | `INSTALL.md` |

---

## 9. 常用操作与入口

- **运行**: `cc-connect` 或 `cc-connect -config /path/to/config.toml`  
- **发送消息到会话**: `cc-connect send -m "..." -p <project> -s <session>`（依赖内部 API）  
- **定时任务**: `cc-connect cron add/list/del`；聊天中 `/cron`  
- **Provider 管理**: `cc-connect provider add/list/remove/import`；聊天中 `/provider`  
- **多机器人中继**: `cc-connect relay send --to <project> "..."`；聊天中 `/bind`  
- **守护进程**: `cc-connect daemon install/start/stop/...`  
- **系统提示词**（cron/relay 说明）：`core.AgentSystemPrompt()`（`core/interfaces.go`）；支持该提示词的 Agent 实现 `SystemPromptSupporter`。

---

本文档与代码同步更新。修改接口或目录时请同步修改此文件，以便 AI 与贡献者快速定位与扩展。
