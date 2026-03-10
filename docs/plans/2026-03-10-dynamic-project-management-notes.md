# 动态 Project 管理方案备忘

## 问题

这个项目是否可以支持一个 `main bot`，由它来管理 project 的增删改查？

## 当前状态

当前实现是“启动时固定”的：

- `cmd/cc-connect/main.go` 在进程启动时读取 `cfg.Projects`
- 每个 project 会创建一个 `core.Engine`
- 每个 engine 会创建自己的 agent 和 platform 实例
- 现有的 config reload 只会更新已经存在的 project 的部分运行时配置

这意味着：当前仓库**还不支持在运行时新增一个全新的 project**。

## 可行性判断

可以做，但 `main bot` 不能只是一个简单的配置文件编辑器。它背后需要有一层运行时 project 生命周期管理能力。

## 推荐方向

引入一个独立的管理型 bot，或者一个专门的 admin project，作为控制面入口。

建议提供的命令：

- `/project list`
- `/project add ...`
- `/project update ...`
- `/project remove <name>`
- `/project restart <name>`
- `/project reload <name>`

## 为什么 main bot 比单纯 config reload 更合理

- 能把“系统管理”与“普通聊天”分开
- 权限控制和审计更集中
- 避免让每个普通 project 都能修改全局运行状态
- 更适合承载 create / stop / restart / reconcile 这类生命周期操作

## 需要新增的运行时组件

这个能力需要一个 `ProjectManager` 或等价的运行时注册中心，负责：

- 加载和校验 project 定义
- 动态创建 agent 和 platform 实例
- 构造并启动 `core.Engine`
- 把新 engine 注册到 API、cron、relay 以及其它共享服务
- 删除 project 时停止并注销 engine
- 在部分启动失败时做回滚

## 设计选项

### 方案一：独立 Admin Bot

单独用一个 admin project 来管理其它所有 project。

优点：

- 职责边界最清晰
- 最容易做 admin-only 权限控制
- 运行安全边界最好

缺点：

- 需要额外配置一个 bot / project

结论：默认推荐这个方案。

### 方案二：任意 Bot 都能管理 Project

每个现有 project 都可以执行 project CRUD 命令。

优点：

- 不需要单独配置 admin bot

缺点：

- 安全边界最弱
- 权限模型更难收敛
- 更容易出现误操作或恶意全局修改

结论：不推荐。

### 方案三：指定一个现有 Bot 作为 Master

从现有普通 project 中指定一个 master bot，让它统一管理其它 project。

优点：

- 比新建 admin project 更省事
- 仍然保留一个统一管理入口

缺点：

- 业务聊天和系统管理耦合在一起
- 职责边界不如独立 admin bot 清楚

结论：可行，但没有方案一干净。

## 关键约束

- project 的增删不是简单改配置，而是运行时生命周期编排
- 新 project 必须一致地接入共享子系统
- 配置持久化必须保持原子性和可恢复性
- 所有管理命令都必须做权限控制
- 删除 / 重启流程必须避免留下孤儿 goroutine、socket 或 session

## 建议的下一步

如果要实现，这一阶段应该先完成下面 5 件事的设计：

1. `ProjectManager` 的职责和接口
2. admin 命令面
3. 运行时注册与注销流程
4. 配置持久化与失败回滚策略
5. 管理操作的鉴权模型

---

## 想法：一个 Bot 动态切换多个 Project

### 原始想法

配置一个项目路径列表。每个路径就是一个 project。用户可以在 bot 里查看所有 project，并动态切换当前激活的 project。

### 问题定义

不是把一个 bot 永久绑定到一个 project，而是让一个 bot 成为多 project 入口：

- 查看可用 project 列表
- 查看当前激活的 project
- 按 chat 或 session 动态切换当前 project

### 可行方案

#### 方案 A：Session 级 Active Project 指针

维护一个 project 路径和 engine 的注册表。每个聊天 session 保存一个 `active project` 指针。像 `/project list`、`/project switch <name>` 这类命令，只改变当前 session 的 active project。

优点：

- 用户体验最自然
- 一个 bot 可以服务多个代码库
- 切换时不需要重启 bot

缺点：

- session 路由会明显变复杂
- 现有“一个 bot 对应一个 engine”的假设需要调整
- session 持久化要额外保存 active project 状态

结论：如果多 project 是核心产品方向，这是最完整的方案。

#### 方案 B：Chat 级绑定

每个聊天或群组在任一时刻只绑定一个 project。切换时，影响的是整个 chat。

优点：

- 心智模型更简单
- 在群聊里更容易解释
- 比 session 级切换的路由复杂度更低

缺点：

- 灵活性较弱
- 多人共享一个群时，切换行为容易互相影响

结论：很适合作为团队 / 群聊场景下的第一版。

#### 方案 C：按需临时指定 Project

bot 平时不保存固定 active project。每次执行命令时显式指定 project，或者由 bot 临时询问用户选择哪个 project。

优点：

- 路由模型最简单
- 没有隐藏状态

缺点：

- 用户体验更啰嗦
- 不适合在同一个 repo 连续工作

结论：适合作为兜底模式，不太适合作为主交互方式。

### 推荐方向

如果要做，推荐分两步走：

1. 第一版先做 `chat` 级绑定切换，因为更简单、更稳
2. 后续如果需要更细粒度，再扩展到 `session` 级切换

这样可以先把第一版做得可理解、可运维，再逐步增加复杂度。

### 需要的运行时改动

- 维护已配置 project 路径及其元数据的注册表
- 引入独立于 bot 身份之外的 `active project` 概念
- 根据 active project 绑定结果来路由消息
- 持久化绑定关系，保证重启后可恢复
- 扩展 slash command 和 help 文案，支持 list / switch / current 等操作

### 风险与约束

- 当前 engine 模型是 project-centric，动态切换会穿透现有假设
- cron、relay、主动发送等能力可能都要显式增加 project 作用域
- 切换范围必须明确：到底是 chat、user 还是 session
- project 名称和路径规范化必须可预测、稳定

### 候选命令

- `/project list`
- `/project current`
- `/project switch <name>`
- `/project info <name>`

---

## 已收敛方案：切换型 MVP（私聊版）

### 目标

让一个 bot 在私聊中动态切换多个本地 project。

### 范围

- 只支持私聊
- 只支持 project 查看与切换
- project 来源于一个全局目录
- 目录下一层子目录中，包含 `.git` 的才算 project
- bot 的 agent 配置固定
- 切换时只改变 `work_dir`
- project 采用懒加载
- 保留最近少量 project 会话

### 配置模型

```toml
[workspace]
root = "/Users/you/code"
require_git = true

[[bots]]
name = "codex"
agent_type = "codex"
default_project = "repo-a"
max_cached_sessions = 3
dm_only = true

[bots.agent_options]
model = "gpt-5-codex"

[[bots.platforms]]
type = "telegram"
[bots.platforms.options]
token = "xxx"
```

### 配置语义

- `workspace.root` 决定“project 从哪里来”
- `bots` 决定“有哪些 bot”
- bot 的 `agent_type / model / provider / platform` 固定不变
- bot 切换 project 时，只更新当前 project 对应的 `work_dir`
- 所有 bot 都从同一个 `workspace.root` 扫描 project

### 运行时模型

需要至少引入以下几类状态：

#### 1. `ProjectCatalog`

保存扫描到的 project 列表。

每项至少包含：

- `name`
- `path`
- `git_root`
- `last_seen_at`

#### 2. `BotBinding`

保存某个用户在某个 bot 私聊窗口中的当前 active project。

键建议为：

- `platform`
- `user_id`
- `bot_id`

值建议为：

- `active_project_name`
- `switched_at`

#### 3. `ProjectRuntime`

表示某个 project 是否已经被当前 bot 懒加载。

至少包含：

- `project_name`
- `engine`
- `status`
- `last_used_at`

#### 4. `SessionCache`

保存最近保留的 project 会话，用于后续切回继续。

至少需要能按下面维度定位：

- `platform + user_id + bot_id + project_name`

### 命令面

第一版只保留以下 4 个命令：

- `/project list`
- `/project current`
- `/project switch <name>`
- `/project switch --fresh <name>`

### 交互语义

- 用户第一次私聊 bot 时，如果还没有 active project，则提示先 `/project list` 再 `/project switch <name>`
- 切换成功后，返回当前 project、路径以及本次是恢复旧会话还是新建会话
- 普通消息会自动路由到当前 active project，无需每条消息显式指定 project

### 消息路由

- bot 收到私聊消息
- 根据 `platform + user_id + bot_id` 查找 active project
- 若没有绑定，则提示用户先选 project
- 若已绑定但 runtime 未启动，则执行懒加载
- 之后把消息转发给该 project 对应的 engine 处理

### 推荐落地顺序

1. 增加 `workspace` 和 `bots` 配置结构
2. 实现 project 扫描器
3. 实现私聊级 active project 绑定存储
4. 改造 bot / engine 路由，支持按 active project 分发
5. 实现 `/project list`、`/project current`、`/project switch`
6. 增加最近使用会话缓存和淘汰策略
7. 补帮助文案、错误提示和使用说明

### 当前最大风险

- 现有实现大概率默认“一个 bot 对应一个固定 engine”
- 切换型方案要求变成“一个 bot 在运行时按 active project 路由到多个 engine”
- 这会是第一版改造中最核心、风险最高的一块
