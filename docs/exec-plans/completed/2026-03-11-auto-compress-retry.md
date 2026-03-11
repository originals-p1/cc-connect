# Auto Compress Retry Execution Plan

## Objective

Rollback commit `69a0e20087043cfc5daacb951a98c7b8522a5a9c` and replace the larger
scanner-buffer approach with proactive auto-compression plus a single retry of
the original request.

## Affected Modules

- `core/`
- `agent/claudecode/`
- `agent/codex/`
- `agent/cursor/`
- `agent/gemini/`
- `agent/opencode/`
- `agent/qoder/`
- `docs/`

## Approach

- Remove `core/scanner.go` and its test.
- Add a shared incremental JSONL line reader with soft-limit detection.
- Introduce an internal event/error signal for "auto-compress needed".
- Update affected agent session read loops to use the new reader.
- Extend engine turn processing to run compression and retry the original
  message once.
- Keep `/compress` behavior as the shared compression mechanism.

## Validation Strategy

- Write failing tests first for the line reader and engine retry behavior.
- Run focused tests while implementing:
  - `go test ./core/...`
- If targeted coverage exposes wider risk, escalate to:
  - `go test ./...`

## Rollback Plan

- Revert the new line reader and engine retry logic.
- Restore the previous scanner-based read loops if needed.
- Reapply the old large-buffer helper only as a temporary fallback.
