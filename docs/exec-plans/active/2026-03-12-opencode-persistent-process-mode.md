# OpenCode Persistent Process Mode Execution Plan

## Objective

Switch the OpenCode agent integration from launching a fresh cold OpenCode backend on every turn to using a session-scoped resident backend process.

## Affected Modules

- `agent/opencode/session.go`
- `agent/opencode/session_test.go`
- `agent/opencode/opencode.go`

## Approach

Start `opencode serve` once when an OpenCode session is created, wait for its local health endpoint to become ready, and route each `Send()` through `opencode run --attach <server>` so the same backend stays warm across turns while preserving the existing NDJSON event parsing flow.

## Validation Strategy

- run `go test ./agent/opencode/...`
- run `go test ./core/...`
- run `make build`

## Rollback Plan

- restore the previous per-turn direct `opencode run --format json` launch path in `agent/opencode/session.go`
- remove the persistent-backend regression coverage from `agent/opencode/session_test.go`

## Steps

1. Add resident OpenCode server lifecycle management to the session implementation.
2. Update send execution to attach to the resident server instead of cold-starting OpenCode directly.
3. Extend tests to cover the resident backend startup and attached run invocation.
4. Run targeted tests and build validation.

## Result

- OpenCode sessions now start a resident `opencode serve` backend per session and wait for `/global/health` before accepting sends.
- Per-turn sends now use `opencode run --attach <server> --format json`, preserving the existing event parsing path while avoiding backend cold starts.
- Regression coverage now verifies attached run arguments and resident backend startup using a fake OpenCode CLI.
- Validation completed with `go test ./agent/opencode/...`, `go test ./core/...`, and `make build`.
