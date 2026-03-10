# AGENTS.md

This file is for coding agents working in this repository. Use it as the execution guide. For deeper architecture and domain context, read `docs/AI_CONTEXT.md`.

## Project Snapshot

- Language: Go
- Entry point: `cmd/cc-connect/main.go`
- Core flow: platform receives message -> `core.Engine` routes it -> agent session processes it -> platform sends replies back
- Main areas:
  - `cmd/cc-connect/`: CLI entrypoint and subcommands
  - `core/`: engine, sessions, commands, events, registries, shared infrastructure
  - `config/`: TOML config schema, validation, persistence helpers
  - `agent/`: integrations for Claude Code, Codex, Cursor, Gemini, Qoder, OpenCode, iFlow
  - `platform/`: chat platform integrations
  - `daemon/`: daemon/service management

## Where To Change Code

- Add or modify an agent:
  - `agent/<name>/`
  - `core/interfaces.go`
  - `core/registry.go`
  - `cmd/cc-connect/main.go`

- Add or modify a platform:
  - `platform/<name>/`
  - `core/interfaces.go`
  - `core/registry.go`
  - `cmd/cc-connect/main.go`

- Change message routing, slash commands, sessions, or event handling:
  - `core/engine.go`
  - `core/session.go`
  - `core/message.go`
  - `core/i18n.go`

- Change config schema or persisted settings:
  - `config/config.go`
  - `config.example.toml`
  - relevant docs in `README.md` or `docs/`

- Change CLI behavior or subcommands:
  - `cmd/cc-connect/*.go`

- Change relay, cron, provider, STT/TTS, or API helpers:
  - `core/relay.go`
  - `core/cron.go`
  - `cmd/cc-connect/provider.go`
  - `core/speech.go`
  - `core/tts.go`
  - `core/api.go`

## Working Rules

- Read the relevant package before editing. This repo uses package-level conventions heavily.
- New agents and platforms must register themselves in `init()` and must also be blank-imported in `cmd/cc-connect/main.go`.
- Do not change config shape in one place only. Keep schema, validation, defaulting, example config, and user-facing docs aligned.
- If a change affects slash commands, help text, or user-visible messages, check `core/i18n.go`.
- Respect the project model: one `[[projects]]` entry binds one agent to one or more platforms.
- Preserve existing package patterns unless there is a clear reason to refactor. This codebase prefers consistency over clever abstractions.
- Keep docs changes focused. `AGENTS.md` is for agent workflow; `docs/AI_CONTEXT.md` is the deeper reference.

## Validation

- Preferred full validation:
  - `go test ./...`
  - `make build`

- If you touch a narrow area, run targeted package tests first, then run broader validation if the change can affect shared flow.
- Do not claim completion without at least one concrete verification command.

## Documentation Sync

Update docs when behavior changes:

- `config.example.toml` for config additions or option changes
- `README.md` for install, usage, support matrix, or top-level workflow changes
- `docs/*.md` for platform-specific behavior
- `docs/AI_CONTEXT.md` only when structural project knowledge has changed materially

## Common Gotchas

- Forgetting the blank import in `cmd/cc-connect/main.go` will make a new agent or platform invisible at runtime.
- Config changes often need updates in more than one place: struct tags, validation, defaults, save/load helpers, and docs.
- User-facing command changes often require matching text updates in `core/i18n.go`.
- `core.Engine` is a high-coupling area. Be careful with changes that affect sessions, permissions, streaming, relay, or command handling.
- Existing worktrees may be dirty. Avoid reverting unrelated user changes.
