# Quality Gates

These are the repository rules that should remain true across changes.

The goal is to reduce reliance on memory and make the repository easier for coding agents to change safely.

## Minimum Validation

Preferred full validation:

```bash
go test ./...
make build
```

For narrow changes, start with targeted package tests, then run broader validation if shared flow may be affected.

## Structural Sync Rules

### Agent and Platform Wiring

When adding a new agent or platform:

- register it in the relevant package `init()`
- wire it into `core/registry.go` conventions
- blank-import it in `cmd/cc-connect/main.go`

This is a good candidate for mechanical checks because it is low-ambiguity and easy to forget.

### Config Sync

When config shape changes:

- update `config/config.go`
- update `config.example.toml`
- review `README.md`
- review affected docs in `docs/`

Do not change config shape in one place only.

### i18n Sync

When user-visible commands or messages change:

- review `core/i18n.go`
- confirm help text and usage examples still match behavior

### Documentation Link Health

When adding or moving docs:

- keep links and file references valid
- prefer linking to the canonical document instead of copying large sections

## Harness-Specific Check

Repository governance check:

```bash
make check-harness
```

First-pass scope:

- every package under `agent/` is blank-imported in `cmd/cc-connect/main.go`
- every package under `platform/` is blank-imported in `cmd/cc-connect/main.go`

Keep this check narrow at first. Only add rules that are precise enough to avoid false positives.
