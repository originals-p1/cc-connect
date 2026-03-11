# Repository Workflows

This document maps common repository changes to the files and checks they typically require.

## Add or Modify an Agent

Primary files:

- `agent/<name>/`
- `core/interfaces.go`
- `core/registry.go`
- `cmd/cc-connect/main.go`
- `config.example.toml`
- `README.md`
- `docs/AI_CONTEXT.md`

What to verify:

- agent registers itself in `init()`
- package is blank-imported in `cmd/cc-connect/main.go`
- config examples and docs match the supported options
- `go test ./...`
- `make build`

## Add or Modify a Platform

Primary files:

- `platform/<name>/`
- `core/interfaces.go`
- `core/registry.go`
- `cmd/cc-connect/main.go`
- `config.example.toml`
- `README.md`
- `docs/*.md`
- `docs/AI_CONTEXT.md`

What to verify:

- platform registers itself in `init()`
- package is blank-imported in `cmd/cc-connect/main.go`
- setup docs and support matrix stay aligned
- `go test ./...`
- `make build`

## Change Config Schema or Persisted Settings

Primary files:

- `config/config.go`
- `config.example.toml`
- `README.md`
- `docs/AI_CONTEXT.md`
- command- or feature-specific docs if behavior changed

What to verify:

- schema, validation, defaults, and persistence stay aligned
- example config reflects the new shape
- user-facing docs explain the new behavior
- targeted tests for `config/`
- `go test ./...` if the change affects shared flow

## Change Slash Commands or User-Facing Messages

Primary files:

- `core/engine.go`
- `core/i18n.go`
- `README.md`
- platform usage guides when command behavior changes

What to verify:

- help text and command behavior match
- all required language strings exist
- docs reflect new or removed commands
- targeted tests for `core/`

## Change Shared Routing, Sessions, or Event Flow

Primary files:

- `core/engine.go`
- `core/session.go`
- `core/message.go`
- `core/interfaces.go`
- related agent/platform packages
- `docs/AI_CONTEXT.md` if the structure changed materially

What to verify:

- check for regressions in session lifecycle, permission flow, and streaming
- run targeted tests for touched packages first
- run `go test ./...`
- run `make build`

## Change Agent-Facing Commands or Skills

Primary files:

- `core/command.go`
- `core/skill.go`
- relevant `agent/<name>/` package
- `AGENTS.md` or `docs/harness/*` if usage guidance changed

What to verify:

- command/skill discovery still resolves the expected files
- docs point agents to the right entrypoints
- targeted tests for `core/`

## Add a Plan-Driven Change

Primary files:

- `docs/exec-plans/active/`
- `docs/exec-plans/completed/`
- code and docs referenced by that plan

What to verify:

- design explains scope and non-goals
- implementation plan names exact files and verification commands
- completed work updates the docs named in the plan
