# 历史计划归档 | Archived Plans

此目录包含已完成的计划文档。新的执行计划请查看 [`docs/exec-plans/active/`](./../exec-plans/active/)。

---

## 说明 | Notes

| 中文 | English |
|------|---------|
| **归档原因** | **Archived Reason** |
| 这些是旧计划文档，功能已实现或计划已过期 | These are old plan documents, features implemented or plans outdated |

---

## 归档计划列表 | Archived Plans

| 日期 | 标题 | 状态 |
|------|------|------|
| 2026-03-11 | [任务命令设计](./2026-03-11-task-command-design.md) | ✅ 已完成 |
| 2026-03-11 | [任务命令](../exec-plans/completed/2026-03-11-task-command.md) | ✅ 已完成 |
| 2026-03-11 | [自动压缩重试设计](./2026-03-11-auto-compress-retry-design.md) | ✅ 已完成 |
| 2026-03-11 | [自动压缩重试](../exec-plans/completed/2026-03-11-auto-compress-retry.md) | ✅ 已完成 |
| 2026-03-10 | [动态项目管理](./2026-03-10-dynamic-project-management-notes.md) | ✅ 已完成 |
| 2026-03-10 | [项目切换 MVP 实现](./2026-03-10-project-switching-mvp-implementation.md) | ✅ 已完成 |
| 2026-03-10 | [Clear 命令设计](./2026-03-10-clear-command-design.md) | ✅ 已完成 |
| 2026-03-10 | [Clear 命令](./2026-03-10-clear-command.md) | ✅ 已完成 |
| 2026-03-10 | [Bot 模式协作设计](./2026-03-10-bot-mode-review-collab-design.md) | ✅ 已完成 |

---

## 迁移指南 | Migration Guide

### 从旧计划迁移到新计划

如果你发现这里有一个计划需要重新激活：

1. 检查功能是否已实现
2. 如果需要重新实现，创建新计划到 `docs/exec-plans/active/`
3. 在新计划中引用旧计划 URL

### 脚本自动归档

```bash
# 将旧计划移动到 archive
cd docs/plans
mkdir -p archive

for f in *.md; do
    if [[ "$f" =~ ^2026-03-.*\.md$ ]]; then
        mv "$f" archive/
    fi
done
```

---

## 相关目录 | Related Directories

| 目录 | 描述 |
|------|------|
| `docs/exec-plans/active/` | 当前活跃的执行计划 |
| `docs/exec-plans/completed/` | 已完成的执行计划 |
| `docs/harness/` | Repository 治理文档 |

---

**维护者**: cc-connect Team

**最后更新**: 2026-03-14
