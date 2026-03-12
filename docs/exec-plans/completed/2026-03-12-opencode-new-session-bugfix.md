# OpenCode New Session Bugfix Execution Plan

## Objective

Fix the OpenCode integration so ordinary sends keep the current conversation
only when a real OpenCode session ID exists, while `/new` reliably starts a new
backend conversation instead of reusing resident attached state.

## Affected Modules

- `agent/opencode/session.go`
- `agent/opencode/session_test.go`

## Approach

Remove the resident `opencode serve` + `run --attach` flow and return to direct
`opencode run --format json` execution per turn, using `--session` only when
`cc-connect` already has a concrete OpenCode session ID. This preserves
multi-turn continuity without keeping backend-attached state that prevents fresh
conversations from being created.

## Validation Strategy

- run `go test ./agent/opencode/...`
- run `go test ./core/...`
- run `make build`

## Rollback Plan

- restore the resident OpenCode server lifecycle in `agent/opencode/session.go`
- restore attach-mode regression tests in `agent/opencode/session_test.go`
- remove the direct-run session creation regression coverage

## Steps

1. Replace the attach-mode send path with direct `opencode run --format json`.
2. Keep stale-session retry support by retrying once without `--session`.
3. Add regression tests that verify new sessions do not use `--attach` and that
   resumed turns still pass `--session`.
4. Run targeted tests and a build check.
