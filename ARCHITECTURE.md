# ARCHITECTURE.md

This is the architecture landing page for coding agents and contributors.

For deeper structural detail, read [`docs/AI_CONTEXT.md`](docs/AI_CONTEXT.md).

## System Purpose

`cc-connect` bridges local AI coding agents to chat platforms so users can operate those agents remotely through messaging apps.

## Core Flow

1. A platform adapter receives a message.
2. `core.Engine` parses commands or routes the message to an agent session.
3. The agent session emits events.
4. `core.Engine` translates those events into replies and sends them back through the platform adapter.

## Main Areas

- `cmd/cc-connect/`: CLI entrypoint, daemon commands, send/relay/provider subcommands
- `core/`: engine, sessions, routing, commands, skills, events, speech, relay, cron
- `config/`: TOML schema, validation, persistence helpers
- `agent/`: agent integrations such as Claude Code, Codex, Cursor, Gemini, Qoder, OpenCode, iFlow
- `platform/`: chat platform integrations such as Feishu, DingTalk, Telegram, Slack, Discord, QQ, WeCom
- `daemon/`: service and daemon helpers
- `docs/`: setup guides, architecture context, workflows, plans

## Key Boundaries

- New agents and platforms must register themselves and be wired into `cmd/cc-connect/main.go`.
- Config changes must be reflected in schema, examples, and user-facing docs.
- User-visible command or help changes should be reviewed together with `core/i18n.go`.
- `core.Engine` is a high-coupling area; changes there can affect sessions, permissions, streaming, relay, and command handling.

## Recommended Reading Order

1. [`AGENTS.md`](AGENTS.md)
2. [`docs/VALIDATION.md`](docs/VALIDATION.md)
3. [`docs/TASK_RECIPES.md`](docs/TASK_RECIPES.md)
4. [`docs/generated/repo-index.json`](docs/generated/repo-index.json)
5. [`docs/AI_CONTEXT.md`](docs/AI_CONTEXT.md)
