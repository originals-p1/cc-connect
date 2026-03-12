# OpenCode Empty Session List Bugfix Execution Plan

## Objective

Fix the OpenCode session listing path so `/list` still works after `/clear` removes every backend session.

## Affected Modules

- `agent/opencode/opencode.go`
- `agent/opencode/opencode_test.go`

## Approach

Treat empty or whitespace-only output from `opencode session list --format json` as an empty session list instead of attempting to unmarshal it as JSON, because some OpenCode CLI states appear to emit no payload when no sessions remain.

## Validation Strategy

- run `go test ./agent/opencode/...`
- run `go test ./core/...`

## Rollback Plan

- restore the previous strict JSON-only parsing in `agent/opencode/opencode.go`
- remove the regression coverage for empty session list output in `agent/opencode/opencode_test.go`

## Steps

1. Update OpenCode session list parsing to handle empty output safely.
2. Add regression tests for empty and whitespace-only session list responses.
3. Run targeted validation for the OpenCode agent and core session listing flow.
