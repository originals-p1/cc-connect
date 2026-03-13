# 技术债务追踪 | Tech Debt Tracker

本文档记录 repository governance 和代码质量的技术债务。

---

## 目录

| 中文 | English |
|------|---------|
| [当前未解决问题](#当前未解决问题) | [Current Gaps](#current-gaps) |
| [技术债务项](#技术债务项) | [Tech Debt Items](#tech-debt-items) |
| [改进计划](#改进计划) | [Improvement Plans](#improvement-plans) |

---

## 当前未解决问题

### 1. 文档组织

| 问题 | 优先级 | 所属文档 |
|------|--------|----------|
| `docs/plans/` 包含旧计划文档，已迁移到 `docs/exec-plans/` | medium | docs/plans/README.md (已归档) |
| 部分文档引用了过时的链接 | medium | docs/ |
| 缺少 `docs/SECURITY.md` | high | ✅ 已创建 |
| 缺少 `docs/CONTRIBUTING.md` | low | ✅ 已创建 |
| 缺少 `docs/AGENT_INTEGRATION.md` | low | ✅ 已创建 |
| 缺少 `docs/PLATFORM_INTEGRATION.md` | low | ✅ 已创建 |
| 缺少 `docs/AGENT_CONFIGURATION.md` | medium | ✅ 已创建 |
| 缺少 `docs/diagnosis.md` | medium | ✅ 已创建 |

### 2. 测试覆盖

| 问题 | 优先级 | 解决方案 |
|------|--------|----------|
| `core/` 模块测试不足 | high | 增加单元测试覆盖 |
| 部分 agent 缺少集成测试 | medium | 添加典型工作流测试 |
| 平台测试覆盖率不均 | high | 统一测试策略，确保每个平台有基本测试 |
| 缺少性能基准测试 | medium | 为性能敏感模块添加 benchmark |

### 3. 代码质量

| 问题 | 优先级 | 解决方案 |
|------|--------|----------|
| 错误处理不一致 | medium | 定义标准错误类型并统一使用 |
| 日志格式不统一 | low | 定义日志规范并执行 |
| 部分代码缺少注释 | medium | 为公共 API 添加 godoc 注释 |

### 4. CI/CD

| 问题 | 优先级 | 解决方案 |
|------|--------|----------|
| 缺少自动 lint 检查 | medium | 在 CI 中运行 golangci-lint |
| 缺少覆盖率报告 | medium | 生成并报告覆盖率 |
| 缺少安全扫描 | high | 添加 govulncheck 和依赖扫描 |
| 缺少文档链接检查 | low | 添加 markdown link checker |

### 5. 配置验证

| 问题 | 优先级 | 解决方案 |
|------|--------|----------|
| 配置示例未自动验证 | medium | 添加 config.example.toml schema 验证 |
| 缺少配置迁移工具 | high | 为 breaking changes 提供 migration script |
| 环境变量引用未验证 | medium | 验证 `${VAR}` 在运行时被正确替换 |

---

## 技术债务项

### debt-core-testing

- **描述**: `core/` 模块测试覆盖率不足
- **影响**: 高耦合性，难以重构
- **估计工作量**: 2-3 周
- **优先级**: high

#### 子任务

- [ ] `core/engine.go` - 增加路由测试
- [ ] `core/session.go` - 增加会话管理测试
- [ ] `core/message.go` - 增加消息处理测试
- [ ] `core/registry.go` - 增加注册表测试

### debt-platform-tests

- **描述**: 部分平台缺少测试
- **影响**: 平台兼容性风险
- **估计工作量**: 1-2 周
- **优先级**: high

#### 优先测试的平台

- [ ] `platform/telegram/` - 完善测试覆盖
- [ ] `platform/feishu/` - 完善测试覆盖
- [ ] `platform/slack/` - 添加基本测试
- [ ] `platform/line/` - 添加基本测试
- [ ] `platform/wecom/` - 添加基本测试
- [ ] `platform/qq/` - 添加基本测试
- [ ] `platform/qqbot/` - 添加基本测试

### debt-error-handling

- **描述**: 错误处理不一致
- **影响**: 难以调试和维护
- **估计工作量**: 3-5 天
- **优先级**: medium

#### 实施计划

1. 定义标准错误类型
2. 为常见错误创建 error variables
3. 重写现有错误处理
4. 添加错误处理测试

### debt-docs-links

- **描述**: 文档链接可能过时
- **影响**: 用户体验
- **估计工作量**: 1 天
- **优先级**: medium

#### 解决方案

```bash
# 检查文档链接
pip install markdown-link-check
find docs -name "*.md" -exec markdown-link-check {} \;
```

### debt-config-validation

- **描述**: 配置验证不足
- **影响**: 配置错误难以调试
- **估计工作量**: 1-2 周
- **优先级**: medium

#### 功能

- [ ] 添加 TOML schema 验证
- [ ] 提供配置示例验证脚本
- [ ] 添加 config.example.toml 自动测试

---

## 改进计划

### Q2 2026 改进计划

#### Month 1

| 目标 | 状态 | 优先级 |
|------|------|--------|
| 归档旧计划文档 | ✅ 已完成 | medium |
| 添加 `docs/SECURITY.md` | ✅ 已完成 | high |
| 添加 `docs/CONTRIBUTING.md` | ✅ 已完成 | low |
| 添加 `docs/AGENT_INTEGRATION.md` | ✅ 已完成 | low |
| 添加 `docs/PLATFORM_INTEGRATION.md` | ✅ 已完成 | low |
| 添加 `docs/AGENT_CONFIGURATION.md` | ✅ 已完成 | medium |
| 添加 `docs/diagnosis.md` | ✅ 已完成 | medium |
| 更新 `docs/VALIDATION.md` | ✅ 已完成 | medium |

#### Month 2-3

| 目标 | 状态 | 优先级 |
|------|------|--------|
| 增加 `core/` 单元测试 | pending | high |
| 统一 agent 平台测试策略 | pending | high |
| 定义标准错误类型 | pending | medium |
| 添加 CI 自动 lint | pending | medium |
| 添加文档链接检查 | pending | low |

### 长期改进

| 目标 | 优先级 |
|------|--------|
| 重构 `core/engine.go`，降低耦合性 | high |
| 添加性能基准测试框架 | medium |
| 实现配置自动迁移工具 | high |
| 添加安全扫描到 CI | high |

---

## 已解决的技术债务

| ID | 描述 | 解决状态 | 日期 |
|----|------|----------|------|
| debt-docs-plans | 归档 `docs/plans/` | ✅ 已解决 | 2026-03-14 |

---

## 贡献指南

如果你想要帮助解决技术债务：

1. 选择一个 debt item（如 `debt-core-testing`）
2. 阅读对应的子任务
3. 创建 PR 提交改进
4. 在 PR 中链接此文档

### 快速开始

```bash
# 查看当前债务
grep -r "debt-" docs/

# 选择一个债务项
# 例如：debt-core-testing
```

---

**最后更新**: 2026-03-14
