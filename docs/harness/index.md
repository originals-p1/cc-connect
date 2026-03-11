# Repository Harness

This directory is the stable working layer for coding agents and contributors.

Use it to answer three questions quickly:

- where to look first
- what files usually change together
- what checks prove the change is safe

## Reading Order

1. `AGENTS.md` — concise execution rules and task-to-file mapping
2. `docs/harness/index.md` — working entrypoint and doc map
3. `docs/harness/workflows.md` — common change paths
4. `docs/harness/quality-gates.md` — verification and sync rules
5. `docs/AI_CONTEXT.md` — deeper architecture and domain context
6. `docs/exec-plans/active/` — current design and execution plans
7. `docs/plans/` — legacy plan history that has not been migrated

## Document Roles

- `AGENTS.md`: short execution guide; do not turn it into a full manual
- `docs/AI_CONTEXT.md`: stable architecture map
- `docs/harness/`: stable working guidance for repository changes
- `docs/exec-plans/`: current active/completed execution-plan layout
- `docs/plans/`: legacy plan history

## First-Stop Paths

- Add or modify an agent:
  Read `AGENTS.md`, then `docs/harness/workflows.md`, then `docs/AI_CONTEXT.md`
- Add or modify a platform:
  Read `AGENTS.md`, then `docs/harness/workflows.md`, then `docs/AI_CONTEXT.md`
- Change config shape:
  Read `AGENTS.md`, then `docs/harness/quality-gates.md`
- Change commands or user-facing messages:
  Read `AGENTS.md`, then `docs/harness/workflows.md`, then `docs/harness/quality-gates.md`
- Change shared routing or session behavior:
  Read `docs/AI_CONTEXT.md` first, then review the relevant plan in `docs/exec-plans/active/` or legacy history in `docs/plans/`

## Maintenance Intent

Keep this directory stable, high-signal, and low-entropy.

- Prefer links over duplicated explanations
- Keep workflow steps concrete
- Update these docs when repository rules change
