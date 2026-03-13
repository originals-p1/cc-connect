# Codex Session Hardening Execution Plan

## Objective

Harden the Codex integration so session failures surface reliably and session
file operations target the correct transcript, while adding regression coverage
for the currently under-tested `agent/codex/session.go` flow.

## Affected Modules

- `agent/codex/session.go`
- `agent/codex/list.go`
- `agent/codex/` tests

Read-only context:

- `core/interfaces.go`
- `core/engine.go`
- `docs/VALIDATION.md`

## Background

Current review of the Codex adapter found two high-confidence issues:

1. `codexSession.readLoop()` only emits `EventError` on process failure when
   `stderr` is non-empty. A non-zero exit with empty `stderr` can therefore
   fail silently.
2. `findSessionFile()` matches transcript files using substring search on the
   basename, which can resolve the wrong session when IDs overlap.

The adapter also lacks dedicated regression coverage for the main session event
flow, so these issues are easy to miss.

## Approach

### 1. Make Codex process failures non-silent

Update `agent/codex/session.go` so a non-zero Codex exit always emits an
`EventError`, even when `stderr` is empty.

Requirements:

- Prefer `stderr` text when available.
- Fall back to `cmd.Wait()` error text when `stderr` is empty.
- Do not emit duplicate error events for the same failed turn.

### 2. Make transcript lookup exact

Tighten `findSessionFile()` in `agent/codex/list.go` so it resolves the target
session transcript by exact session identity rather than loose basename
substring matching.

Acceptable strategies:

- parse candidate files and compare `session_meta.payload.id`, or
- match a filename pattern that is provably exact for Codex session IDs.

Prefer correctness over speed; the current path is already filesystem-bound.

### 3. Add missing Codex session regression tests

Add targeted tests for:

- non-zero exit with empty `stderr`
- non-zero exit with non-empty `stderr`
- transcript lookup when one session ID is a substring of another
- basic session event flow where `thread.started` / `turn.completed` produce a
  final `EventResult`

## Non-Goals

This plan does not include:

- adding image support to Codex
- adding `CommandProvider` support
- changing Codex permission mode semantics
- redesigning model discovery or provider mapping

## Validation Strategy

Minimum validation:

```bash
go test ./agent/codex/...
```

If shared behavior changes beyond the adapter-local scope, escalate to:

```bash
go test ./core/...
go test ./...
make build
```

Manual spot checks after implementation:

- a failed Codex subprocess with empty `stderr` still surfaces a visible error
- deleting or reading history for session `abc` does not hit `xyzabc`

## Rollback Plan

If the hardening change regresses Codex behavior:

1. restore the previous `cmd.Wait()` failure handling in `agent/codex/session.go`
2. restore the previous transcript lookup logic in `agent/codex/list.go`
3. remove the new Codex regression tests that depend on the hardened behavior

## Implementation Steps

### Phase 1: Failure surfacing (已完成)

- [x] fix silent failure handling in `agent/codex/session.go`
- [x] add failure-path regression tests

### Phase 2: Transcript targeting (已完成)

- [x] replace substring-based transcript lookup
- [x] add overlap-session regression tests

### Phase 3: Session coverage baseline (已完成)

- [x] add a basic Codex session event-flow test
- [x] rerun targeted Codex validation

## Current Progress

Implemented in the current workspace:

- `cmd.Wait()` failure handling now falls back to the process error when `stderr` is empty
- duplicate `EventError` emission is suppressed after `turn.failed`
- `findSessionFile()` now keeps scanning until it finds an exact `session_meta.id` match
- regression tests now cover empty-stderr failure, duplicate-error suppression, exact transcript lookup, and a basic completed-turn event flow

Validation run:

- `go test ./agent/codex/...`
