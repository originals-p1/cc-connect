# OpenCode Late Text Empty Response Bugfix Execution Plan

## Objective

Fix the OpenCode adapter so `cc-connect` does not emit `(empty response)` when
OpenCode delivers trailing text after `step_finish` but before the CLI process
actually exits.

## Affected Modules

- `agent/opencode/session.go`
- `agent/opencode/session_test.go`

## Approach

Treat `step_finish` as an internal progress marker only. Emit the final
`EventResult` after the OpenCode process exits cleanly, so any late-arriving
text events are consumed before the engine finalizes the turn.

## Validation Strategy

- run `go test ./agent/opencode/...`
- run `go test ./core/...`

## Rollback Plan

- restore `step_finish`-driven `EventResult` emission in `agent/opencode/session.go`
- remove the late-text regression coverage from `agent/opencode/session_test.go`
