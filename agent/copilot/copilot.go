package copilot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chenhg5/cc-connect/core"
)

func init() {
	core.RegisterAgent("copilot", New)
}

// Agent drives GitHub Copilot CLI through its ACP server mode (`copilot --acp --stdio`).
//
// Modes are applied after session creation using ACP session mode/config APIs when
// the server advertises them. Known user-facing aliases:
//   - "default" / "ask": standard interactive mode
//   - "plan": plan-first mode
//   - "allow-all": auto-approve tool/path/url permissions
//   - "yolo": broad autonomous mode
type Agent struct {
	workDir    string
	cmd        string
	model      string
	mode       string
	sessionEnv []string
	mu         sync.Mutex
}

func New(opts map[string]any) (core.Agent, error) {
	workDir, _ := opts["work_dir"].(string)
	if workDir == "" {
		workDir = "."
	}

	cmd, _ := opts["cmd"].(string)
	if cmd == "" {
		cmd = "copilot"
	}

	model, _ := opts["model"].(string)
	mode, _ := opts["mode"].(string)
	mode = normalizeMode(mode)

	if _, err := exec.LookPath(cmd); err != nil {
		return nil, fmt.Errorf("copilot: %q CLI not found in PATH, install GitHub Copilot CLI first", cmd)
	}

	return &Agent{
		workDir: workDir,
		cmd:     cmd,
		model:   strings.TrimSpace(model),
		mode:    mode,
	}, nil
}

func normalizeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "default", "ask":
		return "ask"
	case "plan":
		return "plan"
	case "allow-all", "allow_all", "auto":
		return "allow-all"
	case "yolo":
		return "yolo"
	default:
		return strings.TrimSpace(raw)
	}
}

func (a *Agent) Name() string { return "copilot" }

func (a *Agent) StartSession(ctx context.Context, sessionID string) (core.AgentSession, error) {
	a.mu.Lock()
	cmd := a.cmd
	workDir := a.workDir
	mode := a.mode
	model := a.model
	extraEnv := append([]string(nil), a.sessionEnv...)
	a.mu.Unlock()

	return newCopilotSession(ctx, cmd, workDir, model, mode, sessionID, extraEnv)
}

func (a *Agent) ListSessions(_ context.Context) ([]core.AgentSessionInfo, error) {
	// Copilot ACP does not currently expose a stable session discovery API that this
	// project can rely on. cc-connect still persists active session IDs in its own
	// session store, so resume works without agent-side listing.
	return nil, nil
}

func (a *Agent) Stop() error { return nil }

func (a *Agent) SetSessionEnv(env []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessionEnv = append([]string(nil), env...)
}

func (a *Agent) SetMode(mode string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mode = normalizeMode(mode)
	slog.Info("copilot: mode changed", "mode", a.mode)
}

func (a *Agent) GetMode() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mode
}

func (a *Agent) PermissionModes() []core.PermissionModeInfo {
	return []core.PermissionModeInfo{
		{Key: "ask", Name: "Ask", NameZh: "询问", Desc: "Ask before sensitive operations", DescZh: "敏感操作前先询问"},
		{Key: "plan", Name: "Plan", NameZh: "规划模式", Desc: "Plan first before making changes", DescZh: "先规划，再决定是否执行"},
		{Key: "allow-all", Name: "Allow All", NameZh: "自动批准", Desc: "Auto-approve tool and path permissions", DescZh: "自动批准工具和路径权限"},
		{Key: "yolo", Name: "YOLO", NameZh: "全自动", Desc: "Maximum autonomy with broad approvals", DescZh: "最大化自治并放宽权限"},
	}
}

func (a *Agent) SetModel(model string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.model = strings.TrimSpace(model)
	slog.Info("copilot: model changed", "model", a.model)
}

func (a *Agent) GetModel() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.model
}

func (a *Agent) AvailableModels(_ context.Context) []core.ModelOption {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.model == "" {
		return nil
	}
	return []core.ModelOption{{Name: a.model}}
}

func (a *Agent) ProjectMemoryFile() string {
	absDir, err := filepath.Abs(a.workDir)
	if err != nil {
		absDir = a.workDir
	}
	return filepath.Join(absDir, ".github", "copilot-instructions.md")
}

func (a *Agent) GlobalMemoryFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".copilot", "copilot-instructions.md")
	}
	return filepath.Join(homeDir, ".copilot", "copilot-instructions.md")
}
