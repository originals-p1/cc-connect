# OpenCode Integration Execution Plan

## Objective

Improve the OpenCode agent integration so it better matches the `cc-connect` agent contract for runtime behavior, provider wiring, session management, and user-visible feedback.

## Affected Modules

- `agent/opencode/opencode.go`
- `agent/opencode/session.go`
- `agent/opencode/` tests
- `core/engine.go`
- `core/` tests
- `config.example.toml`

## Approach

Update the OpenCode adapter to apply mode-specific CLI flags, support attached files for image inputs, wire provider environment variables more completely, expose OpenCode command directories, and support session deletion. Harden event parsing and process error handling so failed runs surface reliable errors. Align the shared engine flow with OpenCode by rendering tool-result events instead of silently dropping them.

## Validation Strategy

- run targeted tests for `agent/opencode/...` and `core/...`
- run `go test ./...`
- run `make build`
- review `config.example.toml` comments against the implemented OpenCode options

## Rollback Plan

- revert the OpenCode agent changes in `agent/opencode/`
- revert the shared engine event-handling changes in `core/engine.go`
- remove any OpenCode-specific config example updates that no longer match behavior

## Steps

1. Add an execution plan for the OpenCode integration work.
2. Update the OpenCode adapter to use the correct runtime flags and input handling.
3. Improve optional agent capabilities such as provider env wiring, command discovery, and session deletion.
4. Add tests for OpenCode session behavior and any shared engine changes.
5. Run validation and summarize the resulting behavior changes.
