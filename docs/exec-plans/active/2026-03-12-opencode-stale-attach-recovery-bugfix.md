# OpenCode Stale Attach Recovery Bugfix Execution Plan

## Objective

Fix the OpenCode session runtime so a stored agent session ID does not keep pointing new resident servers at stale or deleted backend sessions after restart.

## Affected Modules

- `agent/opencode/session.go`
- `agent/opencode/session_test.go`

## Approach

Detect OpenCode attach/session failures that indicate the resident server cannot resolve the persisted session anymore, clear the cached session ID for that in-memory session, and retry once against the same resident server as a fresh turn so the bridge can rebind automatically.

## Validation Strategy

- run `go test ./agent/opencode/...`
- run `go test ./core/...`
- run `make build`

## Rollback Plan

- remove the stale-session retry path from `agent/opencode/session.go`
- remove the retry regression coverage from `agent/opencode/session_test.go`

## Steps

1. Add stale-session detection around attached OpenCode runs.
2. Retry once without `--session` when the backend reports a missing session.
3. Add regression tests covering the retry and session ID reset behavior.
4. Run targeted validation and build checks.

## Result

- OpenCode attached runs now detect missing-session failures from stale resident-server state and retry once as a fresh turn.
- The in-memory OpenCode session binding is cleared before the retry so the next `step_start` can rebind `cc-connect` to the new backend session ID.
- Regression coverage verifies the first run resumes with the stale session ID, the retry drops `--session`, and the session ID is updated from the fresh run.
- Validation completed with `go test ./agent/opencode/...`, `go test ./core/...`, and `make build`.
