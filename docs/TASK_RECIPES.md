# Task Recipes

This document maps common task types to the real files and checks used in this repository.

For deeper workflow guidance, also see [`docs/harness/workflows.md`](harness/workflows.md).

## Agent Task

Use when adding or modifying an agent integration.

Read first:

- `AGENTS.md`
- `ARCHITECTURE.md`
- `docs/AI_CONTEXT.md`

Typical files:

- `agent/<name>/`
- `core/interfaces.go`
- `core/registry.go`
- `cmd/cc-connect/main.go`
- `config.example.toml`

Minimum verification:

- `make check-harness`
- `go test ./...`
- `make build`

## Platform Task

Use when adding or modifying a platform integration.

Read first:

- `AGENTS.md`
- `ARCHITECTURE.md`
- `docs/AI_CONTEXT.md`

Typical files:

- `platform/<name>/`
- `core/interfaces.go`
- `core/registry.go`
- `cmd/cc-connect/main.go`
- platform docs under `docs/`

Minimum verification:

- `make check-harness`
- `go test ./...`
- `make build`

## Config Task

Use when changing config schema or persisted settings.

Read first:

- `AGENTS.md`
- `docs/VALIDATION.md`

Typical files:

- `config/config.go`
- `config.example.toml`
- `README.md`
- affected docs in `docs/`

Minimum verification:

- targeted tests for affected packages
- `go test ./...` if shared flow is affected

## Command or UX Text Task

Use when changing slash commands, help text, or user-visible messages.

Read first:

- `AGENTS.md`
- `docs/VALIDATION.md`

Typical files:

- `core/engine.go`
- `core/i18n.go`
- `README.md`

Minimum verification:

- targeted tests for `core/`
- review help and i18n text together

## Core Routing or Session Task

Use when changing shared runtime flow.

Read first:

- `ARCHITECTURE.md`
- `docs/AI_CONTEXT.md`
- relevant plan files in `docs/exec-plans/active/` or legacy references in `docs/plans/`

Typical files:

- `core/engine.go`
- `core/session.go`
- `core/message.go`
- `core/interfaces.go`

Minimum verification:

- targeted package tests
- `go test ./...`
- `make build`

## Docs or Analysis Task

Use when the task is documentation-only or investigation-only.

Read first:

- `AGENTS.md`
- relevant docs in `docs/`

Typical outputs:

- update docs only
- create a design doc under `docs/exec-plans/active/`
- create an implementation plan when the work will continue

Minimum verification:

- review paths and links
- run repo-wide checks if the docs change repository workflow or validation guidance
