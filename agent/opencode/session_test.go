package opencode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/chenhg5/cc-connect/core"
)

func TestOpencodeSessionSendUsesModeAndImages(t *testing.T) {
	binDir := t.TempDir()
	workDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "args.log")
	cmdPath := filepath.Join(binDir, "opencode")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$CC_ARGS_LOG\"\ncat <<'EOF'\n{\"type\":\"step_start\",\"part\":{\"sessionID\":\"ses_test\"}}\n{\"type\":\"step_finish\",\"part\":{}}\nEOF\n"
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}
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

	for event := range s.Events() {
		if event.Type == core.EventResult {
			break
		}
	}

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read args log: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(b)), "\n")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--agent coder") {
		t.Fatalf("args = %q, want yolo mode to add --agent coder", joined)
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
	s, err := newOpencodeSession(context.Background(), "opencode", ".", "", "default", "", nil)
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
	s, err := newOpencodeSession(context.Background(), "opencode", ".", "", "default", "", nil)
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
	s, err := newOpencodeSession(context.Background(), "opencode", ".", "", "default", "", nil)
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
