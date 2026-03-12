# OpenCode Owned Server Registry Execution Plan

## Objective

Ensure `cc-connect` can shut down OpenCode resident servers that it launched itself, even if a session object was leaked or a higher-level path forgot to close it.

## Affected Modules

- `agent/opencode/opencode.go`
- `agent/opencode/session.go`
- `agent/opencode/opencode_test.go`

## Approach

Add a process-local registry on the OpenCode agent that tracks resident-server-backed sessions created by that agent instance. Register sessions on startup, unregister them on close, and make `Agent.Stop()` close any still-owned sessions so shutdown only touches processes launched by the current `cc-connect` process.

## Validation Strategy

- run `go test ./agent/opencode/...`
- run `go test ./core/...`
- run `make build`

## Rollback Plan

- remove the owned-session registry from `agent/opencode/opencode.go`
- remove session registration hooks from `agent/opencode/session.go`
- remove the shutdown regression coverage from `agent/opencode/opencode_test.go`

## Steps

1. Add owned-session tracking to the OpenCode agent.
2. Register/unregister resident-server sessions as they are created and closed.
3. Make `Agent.Stop()` close any still-owned sessions.
4. Add regression tests and run targeted validation.

## Result

- The OpenCode agent now tracks only the resident-server-backed sessions created by the current `cc-connect` process.
- Session shutdown unregisters itself from the owning agent, and `Agent.Stop()` closes any still-tracked sessions during engine shutdown.
- This cleanup path only touches servers launched by this process, avoiding accidental termination of unrelated manual or external OpenCode instances.
- Validation completed with `go test ./agent/opencode/...`, `go test ./core/...`, and `make build`.
