package opencode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

func TestProviderEnvForOpenCode(t *testing.T) {
	tests := []struct {
		name     string
		provider core.ProviderConfig
		want     []string
	}{
		{
			name:     "anthropic default",
			provider: core.ProviderConfig{Name: "anthropic-main", APIKey: "ak"},
			want:     []string{"ANTHROPIC_API_KEY=ak"},
		},
		{
			name:     "openai by model prefix",
			provider: core.ProviderConfig{Name: "custom", APIKey: "ok", Model: "openai/gpt-4.1"},
			want:     []string{"OPENAI_API_KEY=ok"},
		},
		{
			name:     "azure base url",
			provider: core.ProviderConfig{Name: "azure-prod", APIKey: "zk", BaseURL: "https://azure.example"},
			want:     []string{"AZURE_OPENAI_API_KEY=zk", "AZURE_OPENAI_ENDPOINT=https://azure.example"},
		},
		{
			name:     "local endpoint",
			provider: core.ProviderConfig{Name: "local-llm", BaseURL: "http://localhost:1234/v1"},
			want:     []string{"LOCAL_ENDPOINT=http://localhost:1234/v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providerEnvForOpenCode(tt.provider)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("providerEnvForOpenCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandDirs(t *testing.T) {
	workDir := t.TempDir()
	homeDir := t.TempDir()
	xdgDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", homeDir)
	}

	a := &Agent{workDir: workDir}
	got := a.CommandDirs()
	want := []string{
		filepath.Join(workDir, ".opencode", "commands"),
		filepath.Join(xdgDir, "opencode", "commands"),
		filepath.Join(homeDir, ".config", "opencode", "commands"),
		filepath.Join(homeDir, ".opencode", "commands"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CommandDirs() = %v, want %v", got, want)
	}
}

func TestMemoryFiles(t *testing.T) {
	workDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", homeDir)
	}

	a := &Agent{workDir: workDir}
	if got, want := a.ProjectMemoryFile(), filepath.Join(workDir, "OpenCode.md"); got != want {
		t.Fatalf("ProjectMemoryFile() = %q, want %q", got, want)
	}
	if got, want := a.GlobalMemoryFile(), filepath.Join(homeDir, ".opencode", "OpenCode.md"); got != want {
		t.Fatalf("GlobalMemoryFile() = %q, want %q", got, want)
	}
}

func TestParseOpencodeTime(t *testing.T) {
	ts := parseOpencodeTime("2026-03-12T11:22:33Z")
	if ts.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
	if got, want := ts.UTC().Format(time.RFC3339), "2026-03-12T11:22:33Z"; got != want {
		t.Fatalf("parseOpencodeTime() = %q, want %q", got, want)
	}

	ts = parseOpencodeTime(float64(1741735353))
	if got, want := ts.UTC().Format(time.RFC3339), "2025-03-11T23:22:33Z"; got != want {
		t.Fatalf("parseOpencodeTime(float64) = %q, want %q", got, want)
	}

	ts = parseOpencodeTime(json.Number("1741735353000"))
	if got, want := ts.UTC().Format(time.RFC3339), "2025-03-11T23:22:33Z"; got != want {
		t.Fatalf("parseOpencodeTime(json.Number) = %q, want %q", got, want)
	}
}

func TestListOpencodeSessionsSortsByUpdatedDesc(t *testing.T) {
	binDir := t.TempDir()
	workDir := t.TempDir()
	cmdPath := filepath.Join(binDir, "opencode")
	script := "#!/bin/sh\nif [ \"$1\" = \"session\" ] && [ \"$2\" = \"list\" ]; then\ncat <<'EOF'\n[{\"id\":\"old\",\"title\":\"Old\",\"updated\":\"2026-03-10T01:00:00Z\"},{\"id\":\"new\",\"title\":\"New\",\"updated\":\"2026-03-12T02:00:00Z\"}]\nEOF\nexit 0\nfi\nexit 1\n"
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}

	sessions, err := listOpencodeSessions(cmdPath, workDir)
	if err != nil {
		t.Fatalf("listOpencodeSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("session count = %d, want 2", len(sessions))
	}
	if sessions[0].ID != "new" || sessions[1].ID != "old" {
		t.Fatalf("session order = %v, want [new old]", []string{sessions[0].ID, sessions[1].ID})
	}
}

func TestListOpencodeSessionsAcceptsNumericUpdated(t *testing.T) {
	binDir := t.TempDir()
	workDir := t.TempDir()
	cmdPath := filepath.Join(binDir, "opencode")
	script := "#!/bin/sh\nif [ \"$1\" = \"session\" ] && [ \"$2\" = \"list\" ]; then\ncat <<'EOF'\n[{\"id\":\"old\",\"title\":\"Old\",\"updated\":1741735352},{\"id\":\"new\",\"title\":\"New\",\"updated\":1741735353}]\nEOF\nexit 0\nfi\nexit 1\n"
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}

	sessions, err := listOpencodeSessions(cmdPath, workDir)
	if err != nil {
		t.Fatalf("listOpencodeSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("session count = %d, want 2", len(sessions))
	}
	if sessions[0].ID != "new" || sessions[1].ID != "old" {
		t.Fatalf("session order = %v, want [new old]", []string{sessions[0].ID, sessions[1].ID})
	}
}

func TestListOpencodeSessionsAllowsEmptyOutput(t *testing.T) {
	binDir := t.TempDir()
	workDir := t.TempDir()
	cmdPath := filepath.Join(binDir, "opencode")
	script := "#!/bin/sh\nif [ \"$1\" = \"session\" ] && [ \"$2\" = \"list\" ]; then\nexit 0\nfi\nexit 1\n"
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}

	sessions, err := listOpencodeSessions(cmdPath, workDir)
	if err != nil {
		t.Fatalf("listOpencodeSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("session count = %d, want 0", len(sessions))
	}
}

func TestListOpencodeSessionsAllowsWhitespaceOnlyOutput(t *testing.T) {
	binDir := t.TempDir()
	workDir := t.TempDir()
	cmdPath := filepath.Join(binDir, "opencode")
	script := "#!/bin/sh\nif [ \"$1\" = \"session\" ] && [ \"$2\" = \"list\" ]; then\nprintf '\\n  \\t  \\n'\nexit 0\nfi\nexit 1\n"
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}

	sessions, err := listOpencodeSessions(cmdPath, workDir)
	if err != nil {
		t.Fatalf("listOpencodeSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("session count = %d, want 0", len(sessions))
	}
}

func TestDeleteSessionRunsCLI(t *testing.T) {
	binDir := t.TempDir()
	workDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "delete.log")
	cmdPath := filepath.Join(binDir, "opencode")
	script := "#!/bin/sh\nif [ \"$1\" = \"session\" ] && [ \"$2\" = \"delete\" ]; then\nprintf '%s' \"$3\" > \"$CC_MARKER\"\nexit 0\nfi\necho unexpected >&2\nexit 1\n"
	if err := os.WriteFile(cmdPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}
	t.Setenv("CC_MARKER", marker)

	a := &Agent{cmd: cmdPath, workDir: workDir}
	if err := a.DeleteSession(nil, "ses_123"); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := string(b); got != "ses_123" {
		t.Fatalf("deleted session = %q, want %q", got, "ses_123")
	}
}

func TestAgentStopClosesTrackedSessions(t *testing.T) {
	a := &Agent{sessions: make(map[*opencodeSession]struct{})}
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()
	s1 := &opencodeSession{ctx: ctx1, cancel: cancel1, events: make(chan core.Event), onClose: a.untrackSession}
	s1.alive.Store(true)
	s2 := &opencodeSession{ctx: ctx2, cancel: cancel2, events: make(chan core.Event), onClose: a.untrackSession}
	s2.alive.Store(true)
	a.trackSession(s1)
	a.trackSession(s2)

	if err := a.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if s1.Alive() || s2.Alive() {
		t.Fatalf("sessions should be marked closed after Stop(): s1=%v s2=%v", s1.Alive(), s2.Alive())
	}
	if got := len(a.sessions); got != 0 {
		t.Fatalf("tracked sessions = %d, want 0", got)
	}
}

func TestSkillDirs(t *testing.T) {
	workDir := t.TempDir()
	homeDir := t.TempDir()
	xdgDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", homeDir)
	}

	a := &Agent{workDir: workDir}
	got := a.SkillDirs()
	want := []string{
		filepath.Join(workDir, ".opencode", "skills"),
		filepath.Join(xdgDir, "opencode", "skills"),
		filepath.Join(homeDir, ".config", "opencode", "skills"),
		filepath.Join(homeDir, ".opencode", "skills"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SkillDirs() = %v, want %v", got, want)
	}
}
