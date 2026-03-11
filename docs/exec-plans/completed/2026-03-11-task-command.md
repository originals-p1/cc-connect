# Task Command Execution Plan

## Objective

Add a built-in `/task` command that wraps a user request in repository-execution guidance and sends it to the current active AI session.

## Affected Modules

- `core/engine.go`
- `core/engine_test.go`
- `core/i18n.go`

## Approach

Extend builtin command parsing with a new `task` command. Implement a small helper that converts `/task <requirement>` into a normalized prompt emphasizing project rules, completing the requirement, and avoiding unnecessary questions. Reuse the existing interactive message path so session lifecycle, permissions, and streaming behavior stay unchanged.

## Validation Strategy

- run focused `/task` tests in `core`
- run broader `go test ./core`
- run `go test ./...`

## Rollback Plan

- remove `/task` from built-in command registration
- delete the task prompt helper and command handler
- remove `/task` help/i18n entries and tests

## Steps

1. Add failing tests for `/task` success and usage handling in `core/engine_test.go`.
2. Run the focused tests and confirm they fail for the expected reason.
3. Implement `/task` command parsing and prompt construction in `core/engine.go`.
4. Add `/task` localized help text and command description in `core/i18n.go`.
5. Re-run focused tests, then broader validation commands.
