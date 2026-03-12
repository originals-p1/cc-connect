# Telegram OpenCode Event Filtering Execution Plan

## Objective

Prevent Telegram users from seeing raw internal OpenCode structured payloads such as task/todo JSON while preserving normal final assistant replies.

## Affected Modules

- `agent/opencode/session.go`
- `agent/opencode/session_test.go`

## Approach

Tighten OpenCode event parsing so only user-facing text becomes `EventText`. Ignore structured JSON payloads that look like internal tool/task outputs, while still emitting explicit tool-use and tool-result events through their dedicated handlers.

## Validation Strategy

- run `go test ./agent/opencode/...`
- run `go test ./core/...`

## Rollback Plan

- restore the previous `handleText` behavior in `agent/opencode/session.go`
- remove the added OpenCode session regression tests

## Steps

1. Add a guard that detects structured internal payloads in OpenCode text events.
2. Skip emitting those payloads as assistant text while keeping normal text untouched.
3. Add regression coverage for leaked todo-style JSON and ordinary text events.
4. Run targeted Go tests.

## Result

- Added structured-payload suppression for OpenCode text events that look like internal task state JSON.
- Added regression tests for leaked todo payloads and normal assistant text.
- Validation completed with `go test ./agent/opencode/...` and `go test ./core/...`.
