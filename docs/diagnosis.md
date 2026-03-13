# 诊断指南 | Diagnosis Guide

本文档介绍如何诊断 cc-connect 的常见问题。

---

## 目录

| 中文 | English |
|------|---------|
| [快速诊断](#快速诊断) | [Quick Diagnosis](#quick-diagnosis) |
| [启动问题](#启动问题) | [Startup Issues](#startup-issues) |
| [运行时问题](#运行时问题) | [Runtime Issues](#runtime-issues) |
| [Agent 问题](#agent-问题) | [Agent Issues](#agent-issues) |
| [平台问题](#平台问题) | [Platform Issues](#platform-issues) |
| [命令参考](#命令参考) | [Command Reference](#command-reference) |

---

## 快速诊断

### 1. 检查服务状态

```bash
# 检查进程是否运行
ps aux | grep cc-connect

# 检查端口占用
lsof -i:8080

# 检查守护进程状态
cc-connect daemon status
```

### 2. 查看日志

```bash
# 查看最近日志
cc-connect daemon logs

# 实时查看日志
cc-connect daemon logs -f

# 查看最后 100 行
cc-connect daemon logs -n 100

# 查看过去 24 小时日志
cc-connect daemon logs --since 24h
```

### 3. 检查配置

```bash
# 查看配置
cc-connect config-example

# 验证配置格式
cat ~/.cc-connect/config.toml | grep -v "^#" | grep -v "^$"
```

---

## 启动问题

### 问题 1: 无法启动服务

#### 症状
```bash
$ cc-connect
FATAL: cannot start: error loading config
```

#### 可能原因
1. 配置文件不存在
2. 配置文件格式错误
3. 权限不足

#### 解决方法

```bash
# 1. 检查配置文件
ls -la ~/.cc-connect/config.toml

# 2. 创建配置文件
cp config.example.toml ~/.cc-connect/config.toml
chmod 600 ~/.cc-connect/config.toml

# 3. 验证配置格式
cat ~/.cc-connect/config.toml
```

### 问题 2: 代理端口被占用

#### 症状
```bash
ERROR: listen tcp :8080: bind: address already in use
```

#### 解决方法

```bash
# 1. 查找占用端口的进程
lsof -i:8080

# 2. 更改端口
# 在 config.toml 中修改 platform options
[projects.platforms.options]
port = 8081  # 改为其他端口

# 或停止占用进程
kill -9 <PID>
```

### 问题 3: 权限错误

#### 症状
```bash
$ cc-connect daemon install
ERROR: cannot create systemd service: permission denied
```

#### 解决方法

```bash
# Linux: 使用 sudo
sudo cc-connect daemon install --config ~/.cc-connect/config.toml

# macOS: 确保有写入权限
sudo chown -R $(whoami) ~/.cc-connect
```

---

## 运行时问题

### 问题 1: 消息无响应

#### 检查步骤

1. **查看 Agent 进程**
   ```bash
   ps aux | grep claude
   ps aux | grep codex
   ```

2. **检查会话状态**
   ```bash
   /current  # 在聊天中查看当前会话
   ```

3. **查看日志**
   ```bash
   cc-connect daemon logs -f | grep -i "error\|failed"
   ```

#### 解决方法

```bash
# 重启 Agent 会话
/new

# 或完全重启服务
cc-connect daemon stop
cc-connect daemon start
```

### 问题 2: 自动重连失败

#### 症状
```log
ERROR: WebSocket connection lost
ERROR: reconnect failed
```

#### 解决方法

```bash
# 检查网络连接
ping example.com

# 检查防火墙
sudo ufw status

# 重启服务
cc-connect daemon restart
```

### 问题 3: 内存使用过高

#### 检查

```bash
# 查看内存使用
ps aux | grep cc-connect | awk '{print $6/1024 " MB"}'

# 查看进程详细信息
top -p $(pgrep cc-connect)
```

#### 解决方法

```bash
# 减少会话数量
# 删除不需要的会话
/del <session_id>

# 减少缓存消息数量
# 在 config.toml 中配置
[display]
thinking_max_len = 100  # 减少消息长度
tool_max_len = 200
```

---

## Agent 问题

### Claude Code 问题

#### 问题: Claude Code 不响应

#### 检查

```bash
# 检查 Claude Code 是否安装
claude --version

# 检查 CLAUDECODE 环境变量
echo $CLAUDECODE

# 如果设置了，需要取消设置
unset CLAUDECODE
```

#### 解决

```bash
# 1. 确保 CLAUDECODE 未设置
unset CLAUDECODE

# 2. 重启服务
cc-connect daemon restart

# 3. 测试 Claude Code
claude --version
```

### Codex 问题

#### 问题: Session 文件找不到

#### 解决

```bash
# 查找 Codex session 文件
find ~ -name "*.codex.session" 2>/dev/null

# 检查 work_dir 路径
cat ~/.cc-connect/config.toml | grep work_dir
```

### OpenCode 问题

#### 问题: Session 状态异常

#### 检查

```bash
# 查看 OpenCode 进程
ps aux | grep opencode

# 检查 session 目录
ls -la ~/.opencode/sessions/
```

---

## 平台问题

### 飞书 (Feishu)

#### 问题: 长连接断开

#### 检查

```log
INFO: connected to wss://msg-frontier.feishu.cn/ws/v2
INFO: connection closed
```

#### 解决

```bash
# 1. 检查网络连接
ping msg-frontier.feishu.cn

# 2. 查看防火墙
sudo ufw status

# 3. 重启服务
cc-connect daemon restart
```

### Telegram

#### 问题: Long Polling 超时

#### 检查

```log
ERROR: Telegram API timeout
```

#### 解决

```bash
# 1. 检查网络连接
curl https://api.telegram.org

# 2. 更换网络
# 尝试使用 VPN 或切换网络

# 3. 增加超时时间
# 在 config.toml 中配置
[log]
level = "debug"  # 查看详细错误
```

### WeChat Work

#### 问题: Webhook 验证失败

#### 症状

```log
ERROR: WeChat Work callback verification failed
```

#### 解决

```bash
# 1. 启动 cc-connect 后再配置 webhook URL
# 2. 确保端口与配置一致
[projects.platforms.options]
port = 8080

# 3. 检查防火墙
sudo ufw allow 8080
```

---

## 命令参考

### 诊断命令

```bash
# 查看版本信息
cc-connect --version

# 查看配置示例
cc-connect config-example

# 查看守护进程状态
cc-connect daemon status

# 查看日志
cc-connect daemon logs [-f] [-n N] [--since duration]

# 重启守护进程
cc-connect daemon restart
```

### 会话命令

```bash
# 在聊天中使用以下命令

/current      # 查看当前会话
/list         # 列出所有会话
/new          # 创建新会话
/stop         # 停止当前执行
/help         # 查看帮助
```

### 日志命令

```bash
# 查看最近日志
cc-connect daemon logs

# 实时查看
cc-connect daemon logs -f

# 按时间范围
cc-connect daemon logs --since 1h
cc-connect daemon logs --since 24h

# 按行数
cc-connect daemon logs -n 50
cc-connect daemon logs -n 100
```

---

## 网络诊断

### 检查网络连接

```bash
# 测试到平台 API 的连接
curl -I https://api.openai.com
curl -I https://api.anthropic.com
curl -I https://open.feishu.cn

# 检查 HTTP 代理
echo $HTTP_PROXY
echo $HTTPS_PROXY

# 测试代理
curl -x http://localhost:8080 https://api.openai.com
```

### 端口检查

```bash
# 检查端口是否开放
netstat -tuln | grep 8080
ss -tuln | grep 8080

# 检查端口被哪个进程占用
lsof -i:8080
```

---

## 常见错误代码

### 错误 1: ECONNREFUSED

```
Error: connect ECONNREFUSED 127.0.0.1:8080
```

**原因**: 目标端口无服务监听

**解决**:
```bash
# 检查服务是否运行
ps aux | grep cc-connect

# 启动服务
cc-connect daemon start
```

### 错误 2: EHOSTUNREACH

```
Error: getaddrinfo EHOSTUNREACH
```

**原因**: DNS 解析失败或网络不可达

**解决**:
```bash
# 检查 DNS
nslookup api.openai.com

# 更换 DNS
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
```

### 错误 3: ETIMEDOUT

```
Error: connect ETIMEDOUT
```

**原因**: 连接超时，可能是网络问题

**解决**:
```bash
# 增加超时时间
# 在 config.toml 中配置
[log]
level = "debug"

# 检查网络连接
ping example.com
```

---

## 高级诊断

### 启用 Debug 模式

```toml
[log]
level = "debug"
```

### 分析日志

```bash
# 查看错误日志
cc-connect daemon logs | grep -i "error"

# 查看性能问题
cc-connect daemon logs | grep -i "slow\|timeout"

# 查看连接问题
cc-connect daemon logs | grep -i "connection\|reconnect"
```

### 生成诊断报告

```bash
#!/bin/bash
echo "=== cc-connect Diagnosis Report ==="
echo "Date: $(date)"
echo ""
echo "=== Version ==="
cc-connect --version
echo ""
echo "=== Daemon Status ==="
cc-connect daemon status
echo ""
echo "=== Recent Logs ==="
cc-connect daemon logs -n 50
echo ""
echo "=== Process List ==="
ps aux | grep cc-connect
echo ""
echo "=== Port Status ==="
netstat -tuln | grep 8080
```

---

## 联系支持

如果问题仍未解决：

1. [创建 GitHub Issue](https://github.com/chenhg5/cc-connect/issues)
2. 加入 Discord: https://discord.gg/kHpwgaM4kq
3. 加入 Telegram: https://t.me/+odGNDhCjbjdmMmZl

---

## 本地化支持

| 语言 | 平台 | 支持状态 |
|------|------|----------|
| 中文 | 飞书 | ✅ |
| 中文 | 钉钉 | ✅ |
| 中文 | 企业微信 | ✅ |
| 英文 | Telegram | ✅ |
| 英文 | Discord | ✅ |
| 英文 | Slack | ✅ |

---

**最后更新**: 2026-03-14
