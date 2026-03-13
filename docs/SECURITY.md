# 安全与权限指南 | Security & Permissions

本项目注重安全性和隐私保护。本文档介绍安全最佳实践、敏感信息处理和权限管理。

---

## 表格目录 | Table of Contents

| 中文 | English |
|------|---------|
| [安全最佳实践](#安全最佳实践) | [Security Best Practices](#security-best-practices) |
| [敏感信息处理](#敏感信息处理) | [Sensitive Data Handling](#sensitive-data-handling) |
| [Agent 权限模式](#agent-权限模式) | [Agent Permission Modes](#agent-permission-modes) |
| [API 密钥管理](#api-密钥管理) | [API Key Management](#api-key-management) |
| [网络安全建议](#网络安全建议) | [Network Security Recommendations](#network-security-recommendations) |

---

## 安全最佳实践

### 1. 配置文件安全

#### ✅ 正确做法

**使用环境变量存储敏感信息：**

```bash
# 在 config.toml 中引用环境变量
[[projects.agent.providers]]
name = "anthropic"
api_key = "${ANTHROPIC_API_KEY}"

# 或在启动时设置
export ANTHROPIC_API_KEY="sk-xxx"
export OPENAI_API_KEY="sk-xxx"
cc-connect
```

**保护配置文件权限：**

```bash
# Linux/macOS
chmod 600 ~/.cc-connect/config.toml
ls -la ~/.cc-connect/config.toml
# -rw------- 1 user user config.toml
```

#### ❌ 错误做法

```toml
# ❌ 禁止将 API 密钥硬编码在配置中
[[projects.agent.providers]]
name = "anthropic"
api_key = "sk-ant-xxx"  # 危险！不要提交到 Git
```

---

### 2. 配置文件位置

cc-connect 按以下顺序查找配置文件：

1. **显式指定**：`cc-connect -config /path/to/config.toml`
2. **当前目录**：`./config.toml`（不推荐用于敏感配置）
3. **全局配置**：`~/.cc-connect/config.toml`（**推荐**）

```bash
# 推荐：使用全局配置目录
mkdir -p ~/.cc-connect
chmod 700 ~/.cc-connect
cp config.example.toml ~/.cc-connect/config.toml
chmod 600 ~/.cc-connect/config.toml
```

---

## 敏感信息处理

### 环境变量 vs 配置文件

| 方式 | 优点 | 缺点 | 推荐场景 |
|------|------|------|----------|
| **环境变量** | 不写入磁盘，进程退出即消失 | Shell 历史可能记录 | 临时密钥、CI/CD |
| **配置文件** | 便于版本管理（需要忽略敏感字段） | 需要正确设置文件权限 | 本地长期使用 |

### 安全环境变量设置

```bash
# ❌ 错误：直接在命令行设置（会记录到 shell 历史）
cc-connect -config <(cat ~/.cc-connect/config.toml | ANTHROPIC_API_KEY="sk-xxx")

# ✅ 正确：从文件或交互式输入读取
source ~/.api-keys.sh  # 包含 export ANTHROPIC_API_KEY=xxx
cc-connect
```

---

## Agent 权限模式

### 模式选择指南

所有 Agent 都支持权限模式切换（通过 `/mode` 命令或启动参数）：

| 模式 | 适合场景 | 风险等级 | 推荐 |
|------|---------|---------|------|
| **default** | 正常开发、不信任环境 | ⭐ 低 | ✅ 推荐 |
| **acceptEdits/edit** | 信任环境、接受文件编辑 | ⭐⭐ 中 | ✅ 推荐 |
| **plan** | 仅需分析、不允许执行 | ⭐ 低 | ✅ 推荐 |
| **yolo** | 完全信任的沙箱环境 | ⭐⭐⭐ 高 | ❌ 谨慎使用 |

### 模式详细说明

#### Default Mode（默认模式）

- **行为**：每次工具调用都要求用户批准
- **使用**：`mode = "default"` 或不设置
- **适合**：所有场景，默认推荐

#### Accept Edits Mode（接受编辑）

- **行为**：文件编辑工具自动批准，其他工具仍需批准
- **使用**：`mode = "acceptEdits"` 或 `mode = "edit"`
- **适合**：信任环境，希望提高效率但保持安全性

#### Plan Mode（计划模式）

- **行为**：只允许读取操作，不允许任何修改
- **使用**：`mode = "plan"`
- **适合**：代码审查、安全审计、只读分析

#### YOLO Mode（无限制模式）

- **行为**：所有工具调用自动批准，无需用户干预
- **使用**：`mode = "yolo"`
- **⚠️ 警告**：
  - 仅在完全信任的沙箱环境中使用
  - 确保工作目录有备份
  - 不要在生产环境或重要项目中使用
  - 建议配合 `allowed_tools` 限制可用工具

---

## API 密钥管理

### 支持的 Agent API 密钥映射

| Agent | api_key → 环境变量 | base_url → 环境变量 |
|-------|-------------------|---------------------|
| Claude Code | `ANTHROPIC_API_KEY` | `ANTHROPIC_BASE_URL` |
| Codex | `OPENAI_API_KEY` | `OPENAI_BASE_URL` |
| Gemini CLI | `GEMINI_API_KEY` | N/A |
| Qoder CLI | `QODER_API_KEY` | `QODER_BASE_URL` |
| OpenCode | `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY` 等 | `AZURE_OPENAI_ENDPOINT` / `LOCAL_ENDPOINT` |
| iFlow CLI | `IFLOW_API_KEY` / `IFLOW_apiKey` | `IFLOW_BASE_URL` / `IFLOW_baseUrl` |

### 安全配置示例

```toml
# 使用环境变量（推荐）
[[projects.agent.providers]]
name = "anthropic"
env = { ANTHROPIC_API_KEY = "${ANTHROPIC_API_KEY}" }

# 或使用自定义环境变量
[[projects.agent.providers]]
name = "custom"
env = { 
  API_KEY = "${MY_CUSTOM_API_KEY}",
  BASE_URL = "https://api.example.com" 
}
```

---

## 网络安全建议

### 1. Webhook 平台（需公网 IP）

对于 LINE、WeChat Work 等需要公网 URL 的平台：

```bash
# ✅ 正确：使用反向代理工具
# 选项 A: Cloudflare Tunnel（推荐）
cloudflared tunnel --url http://localhost:8080

# 选项 B: ngrok
ngrok http 8080

# 选项 C: frp（自己服务器）
# 参考: https://github.com/fatedier/frp
```

#### ❌ 禁止的做法

```bash
# ❌ 不要直接暴露端口到公网
# 不使用防火墙或反向代理，直接开放 8080 端口
```

### 2. 防火墙配置

```bash
# Linux (ufw 示例)
sudo ufw allow 8080/tcp           # 仅允许 cc-connect 端口
sudo ufw deny 8081/tcp            # 拒绝其他端口
sudo ufw enable

# macOS (application firewall)
# 系统设置 → 安全性与隐私 → 防火墙
```

### 3. HTTPS 强制使用

对于 Webhook 平台，始终使用 HTTPS：

```
✅ 正确：https://your-domain.com/callback
❌ 错误：http://your-domain.com/callback
```

使用 Cloudflare Tunnel 或 ngrok 自动提供 HTTPS。

---

## 安全检查清单

### 配置前检查

- [ ] API 密钥未硬编码在 `config.toml` 中
- [ ] 使用 `~/.cc-connect/` 而非当前目录存储配置
- [ ] 配置文件权限设置为 `600`（仅所有者可读写）
- [ ] 目录权限设置为 `700`（仅所有者可访问）

### 运行时检查

- [ ] Agent 权限模式适合当前环境（默认推荐 `default` 或 `acceptEdits`）
- [ ] Webhook 平台使用 HTTPS URL
- [ ] 敏感操作（如 `yolo` 模式）仅在沙箱环境使用
- [ ] 定期审查 `~/.cc-connect/sessions/` 中的会话历史

### 维护检查

- [ ] 定期更新 cc-connect 到最新版本
- [ ] 定期轮换 API 密钥
- [ ] 检查 `cc-connect logs` 中的异常行为
- [ ] 备份配置文件和会话数据

---

## 安全事件响应

### 怀疑密钥泄露

1. **立即轮换密钥**：
   ```bash
   # 访问你的 API 提供商控制台
   # 生成新密钥
   # 更新配置
   ```

2. **撤销旧密钥**：
   - 在 API 提供商控制台禁用旧密钥
   - 检查是否有未授权使用

3. **审查访问记录**：
   ```bash
   # 查看 cc-connect 日志
   cc-connect daemon logs --since 24h
   ```

### 遭遇攻击

1. **停止服务**：
   ```bash
   cc-connect daemon stop
   ```

2. **分析日志**：
   ```bash
   cc-connect daemon logs --since 7d | grep -i "error\|unauthorized"
   ```

3. **报告问题**：
   - GitHub Issues: https://github.com/chenhg5/cc-connect/issues
   - 或发送邮件到项目维护者

---

## 依赖安全

### 更新依赖

```bash
# 检查已知漏洞
go list -m -u all
go get -u ./...
go mod tidy
```

### 审计命令

```bash
# 运行安全审计（如果使用 go-audit 或类似工具）
go run github.com/sonatype-nexus-community/nancy@latest check ./...
```

---

## 隐私说明

### 收集的数据

cc-connect **不收集任何用户数据**。所有数据本地处理：

- ✅ 本地存储：配置文件、会话历史
- ✅ 本地处理：消息路由、Agent 通信
- ✅ 加密传输：与平台的 WebSocket/HTTPS 连接

### 第三方服务

当使用以下功能时，数据会发送到第三方：

| 功能 | 第三方 | 数据类型 | 加密 |
|------|--------|----------|------|
| STT (语音转文字) | OpenAI/Groq/Qwen | 音频文件 | ✅ HTTPS |
| TTS (文字转语音) | Qwen/OpenAI | 文本内容 | ✅ HTTPS |
| AI Agent | Anthropic/OpenAI/Google | 对话内容 | ✅ HTTPS |

### 合规性

- **GDPR**：所有数据本地处理，用户数据不离开您的设备
- **CCPA**：无数据销售，无数据共享
- **SOC 2**：企业用户可自行部署，完全控制数据

---

## 安全更新

### 订阅安全公告

- GitHub Security Advisories: https://github.com/chenhg5/cc-connect/security
- GitHub Releases (包含安全修复): https://github.com/chenhg5/cc-connect/releases

### 报告安全漏洞

我们重视安全！发现漏洞请通过以下方式报告：

1. **GitHub Security Advisories**（推荐）：
   - 访问 https://github.com/chenhg5/cc-connect/security
   - 点击 "Report a vulnerability"

2. **加密邮件**（可选）：
   ```
   pgp: 公钥请在 GitHub Profile 中查看
   ```

#### 报告内容

- 漏洞类型（如：RCE、信息泄露、SSRF）
- 复现步骤
- 潜在影响
- 修复建议（如有）

我们会及时响应，通常在 48 小时内给出初步回复。

---

## 参考资源

### 通用安全

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE/SANS Top 25](https://cwe.mitre.org/top25/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cybersecurity)

### Go 安全

- [Go Security Best Practices](https://github.com/golang/security)
- [Awesome Go Security](https://github.com/ariankorp/awesome-go-security)

### API 安全

- [OWASP API Security Top 10](https://owasp.org/www-api-security/)
- [GitHub REST API Best Practices](https://docs.github.com/en/rest/overview/best-practices-for-using-the-rest-api)

---

## 贡献指南

欢迎贡献安全相关改进！

### 提交安全 PR

- 在 PR 标题添加 `[SECURITY]` 前缀
- 详细说明安全影响和修复方案
- 提供测试用例
- 更新相关文档

### 安全相关 Issue

- 标签：`security`, `bug`, `high-priority`
- 优先级：高（通常 48 小时内响应）

---

**最后更新**: 2026-03-14

**维护者**: cc-connect Team

** License**: MIT (请同时参考项目根目录的 LICENSE 文件)
