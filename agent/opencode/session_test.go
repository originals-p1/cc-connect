package opencode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

func writeFakeOpencode(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	cmdPath := filepath.Join(binDir, "opencode")
	script := `#!/bin/sh
set -eu

cmd="${1:-}"
shift || true

case "$cmd" in
  run)
    if [ -n "${CC_ARGS_LOG:-}" ]; then
      printf '%s\n' "$cmd" "$@" >> "$CC_ARGS_LOG"
    fi
    if [ -n "${CC_RUN_OUTPUT:-}" ]; then
      printf '%s\n' "$CC_RUN_OUTPUT"
    else
      cat <<'EOF'
{"type":"step_start","part":{"sessionID":"ses_test"}}
{"type":"step_finish","part":{}}
EOF
    fi
    ;;
  *)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}
	return cmdPath
}

func collectUntilResult(t *testing.T, events <-chan core.Event) []core.Event {
	t.Helper()
	var seen []core.Event
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatalf("events closed before result")
			}
			seen = append(seen, event)
			if event.Type == core.EventResult {
				return seen
			}
		case <-deadline:
			t.Fatal("timed out waiting for result event")
		}
	}
}

func TestOpencodeSessionSendUsesDirectRunModeAndImages(t *testing.T) {
	workDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	cmdPath := writeFakeOpencode(t)
	t.Setenv("CC_ARGS_LOG", logPath)

	s, err := newOpencodeSession(context.Background(), cmdPath, workDir, "openai/gpt-4.1", "yolo", "", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	img := core.ImageAttachment{MimeType: "image/png", Data: []byte("png-bytes"), FileName: "diagram.png"}
	if err := s.Send("review this", []core.ImageAttachment{img}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	collectUntilResult(t, s.Events())

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read args log: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(b)), "\n")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--agent coder") {
		t.Fatalf("args = %q, want yolo mode to add --agent coder", joined)
	}
	if strings.Contains(joined, "--attach") {
		t.Fatalf("args = %q, should not use attached resident server", joined)
	}
	if !strings.Contains(joined, "review this") {
		t.Fatalf("args = %q, want original prompt", joined)
	}
	if !strings.Contains(joined, ".png") {
		t.Fatalf("args = %q, want image path reference", joined)
	}
	if strings.Contains(joined, "png-bytes") {
		t.Fatalf("args = %q, should not inline raw image bytes", joined)
	}
}

func TestOpencodeSessionHandleErrorEvent(t *testing.T) {
	s, err := newOpencodeSession(context.Background(), writeFakeOpencode(t), ".", "", "default", "", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	s.handleEvent(map[string]any{
		"type": "error",
		"error": map[string]any{
			"data": map[string]any{"message": "model not found"},
		},
	})

	event := <-s.Events()
	if event.Type != core.EventError {
		t.Fatalf("event type = %q, want %q", event.Type, core.EventError)
	}
	if event.Error == nil || !strings.Contains(event.Error.Error(), "model not found") {
		t.Fatalf("event error = %v, want model not found", event.Error)
	}
}

func TestOpencodeSessionToolResultUsesToolResultField(t *testing.T) {
	s, err := newOpencodeSession(context.Background(), writeFakeOpencode(t), ".", "", "default", "", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	s.handleToolUse(map[string]any{
		"part": map[string]any{
			"tool": "Bash",
			"state": map[string]any{
				"status": "completed",
				"output": "done",
			},
		},
	})

	event := <-s.Events()
	if event.Type != core.EventToolResult {
		t.Fatalf("event type = %q, want %q", event.Type, core.EventToolResult)
	}
	if event.ToolResult != "done" {
		t.Fatalf("tool result = %q, want %q", event.ToolResult, "done")
	}
	if event.Content != "" {
		t.Fatalf("event content = %q, want empty string", event.Content)
	}
}

func TestWriteImageRefsCleansUp(t *testing.T) {
	refs, cleanup, err := writeImageRefs([]core.ImageAttachment{{MimeType: "image/png", Data: []byte("abc")}})
	if err != nil {
		t.Fatalf("writeImageRefs() error = %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("refs = %v, want 1 ref", refs)
	}
	if _, err := os.Stat(refs[0]); err != nil {
		t.Fatalf("temp image missing before cleanup: %v", err)
	}
	cleanup()
	if _, err := os.Stat(refs[0]); !os.IsNotExist(err) {
		t.Fatalf("temp image still exists after cleanup: %v", err)
	}
}

func TestFileExtForImagePrefersFilename(t *testing.T) {
	if got := fileExtForImage(core.ImageAttachment{MimeType: "image/png", FileName: "photo.jpeg"}); got != ".jpeg" {
		t.Fatalf("fileExtForImage() = %q, want %q", got, ".jpeg")
	}
	if runtime.GOOS == "windows" {
		return
	}
	if got := fileExtForImage(core.ImageAttachment{MimeType: "image/webp"}); got == "" {
		t.Fatal("expected extension for image/webp")
	}
}

func TestHandleToolUseInputMapSummary(t *testing.T) {
	s, err := newOpencodeSession(context.Background(), writeFakeOpencode(t), ".", "", "default", "", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	input := map[string]any{"command": "ls", "path": "."}
	s.handleToolUse(map[string]any{
		"part": map[string]any{
			"tool": "Bash",
			"state": map[string]any{
				"input": input,
			},
		},
	})

	event := <-s.Events()
	if event.Type != core.EventToolUse {
		t.Fatalf("event type = %q, want %q", event.Type, core.EventToolUse)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(event.ToolInput), &decoded); err != nil {
		t.Fatalf("tool input %q is not valid JSON: %v", event.ToolInput, err)
	}
	if decoded["command"] != "ls" {
		t.Fatalf("decoded tool input = %v, want command ls", decoded)
	}
}

func TestOpencodeSessionSuppressesTodoJSONText(t *testing.T) {
	s, err := newOpencodeSession(context.Background(), writeFakeOpencode(t), ".", "", "default", "", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	s.handleText(map[string]any{
		"part": map[string]any{
			"text": `[
				{"content":"Inspect current OpenCode agent integration","priority":"high","status":"in_progress"},
				{"content":"Implement filtering","priority":"high","status":"pending"}
			]`,
		},
	})

	select {
	case event := <-s.Events():
		t.Fatalf("unexpected event: %+v", event)
	default:
	}
}

func TestOpencodeSessionKeepsRegularText(t *testing.T) {
	s, err := newOpencodeSession(context.Background(), writeFakeOpencode(t), ".", "", "default", "", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	s.handleText(map[string]any{
		"part": map[string]any{
			"text": "Fixed the Telegram bridge to only send final user-facing replies.",
		},
	})

	event := <-s.Events()
	if event.Type != core.EventText {
		t.Fatalf("event type = %q, want %q", event.Type, core.EventText)
	}
	if event.Content != "Fixed the Telegram bridge to only send final user-facing replies." {
		t.Fatalf("event content = %q", event.Content)
	}
}

func TestOpencodeSessionResumesWithStoredSessionID(t *testing.T) {
	workDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	cmdPath := writeFakeOpencode(t)
	t.Setenv("CC_ARGS_LOG", logPath)

	s, err := newOpencodeSession(context.Background(), cmdPath, workDir, "", "default", "ses_existing", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	if err := s.Send("continue", nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	collectUntilResult(t, s.Events())

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read args log: %v", err)
	}
	joined := strings.Join(strings.Split(strings.TrimSpace(string(b)), "\n"), " ")
	if !strings.Contains(joined, "--session ses_existing") {
		t.Fatalf("args = %q, want stored session resume flag", joined)
	}
}

func TestOpencodeSessionDelaysResultUntilLateTextArrives(t *testing.T) {
	workDir := t.TempDir()
	cmdPath := writeFakeOpencode(t)
	t.Setenv("CC_RUN_OUTPUT", strings.Join([]string{
		`{"type":"step_start","part":{"sessionID":"ses_late"}}`,
		`{"type":"step_finish","part":{}}`,
		`{"type":"text","part":{"text":"late final answer"}}`,
	}, "\n"))

	s, err := newOpencodeSession(context.Background(), cmdPath, workDir, "", "default", "", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	if err := s.Send("analyze", nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	events := collectUntilResult(t, s.Events())
	if len(events) < 2 {
		t.Fatalf("events = %+v, want text and result", events)
	}
	if events[len(events)-1].Type != core.EventResult {
		t.Fatalf("last event type = %q, want %q", events[len(events)-1].Type, core.EventResult)
	}
	foundText := false
	for _, event := range events[:len(events)-1] {
		if event.Type == core.EventText && event.Content == "late final answer" {
			foundText = true
		}
	}
	if !foundText {
		t.Fatalf("events = %+v, want late final answer text before result", events)
	}
}

func TestOpencodeSessionRetriesWithoutStaleSession(t *testing.T) {
	workDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	binDir := t.TempDir()
	cmdPath := filepath.Join(binDir, "opencode")
	script := `#!/bin/sh
set -eu

cmd="${1:-}"
shift || true

case "$cmd" in
  run)
    if [ -n "${CC_ARGS_LOG:-}" ]; then
      printf '%s\n' "$cmd" "$@" >> "$CC_ARGS_LOG"
    fi
    joined=" $* "
    if printf '%s' "$joined" | grep -q ' --session '; then
      cat <<'EOF'
{"type":"error","error":{"message":"Session not found"}}
EOF
      exit 1
    fi
    cat <<'EOF'
{"type":"step_start","part":{"sessionID":"ses_rebound"}}
{"type":"text","part":{"text":"fresh response"}}
{"type":"step_finish","part":{}}
EOF
    ;;
  *)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}
	t.Setenv("CC_ARGS_LOG", logPath)

	s, err := newOpencodeSession(context.Background(), cmdPath, workDir, "", "default", "ses_stale", nil)
	if err != nil {
		t.Fatalf("newOpencodeSession() error = %v", err)
	}
	defer s.Close()

	if err := s.Send("hello again", nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	events := collectUntilResult(t, s.Events())
	if got := s.CurrentSessionID(); got != "ses_rebound" {
		t.Fatalf("CurrentSessionID() = %q, want %q", got, "ses_rebound")
	}

	hadText := false
	for _, event := range events {
		if event.Type == core.EventError {
			t.Fatalf("unexpected error event: %v", event.Error)
		}
		if event.Type == core.EventText && event.Content == "fresh response" {
			hadText = true
		}
	}
	if !hadText {
		t.Fatalf("events = %+v, want fresh response text event", events)
	}

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read args log: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(b)), "\n")
	var runs [][]string
	var current []string
	for _, arg := range args {
		if arg == "run" {
			if len(current) > 0 {
				runs = append(runs, current)
			}
			current = []string{arg}
			continue
		}
		current = append(current, arg)
	}
	if len(current) > 0 {
		runs = append(runs, current)
	}
	if len(runs) != 2 {
		t.Fatalf("run count = %d, want 2; args=%q", len(runs), args)
	}
	if !strings.Contains(strings.Join(runs[0], " "), "--session ses_stale") {
		t.Fatalf("first run args = %q, want stale session resume", strings.Join(runs[0], " "))
	}
	if strings.Contains(strings.Join(runs[1], " "), "--session") {
		t.Fatalf("second run args = %q, want retry without session flag", strings.Join(runs[1], " "))
	}
}
