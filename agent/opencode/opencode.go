package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

func init() {
	core.RegisterAgent("opencode", New)
}

// Agent drives the OpenCode CLI in headless mode using `opencode run --format json`.
//
// Modes:
//   - "default": standard non-interactive mode
//   - "yolo":    explicit auto mode (same runtime today; kept for UI consistency)
type Agent struct {
	workDir    string
	model      string
	mode       string
	cmd        string // CLI binary name, default "opencode"
	providers  []core.ProviderConfig
	activeIdx  int
	sessionEnv []string
	sessions   map[*opencodeSession]struct{}
	mu         sync.Mutex
}

func New(opts map[string]any) (core.Agent, error) {
	workDir, _ := opts["work_dir"].(string)
	if workDir == "" {
		workDir = "."
	}
	model, _ := opts["model"].(string)
	mode, _ := opts["mode"].(string)
	mode = normalizeMode(mode)
	cmd, _ := opts["cmd"].(string)
	if cmd == "" {
		cmd = "opencode"
	}

	if _, err := exec.LookPath(cmd); err != nil {
		return nil, fmt.Errorf("opencode: %q CLI not found in PATH, install from: https://github.com/opencode-ai/opencode", cmd)
	}

	return &Agent{
		workDir:   workDir,
		model:     model,
		mode:      mode,
		cmd:       cmd,
		activeIdx: -1,
		sessions:  make(map[*opencodeSession]struct{}),
	}, nil
}

func normalizeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "yolo", "auto", "force", "bypasspermissions":
		return "yolo"
	default:
		return "default"
	}
}

func (a *Agent) Name() string { return "opencode" }

func (a *Agent) SetModel(model string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.model = model
	slog.Info("opencode: model changed", "model", model)
}

func (a *Agent) GetModel() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.model
}

func (a *Agent) AvailableModels(_ context.Context) []core.ModelOption {
	return []core.ModelOption{
		{Name: "anthropic/claude-sonnet-4-20250514", Desc: "Claude Sonnet 4 (default)"},
		{Name: "anthropic/claude-opus-4-20250514", Desc: "Claude Opus 4"},
		{Name: "openai/gpt-4o", Desc: "GPT-4o"},
		{Name: "openai/o3", Desc: "OpenAI o3"},
	}
}

func (a *Agent) SetSessionEnv(env []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessionEnv = env
}

func (a *Agent) StartSession(ctx context.Context, sessionID string) (core.AgentSession, error) {
	a.mu.Lock()
	model := a.model
	mode := a.mode
	cmd := a.cmd
	workDir := a.workDir
	extraEnv := a.providerEnvLocked()
	extraEnv = append(extraEnv, a.sessionEnv...)
	if a.activeIdx >= 0 && a.activeIdx < len(a.providers) {
		if m := a.providers[a.activeIdx].Model; m != "" {
			model = m
		}
	}
	a.mu.Unlock()

	session, err := newOpencodeSession(ctx, cmd, workDir, model, mode, sessionID, extraEnv)
	if err != nil {
		return nil, err
	}
	a.trackSession(session)
	session.onClose = a.untrackSession
	return session, nil
}

// ListSessions runs `opencode session list` and parses the JSON output.
func (a *Agent) ListSessions(_ context.Context) ([]core.AgentSessionInfo, error) {
	return listOpencodeSessions(a.cmd, a.workDir)
}

func (a *Agent) DeleteSession(_ context.Context, sessionID string) error {
	c := exec.Command(a.cmd, "session", "delete", sessionID)
	c.Dir = a.workDir
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("opencode: delete session %s: %s", sessionID, msg)
	}
	return nil
}

func (a *Agent) Stop() error {
	a.mu.Lock()
	sessions := make([]*opencodeSession, 0, len(a.sessions))
	for s := range a.sessions {
		sessions = append(sessions, s)
	}
	a.mu.Unlock()

	var errs []string
	for _, s := range sessions {
		if err := s.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("opencode: stop sessions: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (a *Agent) trackSession(s *opencodeSession) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sessions == nil {
		a.sessions = make(map[*opencodeSession]struct{})
	}
	a.sessions[s] = struct{}{}
}

func (a *Agent) untrackSession(s *opencodeSession) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, s)
}

// -- ModeSwitcher --

func (a *Agent) SetMode(mode string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mode = normalizeMode(mode)
	slog.Info("opencode: mode changed", "mode", a.mode)
}

func (a *Agent) GetMode() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mode
}

func (a *Agent) PermissionModes() []core.PermissionModeInfo {
	return []core.PermissionModeInfo{
		{Key: "default", Name: "Default", NameZh: "默认", Desc: "Standard mode", DescZh: "标准模式"},
		{Key: "yolo", Name: "YOLO", NameZh: "全自动", Desc: "Auto-approve all tool calls", DescZh: "自动批准所有工具调用"},
	}
}

// -- ContextCompressor --

func (a *Agent) CompressCommand() string { return "/compact" }

// -- CommandProvider --

func (a *Agent) CommandDirs() []string {
	absDir, err := filepath.Abs(a.workDir)
	if err != nil {
		absDir = a.workDir
	}
	dirs := []string{filepath.Join(absDir, ".opencode", "commands")}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		dirs = append(dirs, filepath.Join(xdg, "opencode", "commands"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs,
			filepath.Join(home, ".config", "opencode", "commands"),
			filepath.Join(home, ".opencode", "commands"),
		)
	}
	return uniqueStrings(dirs)
}

// -- MemoryFileProvider --

func (a *Agent) ProjectMemoryFile() string {
	absDir, err := filepath.Abs(a.workDir)
	if err != nil {
		absDir = a.workDir
	}
	return filepath.Join(absDir, "OpenCode.md")
}

func (a *Agent) GlobalMemoryFile() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode", "OpenCode.md")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".opencode", "OpenCode.md")
}

// -- ProviderSwitcher --

func (a *Agent) SetProviders(providers []core.ProviderConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.providers = providers
}

func (a *Agent) SetActiveProvider(name string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if name == "" {
		a.activeIdx = -1
		slog.Info("opencode: provider cleared")
		return true
	}
	for i, p := range a.providers {
		if p.Name == name {
			a.activeIdx = i
			slog.Info("opencode: provider switched", "provider", name)
			return true
		}
	}
	return false
}

func (a *Agent) GetActiveProvider() *core.ProviderConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeIdx < 0 || a.activeIdx >= len(a.providers) {
		return nil
	}
	p := a.providers[a.activeIdx]
	return &p
}

func (a *Agent) ListProviders() []core.ProviderConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]core.ProviderConfig, len(a.providers))
	copy(result, a.providers)
	return result
}

func (a *Agent) providerEnvLocked() []string {
	if a.activeIdx < 0 || a.activeIdx >= len(a.providers) {
		return nil
	}
	p := a.providers[a.activeIdx]
	env := providerEnvForOpenCode(p)
	for k, v := range p.Env {
		env = append(env, k+"="+v)
	}
	return env
}

func providerEnvForOpenCode(p core.ProviderConfig) []string {
	var env []string
	provider := detectProviderKind(p)
	if p.APIKey != "" {
		switch provider {
		case "openai":
			env = append(env, "OPENAI_API_KEY="+p.APIKey)
		case "gemini":
			env = append(env, "GEMINI_API_KEY="+p.APIKey)
		case "groq":
			env = append(env, "GROQ_API_KEY="+p.APIKey)
		case "azure":
			env = append(env, "AZURE_OPENAI_API_KEY="+p.APIKey)
		case "openrouter":
			env = append(env, "OPENROUTER_API_KEY="+p.APIKey)
		default:
			env = append(env, "ANTHROPIC_API_KEY="+p.APIKey)
		}
	}
	if p.BaseURL != "" {
		switch provider {
		case "azure":
			env = append(env, "AZURE_OPENAI_ENDPOINT="+p.BaseURL)
		case "local":
			env = append(env, "LOCAL_ENDPOINT="+p.BaseURL)
		}
	}
	return env
}

func detectProviderKind(p core.ProviderConfig) string {
	name := strings.ToLower(strings.TrimSpace(p.Name))
	model := strings.ToLower(strings.TrimSpace(p.Model))
	switch {
	case strings.HasPrefix(model, "openai/") || strings.Contains(name, "openai"):
		return "openai"
	case strings.HasPrefix(model, "gemini/") || strings.Contains(name, "gemini") || strings.Contains(name, "google"):
		return "gemini"
	case strings.HasPrefix(model, "groq/") || strings.Contains(name, "groq"):
		return "groq"
	case strings.HasPrefix(model, "azure/") || strings.Contains(name, "azure"):
		return "azure"
	case strings.HasPrefix(model, "openrouter/") || strings.Contains(name, "openrouter"):
		return "openrouter"
	case strings.HasPrefix(model, "local/") || strings.Contains(name, "local") || strings.Contains(name, "self-host"):
		return "local"
	default:
		return "anthropic"
	}
}

// -- Session listing --

// opencodeSessionEntry represents a session from `opencode session list` output.
type opencodeSessionEntry struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Updated any    `json:"updated"`
}

func listOpencodeSessions(cmd, workDir string) ([]core.AgentSessionInfo, error) {
	c := exec.Command(cmd, "session", "list", "--format", "json")
	c.Dir = workDir

	out, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("opencode: session list: %w", err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, nil
	}

	var entries []opencodeSessionEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("opencode: parse session list: %w", err)
	}

	var sessions []core.AgentSessionInfo
	for _, e := range entries {
		modTime := parseOpencodeTime(e.Updated)
		sessions = append(sessions, core.AgentSessionInfo{
			ID:         e.ID,
			Summary:    e.Title,
			ModifiedAt: modTime,
		})
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModifiedAt.After(sessions[j].ModifiedAt)
	})

	return sessions, nil
}

func parseOpencodeTime(raw any) time.Time {
	switch v := raw.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return time.Time{}
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if ts, err := time.Parse(layout, v); err == nil {
				return ts
			}
		}
	case float64:
		return parseUnixTimestamp(v)
	case int64:
		return parseUnixTimestamp(float64(v))
	case int:
		return parseUnixTimestamp(float64(v))
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return parseUnixTimestamp(f)
		}
	}
	return time.Time{}
}

func parseUnixTimestamp(v float64) time.Time {
	if v <= 0 {
		return time.Time{}
	}
	sec := int64(v)
	if sec >= 1_000_000_000_000_000 {
		return time.UnixMicro(sec)
	}
	nsec := int64((v - float64(sec)) * float64(time.Second))
	if sec >= 1_000_000_000_000 {
		return time.UnixMilli(sec)
	}
	return time.Unix(sec, nsec)
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
