# Auto Compress Retry Design

## Objective

Replace the March 11, 2026 scanner-buffer increase with a proactive safeguard:
when an agent emits a single JSONL event line that approaches the line buffer
limit, stop parsing that line, automatically trigger session context compression,
and retry the original user request once.

## Problem

Commit `69a0e20087043cfc5daacb951a98c7b8522a5a9c` raised the scanner limit for
agent stdout to tolerate oversized single-line JSON payloads. That treats the
symptom rather than the cause. The desired behavior is to preserve a bounded
line reader and recover by compacting context before the session degrades.

## Constraints

- Keep agent-specific compression commands behind `core.ContextCompressor`.
- Avoid duplicating compression logic across agent packages.
- Prevent infinite retry loops if compression does not help.
- Keep changes localized to shared runtime flow and affected JSONL agent readers.

## Chosen Approach

1. Remove the shared large-buffer scanner helper introduced by the bad commit.
2. Add a shared line reader in `core/` that reads one line incrementally with:
   - a hard maximum line size
   - a soft threshold below the hard maximum
   - discard-until-newline behavior when the soft threshold is hit
3. Have JSONL agent session readers use that helper instead of `bufio.Scanner`.
4. When the helper reports a soft-limit condition, emit a structured event that
   signals the engine to auto-compress instead of surfacing a normal agent error.
5. In `core.Engine`, detect that event during a turn, invoke the existing
   compression path, then retry the original prompt once with the same images.
6. If the retry also hits the same condition, surface a user-visible error and
   stop without looping.

## Runtime Behavior

1. User sends a prompt.
2. Engine sends the prompt to the active agent session.
3. Agent emits JSONL output.
4. The shared line reader detects that one line is approaching the configured
   maximum before the full line is buffered.
5. The reader discards the rest of that oversized line and returns a sentinel.
6. The session emits an auto-compress request event.
7. Engine runs the agent's native compress command on the same live session.
8. After compression succeeds, engine resends the original user prompt once.
9. The retried turn proceeds normally, or fails definitively if the same limit
   is reached again.

## Error Handling

- If the agent does not implement `ContextCompressor`, return a normal error.
- If compression fails, return the compression error and do not retry.
- If the retry hits the same soft-limit again, return an error that automatic
  compression did not recover the turn.
- If the agent session dies during compression, clean up interactive state as
  the existing `/compress` flow already does.

## Validation

- Unit test the line reader soft-threshold behavior.
- Add engine tests for:
  - auto-compress event triggers compression and a single retry
  - no retry loop on repeated soft-limit events
  - compression unsupported/failure paths
- Run targeted `go test ./core/...`.

