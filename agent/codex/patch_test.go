package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchSessionSource(t *testing.T) {
	tmpDir := t.TempDir()

	sessionID := "test-session-abc123"
	sessionsDir := filepath.Join(tmpDir, ".codex", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fname := filepath.Join(sessionsDir, "rollout-"+sessionID+".jsonl")
	line1 := `{"timestamp":"2026-01-01T00:00:00Z","type":"session_meta","payload":{"id":"` + sessionID + `","source":"exec","originator":"codex_exec","cwd":"/tmp"}}`
	line2 := `{"timestamp":"2026-01-01T00:00:01Z","type":"response_item","payload":{"role":"user"}}`
	content := line1 + "\n" + line2 + "\n"

	if err := os.WriteFile(fname, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override CODEX_HOME for test
	t.Setenv("CODEX_HOME", filepath.Join(tmpDir, ".codex"))

	patchSessionSource(sessionID)

	data, err := os.ReadFile(fname)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.SplitN(string(data), "\n", 2)

	if !strings.Contains(lines[0], `"source":"cli"`) {
		t.Errorf("expected source:cli, got first line: %s", lines[0])
	}
	if !strings.Contains(lines[0], `"originator":"codex_cli_rs"`) {
		t.Errorf("expected originator:codex_cli_rs, got first line: %s", lines[0])
	}
	if strings.Contains(lines[0], `"source":"exec"`) {
		t.Error("source:exec was not replaced")
	}

	// Second line should be untouched
	if !strings.HasPrefix(lines[1], `{"timestamp":"2026-01-01T00:00:01Z"`) {
		t.Errorf("second line was corrupted: %s", lines[1])
	}
}

func TestPatchSessionSource_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-idempotent-xyz"
	sessionsDir := filepath.Join(tmpDir, ".codex", "sessions")
	os.MkdirAll(sessionsDir, 0o755)

	fname := filepath.Join(sessionsDir, "rollout-"+sessionID+".jsonl")
	line1 := `{"type":"session_meta","payload":{"id":"` + sessionID + `","source":"cli","originator":"codex_cli_rs"}}`
	if err := os.WriteFile(fname, []byte(line1+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", filepath.Join(tmpDir, ".codex"))

	patchSessionSource(sessionID)

	data, _ := os.ReadFile(fname)
	if string(data) != line1+"\n" {
		t.Errorf("file was modified when it shouldn't have been")
	}
}

func TestFindSessionFile_ExactMatch(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, ".codex", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sessionID := "abc-def-123"

	// Create a file with an overlapping ID (substring match bug)
	fname1 := filepath.Join(sessionsDir, "xyz-"+sessionID+"-extra.jsonl")
	line1 := `{"type":"session_meta","payload":{"id":"xyz-abc-def-123-extra","cwd":"/tmp"}}`
	if err := os.WriteFile(fname1, []byte(line1+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create the correct file
	fname2 := filepath.Join(sessionsDir, "rollout-"+sessionID+".jsonl")
	line2 := `{"type":"session_meta","payload":{"id":"` + sessionID + `","cwd":"/tmp"}}`
	if err := os.WriteFile(fname2, []byte(line2+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", filepath.Join(tmpDir, ".codex"))

	found := findSessionFile(sessionID)
	if found != fname2 {
		t.Errorf("findSessionFile(%q) = %q, want %q", sessionID, found, fname2)
	}
}

func TestFindSessionFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, ".codex", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sessionID := "nonexistent-session"
	t.Setenv("CODEX_HOME", filepath.Join(tmpDir, ".codex"))

	found := findSessionFile(sessionID)
	if found != "" {
		t.Errorf("findSessionFile(%q) = %q, want empty string", sessionID, found)
	}
}
