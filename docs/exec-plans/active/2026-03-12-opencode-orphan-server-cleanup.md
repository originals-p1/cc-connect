# OpenCode Orphan Server Cleanup Execution Plan

## Objective

Reduce stray `opencode serve` processes by ensuring relay-mode OpenCode sessions are closed once a synchronous relay turn finishes.

## Affected Modules

- `core/engine.go`
- `core/engine_test.go`

## Approach

Treat relay sessions as short-lived agent sessions: start them for the relay turn, defer `Close()` immediately after successful startup, and keep the existing interactive-session lifecycle unchanged so only one-off relay sessions get eagerly cleaned up.

## Validation Strategy

- run `go test ./core/...`
- run `go test ./agent/opencode/...`
- run `make build`

## Rollback Plan

- remove the deferred relay session close from `core/engine.go`
- remove the relay close regression test from `core/engine_test.go`

## Steps

1. Add explicit relay-session cleanup after `StartSession()` succeeds.
2. Add regression coverage proving relay closes the agent session even on success.
3. Run targeted tests and build validation.

## Result

- Relay-mode agent sessions are now always closed after `HandleRelay()` completes, which prevents one-off OpenCode relay turns from leaving resident `opencode serve` processes behind.
- Interactive chat sessions keep their existing lifecycle, so only short-lived relay sessions get eager cleanup.
- Regression coverage verifies relay success still returns the response, persists the agent session ID, and closes the agent session exactly once.
- Validation completed with `go test ./core/...`, `go test ./agent/opencode/...`, and `make build`.
