# Task Command Design

Date: 2026-03-11

## Goal

Add a built-in `/task` command that turns a user request into an explicit engineering task for the currently active AI session.

## User Contract

Users can send:

```text
/task 修复登录问题
```

The command must:

- work without specifying bot or project
- target the current active agent session in the current project
- tell the AI to follow repository rules before doing work
- tell the AI to complete the request with minimal questioning

## Scope

Initial scope is intentionally narrow:

- one built-in command: `/task <requirement>`
- no bot mention parsing
- no cross-project routing
- no extra workflow state beyond the existing session model

## Behavioral Design

`/task` is handled by `core.Engine` as a first-class built-in command.

When invoked, the engine will:

1. validate that a non-empty task body exists
2. build a normalized task prompt
3. send that prompt into the existing interactive agent flow

The normalized prompt will explicitly encode the execution policy so the AI does not need to infer it from `/task` alone.

## Prompt Shape

The injected prompt should preserve the user's original requirement while prepending a short execution contract:

- follow project conventions and repository workflow
- finish the requested work
- avoid unnecessary questions

This keeps `/task` stable across agents while reusing the current message-processing pipeline.

## Error Handling

If the user sends `/task` with no body, the engine replies with usage text instead of forwarding an empty task to the agent.

Normal session locking, permission handling, banned-word checks, and agent event streaming remain unchanged because `/task` reuses the existing interactive execution path.

## User-Facing Surface

The command must appear in:

- built-in command matching
- `/help`
- platform-native command registration via `GetAllCommands`
- localized descriptions in `core/i18n.go`

## Testing

Add focused engine tests that verify:

- `/task <text>` sends a normalized prompt to the agent session
- `/task` without arguments returns usage text and does not send to the agent
- `/task` is available as a built-in command for command registration/help wiring
