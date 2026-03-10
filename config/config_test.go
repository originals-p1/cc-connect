package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadBotsConfig(t *testing.T) {
	path := writeTempConfig(t, `
[workspace]
root = "/tmp/code"
require_git = true

[[bots]]
name = "codex"
agent_type = "codex"
default_project = "repo-a"
max_cached_sessions = 3
dm_only = true

[bots.agent_options]
model = "gpt-5-codex"

[[bots.platforms]]
type = "telegram"
[bots.platforms.options]
token = "xxx"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Workspace.Root != "/tmp/code" {
		t.Fatalf("Workspace.Root = %q, want /tmp/code", cfg.Workspace.Root)
	}
	if cfg.Workspace.RequireGit == nil || !*cfg.Workspace.RequireGit {
		t.Fatalf("Workspace.RequireGit = %v, want true", cfg.Workspace.RequireGit)
	}
	if len(cfg.Bots) != 1 {
		t.Fatalf("len(Bots) = %d, want 1", len(cfg.Bots))
	}
	bot := cfg.Bots[0]
	if bot.Name != "codex" {
		t.Fatalf("bot.Name = %q, want codex", bot.Name)
	}
	if bot.AgentType != "codex" {
		t.Fatalf("bot.AgentType = %q, want codex", bot.AgentType)
	}
	if bot.DefaultProject != "repo-a" {
		t.Fatalf("bot.DefaultProject = %q, want repo-a", bot.DefaultProject)
	}
	if bot.MaxCachedSessions != 3 {
		t.Fatalf("bot.MaxCachedSessions = %d, want 3", bot.MaxCachedSessions)
	}
	if bot.DMOnly == nil || !*bot.DMOnly {
		t.Fatalf("bot.DMOnly = %v, want true", bot.DMOnly)
	}
	if got := bot.AgentOptions["model"]; got != "gpt-5-codex" {
		t.Fatalf("bot.AgentOptions[model] = %v, want gpt-5-codex", got)
	}
	if len(bot.Platforms) != 1 {
		t.Fatalf("len(bot.Platforms) = %d, want 1", len(bot.Platforms))
	}
	if bot.Platforms[0].Type != "telegram" {
		t.Fatalf("bot.Platforms[0].Type = %q, want telegram", bot.Platforms[0].Type)
	}
}

func TestValidateBotsRequiresWorkspaceRoot(t *testing.T) {
	path := writeTempConfig(t, `
[[bots]]
name = "codex"
agent_type = "codex"

[[bots.platforms]]
type = "telegram"
[bots.platforms.options]
token = "xxx"
`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "workspace.root") {
		t.Fatalf("Load() error = %v, want workspace.root validation error", err)
	}
}

func TestValidateBotsRequiresAgentTypeAndPlatform(t *testing.T) {
	path := writeTempConfig(t, `
[workspace]
root = "/tmp/code"

[[bots]]
name = "codex"
`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "bots[0].agent_type") {
		t.Fatalf("Load() error = %v, want bots[0].agent_type validation error", err)
	}
}
