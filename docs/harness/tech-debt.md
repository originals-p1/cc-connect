# Harness Tech Debt

This file tracks repository-governance debt, not feature backlog.

## Current Gaps

- The repository only recently gained a dedicated `docs/harness/` layer; older docs may still point directly to deeper references without a stable working entrypoint.
- Most repository rules still live in documentation rather than automated checks.
- legacy plan history still exists under `docs/plans/`, while current execution plans now live under `docs/exec-plans/`.
- Documentation drift detection is mostly manual.
- Repo-health signals are spread across `AGENTS.md`, `README.md`, plans, and maintainer knowledge.

## Near-Term Follow-Ups

- extend `check-harness` only after each new rule is concrete and low-noise
- decide whether legacy `docs/plans/` history should be migrated, archived, or left read-only
- consider a lightweight docs link checker if current markdown volume keeps growing
- evaluate whether `/doctor` should eventually expose repository-health checks in addition to runtime diagnostics
