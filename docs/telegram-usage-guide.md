# Telegram 平台详细使用指南

本文档介绍如何在 cc-connect 中完整使用 Telegram 平台：从创建机器人、配置、启动，到日常对话、群聊规则、消息类型、斜杠命令、权限审批及常见问题排查。

---

## 一、简介与特点

### 1.1 能做什么

- 在 **Telegram 私聊或群聊** 中与本地 AI Agent（Claude Code、Codex、Cursor、Gemini CLI 等）对话。
- 支持 **文字、图片、语音消息**；Agent 的回复以 Markdown 渲染后的形式展示。
- 支持 **斜杠命令** 管理会话、切换模式/模型/Provider、定时任务、多机器人绑定等。
- **无需公网 IP**：采用 Long Polling，本机可访问 Telegram API 即可（可搭配代理）。

### 1.2 连接方式

| 项目     | 说明 |
|----------|------|
| 连接方式 | Long Polling（cc-connect 主动轮询 Telegram 服务器） |
| 公网 IP  | 不需要 |
| 域名/证书 | 不需要 |
| 适用场景 | 本地开发、内网、家庭网络 + 代理 |

---

## 二、前置条件

- 已安装并配置好 **cc-connect**（参见 [INSTALL.md](../INSTALL.md)）。
- 已配置至少一种 **Agent**（如 Claude Code、Codex、Cursor 等）。
- 拥有 **Telegram 账号**，并能访问 [@BotFather](https://t.me/BotFather)。

---

## 三、创建 Telegram 机器人

### 3.1 打开 BotFather

在 Telegram 中搜索 **@BotFather**（官方机器人管理），与其对话。

> ⚠️ 请认准官方认证的 BotFather，勿用仿冒账号。

### 3.2 创建新机器人

1. 发送：`/newbot`
2. 按提示输入 **显示名称**（如：`My CC-Connect`）
3. 输入 **用户名**（必须以 `bot` 结尾，且全局唯一，如：`my_cc_connect_bot`）

### 3.3 获取 Token

创建成功后，BotFather 会返回类似：

```
Use this token to access the HTTP API:
1234567890:ABCdefGHIjklMNOpqrsTUVwxyz-123456
```

请立即保存该 **Token**。若丢失，可在 BotFather 中：`/mybots` → 选择你的机器人 → **API Token** → **Revoke current token** 重新生成。

### 3.4 群聊中让机器人响应（可选）

若要在 **群组** 中使用机器人，需关闭「群组隐私」：

1. 向 BotFather 发送 `/mybots`
2. 选择你的机器人 → **Bot Settings** → **Group Privacy**
3. 选择 **Turn off**

关闭后，机器人在群内才能收到每条消息（cc-connect 再根据是否 @ 机器人或回复机器人来决定是否处理）。

---

## 四、配置说明

### 4.1 最小配置

在 `config.toml` 的某个 `[[projects]]` 下增加 Telegram 平台块，**必填项仅为 `token`**：

```toml
[[projects]]
name = "my-project"

[projects.agent]
type = "claudecode"

[projects.agent.options]
work_dir = "/path/to/your/project"
mode = "default"

[[projects.platforms]]
type = "telegram"

[projects.platforms.options]
token = "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz-123456"
```

### 4.2 完整配置项

| 选项 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `token` | string | ✅ | BotFather 颁发的 Bot Token |
| `allow_from` | string | 否 | 允许使用的用户 ID，逗号分隔；留空或 `"*"` 表示不限制 |
| `proxy` | string | 否 | HTTP 代理 URL，如 `http://127.0.0.1:7890` |
| `proxy_username` | string | 否 | 代理用户名（若代理需要认证） |
| `proxy_password` | string | 否 | 代理密码 |
| `group_reply_all` | bool | 否 | 为 `true` 时，群内每条消息都交给 Agent（多机器人中继时使用）；默认仅处理 @ 或回复机器人的消息 |
| `share_session_in_channel` | bool | 否 | 为 `true` 时，按「聊天 ID」共享会话（频道场景）；默认按「聊天 ID + 用户 ID」区分会话 |

### 4.3 配置示例

**仅限制用户：**

```toml
[[projects.platforms]]
type = "telegram"

[projects.platforms.options]
token = "你的Token"
allow_from = "123456789,987654321"
```

设置后，只有这些 `message.from.id` 对应的 Telegram 用户能实际使用机器人；其他人即使搜到机器人，也不会被处理。

**使用 HTTP 代理（无法直连 Telegram 时）：**

```toml
[[projects.platforms]]
type = "telegram"

[projects.platforms.options]
token = "你的Token"
proxy = "http://127.0.0.1:7890"
# proxy_username = "user"
# proxy_password = "pass"
```

**群内每条消息都触发（多 bot 协作）：**

```toml
[[projects.platforms]]
type = "telegram"

[projects.platforms.options]
token = "你的Token"
group_reply_all = true
```

### 4.4 获取用户 ID（用于 allow_from 白名单）

1. 先不设 `allow_from`，启动 cc-connect 后向机器人发一条消息。
2. 浏览器访问（将 `你的Token` 替换为实际 token）：
   ```
   https://api.telegram.org/bot你的Token/getUpdates
   ```
3. 在返回的 JSON 中查看 `message.from.id`，即为该用户的数字 ID。

---

## 五、启动与验证

### 5.1 启动

```bash
cc-connect
# 或指定配置文件
cc-connect -config /path/to/config.toml
```

### 5.2 日志确认

成功连接 Telegram 时，日志类似：

```
level=INFO msg="telegram: connected" bot=你的机器人用户名
level=INFO msg="platform started" project=my-project platform=telegram
level=INFO msg="cc-connect is running" projects=1
```

若使用代理，还会看到：

```
level=INFO msg="telegram: using proxy" proxy=127.0.0.1:7890
```

### 5.3 首次对话

- **私聊**：在 Telegram 中搜索机器人用户名 → 点击 **Start** → 直接发送文字。
- **群聊**：将机器人拉入群后，需 **@机器人用户名** 或 **回复机器人的某条消息**，机器人才会处理（除非配置了 `group_reply_all = true`）。

---

## 六、使用方式详解

### 6.1 私聊（Direct Message）

- 会话键为 `telegram:{chatID}:{userID}`，每人独立会话。
- 直接发文字、图片或语音即可，无需 @ 或回复。

### 6.2 群聊（Group / Supergroup）

默认行为（`group_reply_all = false`）下，以下消息会被视为「发给该机器人」并交给 Agent：

1. **@ 机器人**  
   例如：`@my_cc_connect_bot 帮我写一段 Python`
2. **回复机器人的某条消息**  
   在「回复」中输入任意文字即可。
3. **斜杠命令**  
   - `/help`、`/new` 等：若带 `@机器人`（如 `/help@my_cc_connect_bot`），仅该机器人响应。  
   - 不带 `@` 的斜杠命令（如 `/help`）会由当前项目下的该平台处理（单 bot 群内通常只有一个）。

其他群内消息（未 @、未回复机器人）会被忽略，避免误触发。

### 6.3 会话隔离

- **私聊**：每个用户一个会话。
- **群聊**：默认按「群 + 用户」区分，即同一群内不同用户会话独立；同一用户在不同群的会话也独立。

---

## 七、消息类型支持

### 7.1 文字

- 直接发送即可，会作为当前会话的用户输入交给 Agent。
- 群内需满足上述「@ 或回复」规则。

### 7.2 图片

- 发送 **照片**（Photo）时，cc-connect 会下载图片并随当前会话一起发给 Agent。
- 若 Agent 支持多模态（如 Claude Code），可识别图片内容。
- 图片可带 **说明（Caption）**，说明文字会一并作为用户输入；若说明中带 @ 机器人，@ 部分会被去掉后再传给 Agent。

### 7.3 语音消息（Voice）

- 发送 **语音消息** 时，若已在全局配置中开启 [语音转文字（STT）](../README.md#voice-messages-speech-to-text)，cc-connect 会先转成文字再发给 Agent。
- 需在 `config.toml` 中配置 `[speech]`（如 OpenAI / Groq / Qwen 的 Whisper 等），并安装 `ffmpeg`（用于格式转换）。

### 7.4 音频文件（Audio）

- 发送 **音频文件**（非语音消息）同样会按 STT 流程处理（若启用），行为与语音消息类似。

---

## 八、斜杠命令一览

以下命令在 Telegram 中以 `/命令名` 或 `/命令名 参数` 使用；部分支持**前缀缩写**（如 `/pro l` 等价于 `/provider list`）。

| 命令 | 说明 |
|------|------|
| `/help` | 显示帮助与所有可用命令 |
| `/new [名称]` | 新建会话，可选会话名称 |
| `/list [页码]` | 列出当前项目的 Agent 会话，支持分页 |
| `/switch <会话ID>` | 切换到指定会话 |
| `/current` | 显示当前会话信息 |
| `/history [n]` | 显示最近 n 条消息（默认 10） |
| `/mode [名称]` | 查看或切换权限模式（如 default / yolo / plan） |
| `/model [名称]` | 查看或切换模型（若 Agent 支持） |
| `/provider` | 查看当前 API Provider |
| `/provider list` | 列出所有 Provider |
| `/provider switch <名称>` | 切换 Provider |
| `/provider add ...` / `remove` | 添加/移除 Provider（见 README） |
| `/memory` | 读写 Agent 指令文件（若 Agent 支持） |
| `/cron` | 列出定时任务 |
| `/cron add <分 时 日 月 周> <任务描述>` | 添加定时任务 |
| `/cron del <id>` | 删除定时任务 |
| `/bind` | 查看当前群聊绑定的项目 |
| `/bind <项目名>` | 将项目绑定到当前群（多机器人中继） |
| `/bind -<项目名>` | 解除绑定 |
| `/quiet` | 切换「思考过程/工具进度」是否推送到聊天 |
| `/stop` | 停止当前 Agent 执行 |
| `/restart` | 重启 cc-connect 进程（若启用） |
| `/upgrade` | 检查并升级 cc-connect（若启用） |
| `/version` | 显示版本信息 |
| `/alias` | 列出命令别名 |
| `/alias add <触发词> <命令>` | 添加别名（如：帮助 → /help） |
| `/alias del <触发词>` | 删除别名 |
| `/commands` | 自定义斜杠命令管理（添加/删除） |

---

## 九、权限请求与内联按钮

当 Agent 需要执行敏感操作（如运行命令、写文件）时，cc-connect 会向用户请求权限。

### 9.1 在 Telegram 中的表现

- 会发送一条消息，描述当前请求（如工具名、简要输入），并附带 **Inline Keyboard**：
  - **Allow**：允许本次请求
  - **Deny**：拒绝
  - **Allow All**：本会话后续请求全部自动允许

### 9.2 操作方式

- 点击对应按钮即可；点击后消息会更新为「✅ Allowed」或「❌ Denied」等，并通知 Agent。
- 部分场景下也可在聊天中**直接回复文字**：`allow` / `deny` / `allow all`（视引擎实现而定）。

---

## 十、流式预览与打字状态

- **流式预览**：Agent 输出较长时，cc-connect 可能先发一条「预览」消息，并随内容增加**原地编辑**该条消息（Telegram 支持 Edit Message），避免刷屏。
- **打字状态**：在等待 Agent 首次回复期间，Telegram 端会显示「typing…」状态，直到首段内容发出。

二者均由 cc-connect 与 Telegram 适配层自动处理，无需额外配置。

---

## 十一、Bot 命令菜单（可选）

cc-connect 支持将常用斜杠命令注册到 Telegram 的 **Bot 命令菜单**（用户点击输入框旁的「/」可见）。

- 若平台实现了 `CommandRegistrar`，启动时会自动调用 Telegram 的 `setMyCommands`。
- 你也可以在 **BotFather** 中手动设置：
  1. 发送 `/setcommands`
  2. 选择你的机器人
  3. 输入命令列表，例如：
     ```
     help - 显示帮助
     new - 新建会话
     list - 列出会话
     mode - 查看/切换权限模式
     ```

Telegram 要求命令名仅含小写字母、数字和下划线，且长度 1～32 字符；不符合的斜杠命令可能不会出现在菜单中，但仍可正常输入使用。

---

## 十二、常见问题与排查

### 12.1 机器人不回复消息

- 确认 **cc-connect 进程正在运行**，且日志中有 `telegram: connected`。
- 确认 **token 正确**（无多余空格、复制完整）。
- **群聊**：确认已关闭 BotFather 中的 **Group Privacy**，且消息是 @ 机器人或回复机器人的。
- 若在**代理环境**下，确认 `proxy` 配置正确，且本机通过该代理能访问 `api.telegram.org`。

### 12.2 群内发了 @ 机器人但仍无反应

- 检查是否有多余空格或错误拼写，@ 后必须紧跟机器人用户名。
- 查看 cc-connect 日志是否有 `telegram: ignoring group message not directed at bot` 等调试信息，便于确认过滤逻辑。

### 12.3 语音消息没有转成文字

- 确认全局配置中启用了 `[speech]`，并配置了有效的 API（如 OpenAI/Whisper）。
- 确认本机已安装 **ffmpeg**（语音格式转换需要）。
- 查看日志是否有 speech/STT 相关报错。

### 12.4 图片发出去没反应

- 确认当前项目使用的 **Agent 支持多模态**（如 Claude Code）；不支持多模态的 Agent 可能忽略图片或报错。
- 查看 cc-connect 日志是否有 `telegram: download photo failed` 等错误。

### 12.5 如何重新生成 Token

1. 在 BotFather 发送 `/mybots`
2. 选择你的机器人 → **API Token** → **Revoke current token**
3. 将新 Token 更新到 `config.toml` 的 `token`，重启 cc-connect。

### 12.6 代理下连接失败

- 确认 `proxy` 为完整 URL，如 `http://127.0.0.1:7890`。
- 若代理需认证，填写 `proxy_username` 与 `proxy_password`。
- 用浏览器或 `curl` 通过同一代理访问 `https://api.telegram.org` 验证网络可达性。

---

## 十三、参考链接

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [BotFather 与创建机器人](https://core.telegram.org/bots#botfather)
- [cc-connect README（语音/多模态/Provider 等）](../README.md)
- [快速接入步骤（英文）](./telegram.md)

---

## 十四、小结

| 步骤 | 要点 |
|------|------|
| 创建机器人 | 在 @BotFather 用 `/newbot` 获取 Token |
| 配置 | 在 `config.toml` 的 `[[projects.platforms]]` 中 `type = "telegram"`，必填 `token`；按需配置 `allow_from`、`proxy`、`group_reply_all` 等 |
| 启动 | 执行 `cc-connect`，看到 `telegram: connected` 即表示就绪 |
| 私聊 | 直接发文字/图片/语音 |
| 群聊 | 关闭 Group Privacy，并 @ 机器人或回复机器人消息 |
| 权限 | 通过 Inline 按钮或文字回复 allow/deny/allow all |
| 进阶 | 斜杠命令管理会话/模式/Provider/定时任务；多机器人时可用 `/bind` 与 `group_reply_all` |

按上述步骤即可在项目中完整使用 Telegram 平台。若需扩展更多平台或 Agent，可参考 [README - 扩展](../README.md#extending) 与 [AI_CONTEXT.md](./AI_CONTEXT.md)。
