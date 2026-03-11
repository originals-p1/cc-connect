# Validation Guide

This document defines how to verify changes in this repository.

For repository workflow context, also see [`docs/harness/quality-gates.md`](harness/quality-gates.md).

## Standard Validation

Preferred full validation:

```bash
make check-harness
go test ./...
make build
```

Use the full set when the change touches shared flow, runtime wiring, config shape, or user-visible behavior.

## Narrow Changes

If a change is localized, start with targeted checks first, then escalate to the full set if the change can affect shared behavior.

Typical examples:

- `go test ./core/...` for `core/` changes
- `go test ./config/...` for config changes
- `go test ./cmd/cc-connect/...` for CLI changes

## Documentation and Sync Checks

When behavior changes, review whether these need updates:

- `config.example.toml`
- `README.md`
- platform- or feature-specific docs in `docs/`
- `docs/AI_CONTEXT.md` when structural knowledge changes materially
- `core/i18n.go` for user-visible command/help/message changes

## Harness Check

Repository governance check:

```bash
make check-harness
```

Current first-pass coverage:

- each package under `agent/` is blank-imported in `cmd/cc-connect/main.go`
- each package under `platform/` is blank-imported in `cmd/cc-connect/main.go`

## Completion Rule

Do not claim a task is complete without fresh verification evidence from the relevant commands.
