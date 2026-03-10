# Clear Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `/clear` slash command that deletes all sessions for the current project and resets the caller's local active session state.

**Architecture:** Extend `core.Engine` with a bulk-delete command that reuses the existing session deletion capability exposed by agent backends. Add session-manager helpers to drop local sessions and names for deleted agent session IDs, then wire new user-facing strings into `core/i18n.go`.

**Tech Stack:** Go, `core.Engine`, `core.SessionManager`, existing agent `ListSessions` / `DeleteSession` interfaces, Go tests

---

### Task 1: Add the failing engine test

**Files:**
- Modify: `core/engine_test.go`
- Reference: `core/engine.go`

**Step 1: Write the failing test**

Add a test that:
- creates a stub agent with three listable sessions
- marks one of them as the current local active session
- invokes `/clear`
- expects all three backend sessions to be deleted
- expects the local active session to be reset so it no longer points at the deleted session

**Step 2: Run test to verify it fails**

Run: `go test ./core -run TestCmdClear -v`
Expected: FAIL because `/clear` is not implemented.

**Step 3: Write minimal implementation**

Do not implement in this task.

**Step 4: Run test to verify it passes**

Do not run yet.

**Step 5: Commit**

```bash
git add core/engine_test.go
git commit -m "test: add clear command coverage"
```

### Task 2: Add local session-manager support for clearing deleted bindings

**Files:**
- Modify: `core/session.go`
- Modify: `core/session_test.go`

**Step 1: Write the failing test**

Add a focused `SessionManager` test that seeds:
- multiple local sessions for one user
- custom names for deleted agent session IDs
- an active session pointing at one deleted agent session

Assert that a new helper removes the affected local sessions, clears matching session names, and clears the active-session pointer if needed.

**Step 2: Run test to verify it fails**

Run: `go test ./core -run TestSessionManagerClearAgentSessionsForUser -v`
Expected: FAIL because the helper does not exist.

**Step 3: Write minimal implementation**

Add a `SessionManager` helper that accepts `userKey` and deleted agent session IDs, then:
- removes matching local sessions from `sessions`
- removes them from `userSessions[userKey]`
- clears `activeSession[userKey]` if it referenced a removed local session
- deletes matching entries from `sessionNames`
- persists via `saveLocked()`

**Step 4: Run test to verify it passes**

Run: `go test ./core -run TestSessionManagerClearAgentSessionsForUser -v`
Expected: PASS

**Step 5: Commit**

```bash
git add core/session.go core/session_test.go
git commit -m "feat: add session manager clear helper"
```

### Task 3: Implement `/clear` command in the engine

**Files:**
- Modify: `core/engine.go`
- Modify: `core/i18n.go`

**Step 1: Write the failing test**

Reuse `TestCmdClear` from Task 1 and add one more case for agents without `SessionDeleter`.

**Step 2: Run test to verify it fails**

Run: `go test ./core -run 'TestCmdClear|TestCmdClearUnsupportedAgent' -v`
Expected: FAIL because command routing/messages are incomplete.

**Step 3: Write minimal implementation**

Implement `/clear` by:
- checking `SessionDeleter` support
- listing agent sessions
- deleting each listed session
- collecting deleted agent session IDs
- resetting local state through the new `SessionManager` helper
- replying with a summary message

Add user-facing strings for:
- usage/help entry
- unsupported agent
- success summary

Wire `/clear` into command matching and help output.

**Step 4: Run test to verify it passes**

Run: `go test ./core -run 'TestCmdClear|TestCmdClearUnsupportedAgent' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add core/engine.go core/i18n.go core/engine_test.go
git commit -m "feat: add clear sessions command"
```

### Task 4: Run broader verification

**Files:**
- Modify: none

**Step 1: Write the failing test**

No new test.

**Step 2: Run targeted verification**

Run: `go test ./core -v`
Expected: PASS

**Step 3: Run full verification**

Run: `go test ./...`
Expected: PASS

**Step 4: Optional build verification**

Run: `make build`
Expected: PASS

**Step 5: Commit**

```bash
git add docs/plans/2026-03-10-clear-command-design.md docs/plans/2026-03-10-clear-command.md
git commit -m "docs: add clear command design and plan"
```
