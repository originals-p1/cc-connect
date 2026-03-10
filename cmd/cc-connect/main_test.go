package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
	handler core.MessageHandler
	sent    []string
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
func (p *smokePlatform) Send(context.Context, any, string) error { return nil }
func (p *smokePlatform) Stop() error                             { return nil }

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
