# AGENTS.md

## Agent Operating Guide

This repository is designed to be operated by software agents and human developers.

The goal of this document is to provide **a minimal entry point** for understanding the repository.

Detailed documentation exists elsewhere.

---

## 1. Start Here

Before making changes read:

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [docs/VALIDATION.md](docs/VALIDATION.md)
- [docs/TASK_RECIPES.md](docs/TASK_RECIPES.md)
- [docs/generated/repo-index.json](docs/generated/repo-index.json)

These documents define:

- system architecture
- task workflows
- validation procedures
- repository structure

Agents should prefer **structured metadata** (`repo-index.json`) when available, then follow links into the corresponding source documents.

---

## 2. Standard Workflow

For any non-trivial change follow this workflow:

Context → Plan → Implement → Validate → Finalize

### Context

Understand the repository by reading:

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [docs/generated/repo-index.json](docs/generated/repo-index.json)
- existing execution plans in [`docs/plans/`](docs/plans/) or [`docs/exec-plans/active/`](docs/exec-plans/active/)

---

### Plan

Create a plan in:

[`docs/exec-plans/active/`](docs/exec-plans/active/)

Plan must include:

- objective
- affected modules
- approach
- validation strategy
- rollback plan

---

### Implement

When modifying code:

- keep changes minimal
- avoid modifying unrelated modules
- respect architecture boundaries

---

### Validate

Validation rules are defined in:

[`docs/VALIDATION.md`](docs/VALIDATION.md)

Typical validation:

- build
- unit tests
- integration tests
- benchmark (if performance-sensitive)

---

### Finalize

After successful validation:

- update relevant documentation
- move plan to:

  [`docs/exec-plans/completed/`](docs/exec-plans/completed/)

---

## 3. Architectural Rules

Architectural constraints are defined in:

[`ARCHITECTURE.md`](ARCHITECTURE.md)

Core rules:

- no circular dependencies
- no cross-layer dependencies
- modules interact through public interfaces only

---

## 4. Performance Sensitive Modules

Some modules are performance critical.

Changes to these modules require:

- benchmark validation
- regression checks
- CPU/memory impact analysis

See [`docs/VALIDATION.md`](docs/VALIDATION.md).

---

## 5. Task Types

Tasks are categorized as:

bugfix
feature
refactor
perf
docs
analysis

Task workflows are defined in:

[`docs/TASK_RECIPES.md`](docs/TASK_RECIPES.md)

---

## 6. Repository Structure

Machine-readable repository structure:

[`docs/generated/repo-index.json`](docs/generated/repo-index.json)

Agents should use this file to understand modules and dependencies.

---

## 7. Safety Rules

Agents must follow these principles:

- minimal change
- validation first
- maintain architectural integrity
- ensure rollback capability

If uncertain, create a plan instead of guessing.

---
