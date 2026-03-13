package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

func writeFakeCodex(t *testing.T, script string) string {
	t.Helper()
	binDir := t.TempDir()
	cmdPath := filepath.Join(binDir, "codex")
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}

	pathEnv := binDir
	if current := os.Getenv("PATH"); current != "" {
		pathEnv += string(os.PathListSeparator) + current
	}
	t.Setenv("PATH", pathEnv)
	if runtime.GOOS == "windows" {
		t.Fatalf("fake codex helper is shell-script based and not supported on windows")
	}
	return cmdPath
}

func collectEventsUntil(t *testing.T, ch <-chan core.Event, wait time.Duration) []core.Event {
	t.Helper()
	var events []core.Event
	timer := time.NewTimer(wait)
	defer timer.Stop()

	for {
		select {
		case evt := <-ch:
			events = append(events, evt)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(wait)
		case <-timer.C:
			return events
		}
	}
}

func TestCodexSessionEmitsErrorWhenProcessFailsWithoutStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake codex not supported on windows")
	}
	writeFakeCodex(t, "#!/bin/sh\nexit 7\n")

	cs, err := newCodexSession(context.Background(), t.TempDir(), "", "suggest", "", nil)
	if err != nil {
		t.Fatalf("newCodexSession() error = %v", err)
	}
	defer cs.Close()

	if err := cs.Send("hello", nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	events := collectEventsUntil(t, cs.Events(), 200*time.Millisecond)
	if len(events) != 1 {
		t.Fatalf("events = %+v, want exactly 1 error event", events)
	}
	if events[0].Type != core.EventError {
		t.Fatalf("event type = %q, want %q", events[0].Type, core.EventError)
	}
	if events[0].Error == nil || events[0].Error.Error() == "" {
		t.Fatalf("event error = %v, want non-empty error", events[0].Error)
	}
}

func TestCodexSessionDoesNotDuplicateTurnFailedError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake codex not supported on windows")
	}
	writeFakeCodex(t, `#!/bin/sh
cat <<'EOF'
{"type":"turn.started"}
{"type":"turn.failed","error":{"message":"tool crashed"}}
EOF
exit 1
`)

	cs, err := newCodexSession(context.Background(), t.TempDir(), "", "suggest", "", nil)
	if err != nil {
		t.Fatalf("newCodexSession() error = %v", err)
	}
	defer cs.Close()

	if err := cs.Send("hello", nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	events := collectEventsUntil(t, cs.Events(), 200*time.Millisecond)
	if len(events) != 1 {
		t.Fatalf("events = %+v, want exactly 1 error event", events)
	}
	if events[0].Type != core.EventError {
		t.Fatalf("event type = %q, want %q", events[0].Type, core.EventError)
	}
	if got, want := events[0].Error.Error(), "tool crashed"; got != want {
		t.Fatalf("event error = %q, want %q", got, want)
	}
}

func TestCodexSessionEmitsResultForBasicCompletedTurn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake codex not supported on windows")
	}
writeFakeCodex(t, `#!/bin/sh
cat <<'EOF'
{"type":"thread.started","thread_id":"thread_123"}
{"type":"turn.started"}
{"type":"item.completed","item":{"type":"agent_message","text":"final answer"}}
{"type":"turn.completed"}
EOF
`)

	cs, err := newCodexSession(context.Background(), t.TempDir(), "", "suggest", "", nil)
	if err != nil {
		t.Fatalf("newCodexSession() error = %v", err)
	}
	defer cs.Close()

	if err := cs.Send("hello", nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	events := collectEventsUntil(t, cs.Events(), 200*time.Millisecond)
	if len(events) != 2 {
		t.Fatalf("events = %+v, want text + result", events)
	}
	if events[0].Type != core.EventText || events[0].Content != "final answer" {
		t.Fatalf("first event = %+v, want final text event", events[0])
	}
	if events[1].Type != core.EventResult {
		t.Fatalf("second event = %+v, want result event", events[1])
	}
	if got, want := cs.CurrentSessionID(), "thread_123"; got != want {
		t.Fatalf("CurrentSessionID() = %q, want %q", got, want)
	}
}
