package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/config"
	"github.com/chenhg5/cc-connect/core"
)

type smokeAgent struct{}

func (a *smokeAgent) Name() string { return "smoke-agent" }
func (a *smokeAgent) StartSession(context.Context, string) (core.AgentSession, error) {
	return &smokeAgentSession{}, nil
}
func (a *smokeAgent) ListSessions(context.Context) ([]core.AgentSessionInfo, error) { return nil, nil }
func (a *smokeAgent) Stop() error                                                   { return nil }

type smokeAgentSession struct{}

func (s *smokeAgentSession) Send(string, []core.ImageAttachment) error             { return nil }
func (s *smokeAgentSession) RespondPermission(string, core.PermissionResult) error { return nil }
func (s *smokeAgentSession) Events() <-chan core.Event                             { return make(chan core.Event) }
func (s *smokeAgentSession) CurrentSessionID() string                              { return "smoke" }
func (s *smokeAgentSession) Alive() bool                                           { return true }
func (s *smokeAgentSession) Close() error                                          { return nil }

type smokePlatform struct {
	handler            core.MessageHandler
	sent               []string
	sentReplyCtx       []any
	registeredCommands []core.BotCommandInfo
}

func (p *smokePlatform) Name() string { return "smoke-platform" }
func (p *smokePlatform) Start(handler core.MessageHandler) error {
	p.handler = handler
	return nil
}
func (p *smokePlatform) Reply(_ context.Context, _ any, content string) error {
	p.sent = append(p.sent, content)
	return nil
}
func (p *smokePlatform) Send(_ context.Context, replyCtx any, content string) error {
	p.sent = append(p.sent, content)
	p.sentReplyCtx = append(p.sentReplyCtx, replyCtx)
	return nil
}
func (p *smokePlatform) Stop() error { return nil }
func (p *smokePlatform) RegisterCommands(commands []core.BotCommandInfo) error {
	p.registeredCommands = append([]core.BotCommandInfo(nil), commands...)
	return nil
}
func (p *smokePlatform) ReconstructReplyCtx(sessionKey string) (any, error) {
	return "reply:" + sessionKey, nil
}

var lastSmokePlatform *smokePlatform

func init() {
	core.RegisterAgent("smoke-agent", func(map[string]any) (core.Agent, error) {
		return &smokeAgent{}, nil
	})
	core.RegisterPlatform("smoke-platform", func(map[string]any) (core.Platform, error) {
		lastSmokePlatform = &smokePlatform{}
		return lastSmokePlatform, nil
	})
}

func TestStartBotModeSmoke(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "repo-a", ".git"), 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	cfg := &config.Config{
		DataDir: filepath.Join(t.TempDir(), "data"),
		Workspace: config.WorkspaceConfig{
			Root: workspace,
		},
		Bots: []config.BotConfig{{
			Name:      "smoke-bot",
			AgentType: "smoke-agent",
			Platforms: []config.PlatformConfig{{
				Type:    "smoke-platform",
				Options: map[string]any{},
			}},
		}},
	}

	cleanup, err := startBotMode(cfg)
	if err != nil {
		t.Fatalf("startBotMode() error = %v", err)
	}
	defer cleanup()

	if lastSmokePlatform == nil || lastSmokePlatform.handler == nil {
		t.Fatal("smoke platform handler was not installed")
	}

	lastSmokePlatform.handler(lastSmokePlatform, &core.Message{
		Platform: "smoke-platform",
		UserID:   "u1",
		IsDM:     true,
		Content:  "/project list",
	})

	if len(lastSmokePlatform.sent) == 0 {
		t.Fatal("expected smoke platform to receive a reply")
	}
}

func TestStartBotModeRegistersPlatformCommands(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "repo-a", ".git"), 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	cfg := &config.Config{
		DataDir: filepath.Join(t.TempDir(), "data"),
		Workspace: config.WorkspaceConfig{
			Root: workspace,
		},
		Bots: []config.BotConfig{{
			Name:      "smoke-bot",
			AgentType: "smoke-agent",
			Platforms: []config.PlatformConfig{{
				Type:    "smoke-platform",
				Options: map[string]any{},
			}},
		}},
	}

	cleanup, err := startBotMode(cfg)
	if err != nil {
		t.Fatalf("startBotMode() error = %v", err)
	}
	defer cleanup()

	if lastSmokePlatform == nil {
		t.Fatal("expected smoke platform to be created")
	}
	if len(lastSmokePlatform.registeredCommands) == 0 {
		t.Fatal("expected bot mode to register platform commands")
	}
}

func TestStartBotModeSendsStartupNotificationToLastActiveSession(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "repo-a", ".git"), 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	dataDir := filepath.Join(t.TempDir(), "data")
	store := core.NewLastActiveSessionStore(filepath.Join(dataDir, "bot_last_active_sessions.json"))
	store.Set("smoke-bot", "smoke-platform", "smoke-platform:chat-1:user-1")

	cfg := &config.Config{
		DataDir: dataDir,
		Language: "zh",
		Workspace: config.WorkspaceConfig{
			Root: workspace,
		},
		Bots: []config.BotConfig{{
			Name:      "smoke-bot",
			AgentType: "smoke-agent",
			Platforms: []config.PlatformConfig{{
				Type:    "smoke-platform",
				Options: map[string]any{},
			}},
		}},
	}

	cleanup, err := startBotMode(cfg)
	if err != nil {
		t.Fatalf("startBotMode() error = %v", err)
	}
	defer cleanup()

	if lastSmokePlatform == nil {
		t.Fatal("expected smoke platform to be created")
	}
	if len(lastSmokePlatform.sent) == 0 {
		t.Fatal("expected startup notification to be sent")
	}
	if got := lastSmokePlatform.sent[len(lastSmokePlatform.sent)-1]; got != "✅ ccc 已启动。" {
		t.Fatalf("startup notification = %q, want startup success message", got)
	}
	if len(lastSmokePlatform.sentReplyCtx) == 0 || lastSmokePlatform.sentReplyCtx[len(lastSmokePlatform.sentReplyCtx)-1] != "reply:smoke-platform:chat-1:user-1" {
		t.Fatalf("reply ctx = %v, want reconstructed last-active session", lastSmokePlatform.sentReplyCtx)
	}
}

func TestWaitForShutdownOrRestartReturnsRestartRequest(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	restartCh := make(chan core.RestartRequest, 1)

	go func() {
		restartCh <- core.RestartRequest{
			SessionKey: "telegram:chat-1:user-1",
			Platform:   "telegram",
		}
	}()

	req := waitForShutdownOrRestart(sigCh, restartCh)
	if req == nil {
		t.Fatal("expected restart request")
	}
	if req.SessionKey != "telegram:chat-1:user-1" || req.Platform != "telegram" {
		t.Fatalf("restart request = %+v, want telegram chat request", req)
	}
}

func TestRestartCurrentProcessSavesNotifyWithoutExec(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	cfg := &config.Config{DataDir: dataDir}
	req := &core.RestartRequest{
		SessionKey: "telegram:chat-1:user-1",
		Platform:   "telegram",
	}

	orig := restartProcessFn
	defer func() { restartProcessFn = orig }()

	called := false
	var gotExec string
	restartProcessFn = func(execPath string) error {
		called = true
		gotExec = execPath
		return nil
	}

	if err := restartCurrentProcess(cfg, req); err != nil {
		t.Fatalf("restartCurrentProcess() error = %v", err)
	}
	if !called {
		t.Fatal("expected restart process to be invoked")
	}
	if gotExec == "" {
		t.Fatal("expected executable path to be passed to restart process")
	}
	notify := core.ConsumeRestartNotify(dataDir)
	if notify == nil {
		t.Fatal("expected restart notification file to be saved")
	}
	if notify.SessionKey != req.SessionKey || notify.Platform != req.Platform {
		t.Fatalf("notify = %+v, want %+v", notify, req)
	}
}

func TestRunBotModeExitsOnRestartRequest(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "repo-a", ".git"), 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	cfg := &config.Config{
		DataDir: filepath.Join(t.TempDir(), "data"),
		Workspace: config.WorkspaceConfig{
			Root: workspace,
		},
		Bots: []config.BotConfig{{
			Name:      "smoke-bot",
			AgentType: "smoke-agent",
			Platforms: []config.PlatformConfig{{
				Type:    "smoke-platform",
				Options: map[string]any{},
			}},
		}},
	}

	orig := restartProcessFn
	defer func() { restartProcessFn = orig }()
	restartProcessFn = func(string) error { return nil }

	done := make(chan error, 1)
	go func() {
		done <- runBotMode(cfg)
	}()

	time.Sleep(100 * time.Millisecond)
	core.RestartCh <- core.RestartRequest{
		SessionKey: "telegram:chat-1:user-1",
		Platform:   "telegram",
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runBotMode() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runBotMode() did not exit on restart request")
	}
}
