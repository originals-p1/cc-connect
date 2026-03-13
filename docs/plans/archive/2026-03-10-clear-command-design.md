# Clear Command Design

**Date:** 2026-03-10

**Goal:** Add a `/clear` command that deletes all sessions for the current project without confirmation.

## Scope

- The command applies to the current project only.
- It deletes all agent-backed sessions visible to the current engine.
- It also resets the caller's local active session state so the next message starts fresh.
- It does not affect project routing or sessions in other projects.

## User Experience

- User runs `/clear`.
- If the current agent does not support deletion, return a clear error.
- If session listing fails, return the backend error.
- On success, return a summary with the number of deleted sessions.
- After success, the caller no longer has an active local session bound to a deleted agent session.

## Behavior

### Agent layer

- Reuse existing `ListSessions` and `DeleteSession` support.
- Delete every session returned by `ListSessions`.
- Fail fast on the first delete error and report it.

### Local session state

- Remove local session-manager entries for the caller that point at deleted agent sessions.
- Clear any custom display names tied to deleted agent sessions.
- Ensure the caller no longer points at a stale active session after `/clear`.

### Command surface

- Add `/clear` to built-in command routing and help text.
- Keep semantics parallel to `/delete`, but bulk-oriented and non-interactive.

## Testing

- Add a failing engine test for `/clear` deleting all agent sessions and resetting local state.
- Verify unsupported-agent behavior.
- Verify the command is exposed in help/command routing as needed.

