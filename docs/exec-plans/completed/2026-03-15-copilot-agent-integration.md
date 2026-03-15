## Objective

Add GitHub Copilot CLI support to `cc-connect` as a new agent integration, using the most stable machine-integrable interface available.

## Affected Modules

- `agent/copilot/`
- `cmd/cc-connect/main.go`
- `config.example.toml`
- `docs/AI_CONTEXT.md`
- `docs/AGENT_INTEGRATION.md`

## Approach

Implement a new `copilot` agent package that registers via `core.RegisterAgent`.
Prefer the Copilot CLI ACP server mode over terminal scraping so the integration can use structured request/response events and support persistent sessions with minimal coupling to the human-facing TUI.
Mirror the existing agent integration conventions for model/mode configuration, environment propagation, and registration.

## Validation Strategy

- `go test ./agent/copilot/...`
- `go test ./...`
- `make build`
- `make check-harness`

## Rollback Plan

Revert the `agent/copilot` package and the related registration/documentation changes if ACP integration proves incompatible or breaks shared agent wiring.
