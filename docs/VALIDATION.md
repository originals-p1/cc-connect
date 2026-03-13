# Validation Guide

This document defines how to verify changes in this repository.

For repository workflow context, also see [`docs/harness/quality-gates.md`](harness/quality-gates.md).

## Standard Validation

Preferred full validation:

```bash
make check-harness
go test ./...
make build
```

Use the full set when the change touches shared flow, runtime wiring, config shape, or user-visible behavior.

## Test Categories

### 1. Unit Tests

Package-specific tests for isolated functionality:

```bash
go test ./agent/claudecode/... -v
go test ./platform/telegram/... -v
go test ./config/... -v
go test ./core/... -v
go test ./daemon/... -v
```

### 2. Integration Tests

Cross-package tests for interactions:

```bash
# Run all tests
go test ./...

# With coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out
```

### 3. Linting

Code quality checks:

```bash
make lint
# or
golangci-lint run ./...
```

### 4. Build Verification

Ensure the project builds correctly:

```bash
make build
# or build specific platform
make release TARGET=linux/amd64
```

### 5. Repository Governance

```bash
make check-harness
```

## Narrow Changes

If a change is localized, start with targeted checks first, then escalate to the full set if the change can affect shared behavior.

Typical examples:

- `go test ./core/...` for `core/` changes
- `go test ./config/...` for config changes
- `go test ./cmd/cc-connect/...` for CLI changes

## Test Coverage Targets

| Package | Unit Test Coverage | Integration Test Coverage |
|---------|-------------------|--------------------------|
| `core/` | ≥ 80% | ≥ 60% |
| `agent/*/` | ≥ 70% | ≥ 50% |
| `platform/*/` | ≥ 70% | ≥ 50% |
| `config/` | ≥ 85% | N/A |

## Documentation and Sync Checks

When behavior changes, review whether these need updates:

- `config.example.toml`
- `README.md`
- platform- or feature-specific docs in `docs/`
- `docs/AI_CONTEXT.md` when structural knowledge changes materially
- `core/i18n.go` for user-visible command/help/message changes
- `docs/VALIDATION.md` when validation process changes
- `docs/AGENT_INTEGRATION.md` for new agent features
- `docs/PLATFORM_INTEGRATION.md` for new platform features

## Agent Integration Validation

When adding or modifying an agent:

```bash
# 1. Agent tests
go test ./agent/<name>/... -v

# 2. Integration with core
go test ./core/... -v

# 3. Full project build
make build

# 4. Registry check
make check-harness
```

## Platform Integration Validation

When adding or modifying a platform:

```bash
# 1. Platform tests
go test ./platform/<name>/... -v

# 2. Integration tests
go test ./core/... -v

# 3. Build verification
make build

# 4. Registry check
make check-harness
```

## Performance Validation

For performance-sensitive modules:

```bash
# Profile CPU usage
go test -bench=. -benchmem ./core/...
go test -cpuprofile=cpu.out ./core/...
go tool pprof cpu.out

# Test memory usage
go test -memprofile=mem.out ./core/...
go tool pprof -text mem.out
```

## Harness Check

Repository governance check:

```bash
make check-harness
```

Current first-pass coverage:

- each package under `agent/` is blank-imported in `cmd/cc-connect/main.go`
- each package under `platform/` is blank-imported in `cmd/cc-connect/main.go`

## Automated CI Checks

The following checks run on CI:

- [x] Go version compatibility (1.22+)
- [x] Linting (golangci-lint)
- [x] Unit tests
- [x] Integration tests
- [x] Build on Linux/macOS/Windows
- [x] Code coverage ≥ 70%
- [x] No security vulnerabilities (govulncheck)
- [x] Repository governance (check-harness)

## Completion Rule

Do not claim a task is complete without fresh verification evidence from the relevant commands.
