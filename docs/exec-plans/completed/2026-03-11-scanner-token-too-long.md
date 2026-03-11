# Scanner Token Too Long Execution Plan

## Objective

Fix agent stdout scanning so large single-line event payloads do not fail with `bufio.Scanner: token too long`.

## Affected Modules

- `core/`
- `agent/opencode/`
- `agent/gemini/`
- `agent/cursor/`
- `agent/qoder/`

## Approach

Add one shared scanner helper with an increased buffer limit, cover it with a focused test, then switch affected agent session readers to that helper so behavior stays consistent across integrations.

## Validation Strategy

- run focused tests for the new scanner helper
- run targeted agent/core tests if needed
- run `go test ./...`

## Rollback Plan

- remove the shared helper and its test
- revert affected agent sessions back to their previous scanner construction
