# Project Switching MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a DM-only project-switching MVP so one bot can scan a single workspace root, list git projects, and switch its active `work_dir` per user without changing the bot's fixed agent/platform configuration.

**Architecture:** Keep bot configuration fixed and treat project switching as runtime routing. Add a workspace catalog, a per-user active-project binding store keyed by `platform + user_id + bot_id`, and a bot-scoped runtime manager that lazily creates per-project engines/sessions as users switch projects.

**Tech Stack:** Go, TOML config, existing `core.Engine`/`SessionManager`, existing platform integrations, Go test

---

### Task 1: Add config schema for a single workspace root and bot list

**Files:**
- Modify: `config/config.go`
- Modify: `cmd/cc-connect/main.go`
- Modify: `config.example.toml`
- Modify: `README.md`
- Test: `config/config_test.go`

**Step 1: Write the failing config tests**

Add tests that load TOML with:

- `[workspace] root = "/tmp/code" require_git = true`
- `[[bots]]` entries with `name`, `agent_type`, `default_project`, `max_cached_sessions`, `dm_only`
- nested `[bots.agent_options]`
- nested `[[bots.platforms]]` and `[bots.platforms.options]`

Verify:

- new fields unmarshal correctly
- missing `workspace.root` fails when `bots` mode is used
- missing `bots[].agent_type` or empty platforms fails

**Step 2: Run test to verify it fails**

Run: `go test ./config -run 'TestLoad.*Workspace|TestLoad.*Bots|TestValidate.*Bots' -v`

Expected: FAIL because the new config structs and validation do not exist yet.

**Step 3: Write minimal config implementation**

In `config/config.go`:

- add `WorkspaceConfig` with `Root string` and `RequireGit *bool`
- add `BotConfig` with:
  - `Name string`
  - `AgentType string`
  - `AgentOptions map[string]any`
  - `Providers []ProviderConfig`
  - `Platforms []PlatformConfig`
  - `DefaultProject string`
  - `MaxCachedSessions int`
  - `DMOnly *bool`
- extend `Config` to support `Workspace WorkspaceConfig` and `Bots []BotConfig`
- update validation to allow either legacy `[[projects]]` mode or new `[[bots]]` mode
- keep `Projects` behavior intact for backward compatibility

**Step 4: Run test to verify it passes**

Run: `go test ./config -run 'TestLoad.*Workspace|TestLoad.*Bots|TestValidate.*Bots' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add config/config.go config/config_test.go config.example.toml README.md cmd/cc-connect/main.go
git commit -m "feat: add workspace and bot config schema"
```

### Task 2: Add a workspace scanner and project catalog

**Files:**
- Create: `core/project_catalog.go`
- Test: `core/project_catalog_test.go`

**Step 1: Write the failing tests**

Add tests for:

- scanning one root returns only direct child directories with `.git`
- non-git directories are excluded when `require_git` is true
- project names are derived deterministically from directory name
- duplicate names are rejected with a clear error
- missing root returns an error

**Step 2: Run test to verify it fails**

Run: `go test ./core -run 'TestProjectCatalog|TestScanWorkspace' -v`

Expected: FAIL because no workspace catalog exists yet.

**Step 3: Write minimal implementation**

Create `core/project_catalog.go` with:

- `type ProjectInfo struct { Name, Path, GitRoot string }`
- `type ProjectCatalog struct { Root string; Projects map[string]ProjectInfo }`
- `func ScanWorkspace(root string, requireGit bool) (*ProjectCatalog, error)`
- direct-child scanning only
- deterministic name derivation from base directory name

**Step 4: Run test to verify it passes**

Run: `go test ./core -run 'TestProjectCatalog|TestScanWorkspace' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add core/project_catalog.go core/project_catalog_test.go
git commit -m "feat: add workspace project catalog"
```

### Task 3: Add DM-scoped active-project binding persistence

**Files:**
- Create: `core/project_binding.go`
- Test: `core/project_binding_test.go`

**Step 1: Write the failing tests**

Add tests for:

- set/get active project by `platform + user_id + bot_id`
- save and reload bindings from disk
- switching active project overwrites previous binding
- default empty binding when none exists

**Step 2: Run test to verify it fails**

Run: `go test ./core -run 'TestProjectBinding' -v`

Expected: FAIL because no binding store exists yet.

**Step 3: Write minimal implementation**

Create `core/project_binding.go` with:

- `type BindingKey struct { Platform, UserID, BotID string }`
- `type BindingRecord struct { ActiveProject string; SwitchedAt time.Time }`
- `type BindingStore struct { ... }`
- JSON persistence using the existing atomic write pattern
- lookup/set/save/load methods

**Step 4: Run test to verify it passes**

Run: `go test ./core -run 'TestProjectBinding' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add core/project_binding.go core/project_binding_test.go
git commit -m "feat: add active project binding store"
```

### Task 4: Introduce a bot runtime manager for lazy project engines

**Files:**
- Create: `core/bot_runtime.go`
- Modify: `core/interfaces.go`
- Test: `core/bot_runtime_test.go`

**Step 1: Write the failing tests**

Add tests for:

- first access to a project lazily creates a runtime
- repeated access reuses the same runtime
- LRU eviction closes least-recently-used project session/runtime when `max_cached_sessions` is exceeded
- bot identity is part of the runtime key

**Step 2: Run test to verify it fails**

Run: `go test ./core -run 'TestBotRuntime' -v`

Expected: FAIL because there is no bot runtime manager yet.

**Step 3: Write minimal implementation**

Create `core/bot_runtime.go` with:

- `type BotSpec` describing fixed bot configuration needed at runtime
- `type ProjectRuntime` containing project name, work dir, engine/session handles, and timestamps
- `type BotRuntimeManager` that:
  - holds one `ProjectCatalog`
  - holds one `BindingStore`
  - lazily creates project runtimes
  - reuses active runtime for the same project
  - evicts least recently used runtime when above limit

Update `core/interfaces.go` only if a small new optional interface is needed for bot identity; otherwise keep platform interfaces unchanged.

**Step 4: Run test to verify it passes**

Run: `go test ./core -run 'TestBotRuntime' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add core/bot_runtime.go core/bot_runtime_test.go core/interfaces.go
git commit -m "feat: add lazy bot runtime manager"
```

### Task 5: Define bot identity and DM-only routing metadata in incoming messages

**Files:**
- Modify: `core/message.go`
- Modify: `platform/telegram/telegram.go`
- Modify: `platform/discord/discord.go`
- Modify: `platform/feishu/feishu.go`
- Modify: `platform/slack/slack.go`
- Modify: `platform/dingtalk/dingtalk.go`
- Modify: `platform/line/line.go`
- Modify: `platform/qq/qq.go`
- Modify: `platform/qqbot/qqbot.go`
- Modify: `platform/wecom/wecom.go`
- Test: platform-specific tests where present, plus `core` tests if helpers are added

**Step 1: Write the failing tests**

Add or extend tests to prove:

- DM/private-chat messages are marked as DM-capable for switching mode
- group messages are rejected or bypassed when `dm_only` is enabled
- each incoming message includes enough bot identity to key bindings deterministically

**Step 2: Run test to verify it fails**

Run: `go test ./platform/... ./core/...`

Expected: FAIL because message metadata is insufficient for bot-scoped binding/routing.

**Step 3: Write minimal implementation**

Update `core.Message` to carry the minimum routing metadata, for example:

- `ChatID string`
- `BotID string`
- `IsDM bool`

Update platform adapters so they populate these fields consistently for incoming messages.

**Step 4: Run test to verify it passes**

Run: `go test ./platform/... ./core/...`

Expected: PASS

**Step 5: Commit**

```bash
git add core/message.go platform/telegram/telegram.go platform/discord/discord.go platform/feishu/feishu.go platform/slack/slack.go platform/dingtalk/dingtalk.go platform/line/line.go platform/qq/qq.go platform/qqbot/qqbot.go platform/wecom/wecom.go
git commit -m "feat: add dm routing metadata for project switching"
```

### Task 6: Build bot-mode startup path in main and preserve legacy project mode

**Files:**
- Modify: `cmd/cc-connect/main.go`
- Modify: `core/api.go`
- Test: `cmd/cc-connect/main_test.go` or targeted tests in `core` if startup helpers are extracted

**Step 1: Write the failing tests**

Add tests for:

- legacy `[[projects]]` config still starts as before
- new `workspace + bots` config builds bot runtimes instead of fixed per-project engines
- API registration behavior remains correct for legacy mode

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/cc-connect ./core -run 'TestMain.*BotMode|TestMain.*LegacyProjectMode' -v`

Expected: FAIL because the startup path only supports legacy project mode.

**Step 3: Write minimal implementation**

Refactor `cmd/cc-connect/main.go`:

- extract legacy `buildEnginesFromProjects(...)`
- add `buildBotRuntimes(...)`
- for bot mode, create one platform handler per bot that routes through `BotRuntimeManager`
- keep daemon/API wiring working for legacy mode; explicitly disable or defer unsupported API features for bot mode if necessary

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/cc-connect ./core -run 'TestMain.*BotMode|TestMain.*LegacyProjectMode' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add cmd/cc-connect/main.go core/api.go cmd/cc-connect/main_test.go
git commit -m "feat: add bot-mode startup path"
```

### Task 7: Add `/project` commands for DM switching

**Files:**
- Modify: `core/engine.go`
- Modify: `core/i18n.go`
- Modify: `core/command.go` if command registration needs extension
- Test: `core/engine_test.go`

**Step 1: Write the failing tests**

Add tests for:

- `/project list` returns scanned projects
- `/project current` returns active project
- `/project switch <name>` binds the project and routes later messages correctly
- `/project switch --fresh <name>` creates a fresh session instead of reusing cached session
- DM-only enforcement returns a clear error in non-DM contexts

**Step 2: Run test to verify it fails**

Run: `go test ./core -run 'TestProjectCommand|TestProjectSwitch' -v`

Expected: FAIL because `/project` command family does not exist yet.

**Step 3: Write minimal implementation**

Extend `core.Engine` command handling with a `/project` command family that delegates to the bot runtime/binding layer:

- `list`
- `current`
- `switch`
- `switch --fresh`

Add i18n strings for success, missing project, DM-only, and empty catalog cases.

**Step 4: Run test to verify it passes**

Run: `go test ./core -run 'TestProjectCommand|TestProjectSwitch' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add core/engine.go core/i18n.go core/engine_test.go core/command.go
git commit -m "feat: add project switching commands"
```

### Task 8: Wire project-specific work_dir into agent session creation

**Files:**
- Modify: `agent/claudecode/claudecode.go`
- Modify: `agent/codex/codex.go`
- Modify: `agent/cursor/cursor.go`
- Modify: `agent/gemini/gemini.go`
- Modify: `agent/iflow/iflow.go`
- Modify: `agent/opencode/opencode.go`
- Modify: `agent/qoder/qoder.go`
- Test: targeted agent tests where present, plus new focused tests if session factory helpers are extracted

**Step 1: Write the failing tests**

Add tests proving:

- bot-fixed agent config can be cloned with a project-specific `work_dir`
- switching projects does not leak the previous project's `work_dir`
- provider/model settings remain fixed while `work_dir` changes

**Step 2: Run test to verify it fails**

Run: `go test ./agent/... -run 'Test.*WorkDir|Test.*BotConfigClone' -v`

Expected: FAIL because agents assume a single startup-time work dir.

**Step 3: Write minimal implementation**

Introduce a consistent helper pattern per agent so a runtime can create an agent instance or session using:

- fixed bot agent options
- project-specific `work_dir`

Prefer extracting small internal constructors rather than rewriting each agent package wholesale.

**Step 4: Run test to verify it passes**

Run: `go test ./agent/... -run 'Test.*WorkDir|Test.*BotConfigClone' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add agent/claudecode/claudecode.go agent/codex/codex.go agent/cursor/cursor.go agent/gemini/gemini.go agent/iflow/iflow.go agent/opencode/opencode.go agent/qoder/qoder.go
git commit -m "feat: support project-specific work dirs in bot mode"
```

### Task 9: Add documentation and example config for bot switching mode

**Files:**
- Modify: `config.example.toml`
- Modify: `README.md`
- Modify: `docs/AI_CONTEXT.md`
- Modify: `AGENTS.md`

**Step 1: Write the doc changes**

Document:

- `workspace` and `bots` config structure
- DM-only switching scope
- `.git`-based scanning rules
- `/project` commands
- limitations relative to legacy `[[projects]]` mode

**Step 2: Review for consistency**

Confirm config names, command names, and limitations match implementation exactly.

**Step 3: Commit**

```bash
git add config.example.toml README.md docs/AI_CONTEXT.md AGENTS.md
git commit -m "docs: document bot project switching mode"
```

### Task 10: Full verification and backward-compatibility pass

**Files:**
- Verify: `config/config.go`
- Verify: `cmd/cc-connect/main.go`
- Verify: `core/*.go`
- Verify: `agent/*.go`
- Verify: `platform/*.go`

**Step 1: Run focused package tests**

Run:

```bash
go test ./config ./core ./agent/... ./platform/... ./cmd/cc-connect -v
```

Expected: PASS

**Step 2: Run full test suite**

Run:

```bash
go test ./... -v
```

Expected: PASS

**Step 3: Run build verification**

Run:

```bash
make build
```

Expected: build completes successfully and produces `./cc-connect`

**Step 4: Manual smoke check**

Using a local config in bot mode:

- start the app
- DM the bot
- run `/project list`
- run `/project switch <name>`
- send a normal prompt
- run `/project current`
- run `/project switch --fresh <name>`

Expected:

- DM-only switching works
- active project changes correctly
- normal messages route to the selected project
- `--fresh` starts a fresh session

**Step 5: Commit**

```bash
git add .
git commit -m "test: verify project switching mvp"
```
