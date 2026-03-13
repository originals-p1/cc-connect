# Auto Compress Retry Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace oversized agent stdout line buffering with proactive auto-compression and a single retry of the original request.

**Architecture:** Add a bounded incremental line reader in `core/` that can detect when a JSONL record approaches the configured maximum without buffering the whole line. Session readers emit a structured auto-compress signal, and `core.Engine` handles that signal by invoking the existing compression flow and retrying the turn once.

**Tech Stack:** Go, existing `core.Engine` turn loop, agent session JSONL readers, Go test

---

### Task 1: Add reader coverage first

**Files:**
- Create: `core/line_reader_test.go`
- Delete: `core/scanner_test.go`
- Modify: `core/`

**Step 1: Write the failing test**

Add tests that prove:
- a normal line is returned intact
- a near-limit line returns the soft-limit sentinel and discards the remainder

**Step 2: Run test to verify it fails**

Run: `go test ./core/...`
Expected: FAIL because the new reader and sentinel do not exist yet.

**Step 3: Write minimal implementation**

Create the incremental reader and only the minimum exported API needed by tests.

**Step 4: Run test to verify it passes**

Run: `go test ./core/...`
Expected: PASS for the new reader tests.

### Task 2: Drive engine retry behavior with tests

**Files:**
- Modify: `core/engine_test.go`
- Modify: `core/message.go`

**Step 1: Write the failing test**

Add focused engine tests that prove:
- auto-compress signal triggers compression
- the original prompt is retried once
- repeated auto-compress on retry stops with an error instead of looping

**Step 2: Run test to verify it fails**

Run: `go test ./core/...`
Expected: FAIL because engine does not handle the auto-compress signal yet.

**Step 3: Write minimal implementation**

Add the event/sentinel plumbing and the single-retry control flow.

**Step 4: Run test to verify it passes**

Run: `go test ./core/...`
Expected: PASS for the new engine tests.

### Task 3: Wire agent session readers to the new bounded reader

**Files:**
- Modify: `agent/claudecode/session.go`
- Modify: `agent/codex/session.go`
- Modify: `agent/cursor/session.go`
- Modify: `agent/gemini/session.go`
- Modify: `agent/opencode/session.go`
- Modify: `agent/qoder/session.go`
- Delete: `core/scanner.go`

**Step 1: Write the failing test**

Use existing/new core tests so wiring is exercised through shared reader behavior.

**Step 2: Run test to verify it fails**

Run: `go test ./core/...`
Expected: FAIL until session readers emit the new signal consistently.

**Step 3: Write minimal implementation**

Replace `NewLineScanner` usage with the bounded line reader and map the sentinel
to the engine-visible auto-compress event.

**Step 4: Run test to verify it passes**

Run: `go test ./core/...`
Expected: PASS.

### Task 4: Validate and finalize

**Files:**
- Modify: `docs/exec-plans/active/2026-03-11-auto-compress-retry.md`
- Move: `docs/exec-plans/completed/2026-03-11-auto-compress-retry.md`

**Step 1: Run targeted validation**

Run: `go test ./core/...`
Expected: PASS.

**Step 2: Run broader validation if needed**

Run: `go test ./...`
Expected: PASS or clear report of unrelated failures.

**Step 3: Finalize plan**

Move the execution plan to completed after verification succeeds.
