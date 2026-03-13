# OpenCode 集成改进计划

**创建日期**: 2026-03-13  
**状态**: 实施中  
**任务类型**: feature / analysis

## Objective

在不破坏现有 OpenCode 会话、relay 和命令系统行为的前提下，补齐 OpenCode 集成中确认存在的缺口，并把未证实或跨模块影响较大的想法从本计划中拆出。

本计划只覆盖以下两类工作：

- OpenCode agent 局部增强
- 已有能力的验证与补测

本计划明确不直接覆盖：

- `core/command.go` 的递归命令扫描改造
- 未经证实的 OpenCode session ID 格式校验
- 未经确认的 provider env 变量命名扩展

## Affected Modules

- `agent/opencode/opencode.go`
- `agent/opencode/session.go`
- `agent/opencode/opencode_test.go`
- `agent/opencode/session_test.go`
- 只读参考:
  - `core/interfaces.go`
  - `core/engine.go`
  - `core/skill.go`

## Background

当前 OpenCode 集成已经具备以下能力：

- 已实现 `SessionEnvInjector`
- engine 已在 interactive session 和 relay 场景注入 `CC_PROJECT` / `CC_SESSION_KEY`
- `AvailableModels()`、`CommandDirs()`、memory file 路径、provider 基础映射均已存在

因此，本计划不再把这些能力误判为“缺失实现”，而是聚焦以下已确认问题：

1. OpenCode 尚未实现 `SkillProvider`，导致 `/skills` 无法发现 OpenCode 目录下的 skills。
2. OpenCode provider 环境变量映射可能不完整，但扩展项必须先以 OpenCode 实际支持的 env 约定为准。
3. 现有 `SessionEnvInjector` 链路虽然代码已接通，但仍需要通过测试确认 cron / relay 场景行为。

## Approach

### 1. 补齐 `SkillProvider`

在 `agent/opencode/opencode.go` 中实现 `SkillDirs() []string`，目录策略与 `CommandDirs()` 保持一致：

- `<workdir>/.opencode/skills`
- `$XDG_CONFIG_HOME/opencode/skills`
- `~/.config/opencode/skills`
- `~/.opencode/skills`

要求：

- 复用现有 `uniqueStrings()` 去重
- 不改动 `core/skill.go` 的扫描语义
- 不引入跨 agent 的共享 skill 行为

### 2. 验证并补测 `SessionEnvInjector` 现有链路

不修改接口设计，只验证当前链路是否正确：

- `core.Engine` 在 interactive session 启动前调用 `SetSessionEnv()`
- `core.Engine` 在 relay session 启动前调用 `SetSessionEnv()`
- `agent/opencode.Agent.StartSession()` 确实把 `sessionEnv` 合并进子进程环境

优先补充单元测试，必要时增加回归测试，避免未来重构时链路失效。

### 3. 审慎扩展 provider env 映射

仅在确认 OpenCode CLI 实际支持的前提下，扩展 `providerEnvForOpenCode()`。

本项的接受标准：

- 只添加有明确定义的 env 变量
- 为每个新增映射补充单元测试
- 不引入基于猜测的 `Thinking` 或 `BaseURL` 映射

如果 OpenCode 文档无法确认某项 env 约定，则该项从本计划移出，不在本次实现。

## Non-Goals

以下内容暂不纳入本计划：

- 为 OpenCode 实现 `SystemPromptSupporter`
  - 当前 engine 仅按是否实现接口分支，不按返回值分支；直接实现该接口会改变现有 relay setup 提示行为。
- 修改 `core/command.go` 以支持命令子目录递归扫描
  - 该改动影响所有 `CommandProvider`，应单独立项。
- 为 `handleStepStart()` 增加基于 `ses_` 前缀的 session ID 校验
  - 当前没有足够证据证明 OpenCode session ID 格式固定。
- 把 `AvailableModels()` 改造成动态 API 拉取
  - 在缺少稳定 OpenCode 模型发现接口前，维持静态回退更稳妥。

## Validation Strategy

最低验证：

```bash
go test ./agent/opencode/...
go test ./core/...
```

若本次实现涉及 shared flow、用户可见行为或 `core/` 改动，则升级为：

```bash
make check-harness
go test ./...
make build
```

人工检查：

- `/skills` 能列出 `.opencode/skills/<name>/SKILL.md`
- relay 路径下 OpenCode 子进程能拿到 `CC_PROJECT` 和 `CC_SESSION_KEY`
- interactive session 路径下 OpenCode 子进程能拿到 `CC_PROJECT` 和 `CC_SESSION_KEY`
- provider 切换后，确认新增 env 映射只在已验证 provider 上生效

## Rollback Plan

若实现后出现回归，按以下顺序回滚：

1. 移除 `agent/opencode/opencode.go` 中新增的 `SkillDirs()` 实现
2. 回退 `providerEnvForOpenCode()` 的新增 env 映射
3. 删除对应新增测试，恢复到当前稳定行为

本计划不修改 engine 的 system prompt 分支逻辑，因此无需为该部分设计回滚。

## Implementation Steps

### Phase 1: 明确且低风险的 agent 局部增强 (已完成)

- [x] 为 OpenCode 实现 `SkillProvider`
- [x] 为 skills 目录发现补充测试

### Phase 2: 现有能力验证与补测

- [ ] 为 `SetSessionEnv()` 注入链路补充测试
- [ ] 验证 relay / interactive session 两条路径的环境变量传播

### Phase 3: 条件性 provider env 扩展

- [ ] 确认 OpenCode 文档或现有 CLI 约定
- [ ] 仅实现已证实的 env 映射
- [ ] 为新增映射补充测试

## Current Progress

本轮已落地内容：

- OpenCode 已实现 `SkillProvider`
- 已补充 skills 目录发现测试
- 已修复 late-text 场景下过早发送 `EventResult` 的问题
- 已移除错误的 `SystemPromptSupporter` 实现，避免影响 relay setup 提示

本轮验证记录：

- `go test ./agent/opencode/...`
- `go test ./core/...`

## Deferred Follow-Ups

如果后续确认 OpenCode 官方确实支持以下能力，应另开独立计划：

- `core/command.go` 递归扫描命令子目录
- 更可靠的 OpenCode 模型发现机制
- engine 对 `SystemPromptSupporter` 返回值的显式判断，而不只是接口存在性判断

## References

- `AGENTS.md`
- `ARCHITECTURE.md`
- `docs/VALIDATION.md`
- `docs/TASK_RECIPES.md`
- `core/interfaces.go`
- `core/engine.go`
- `core/skill.go`
