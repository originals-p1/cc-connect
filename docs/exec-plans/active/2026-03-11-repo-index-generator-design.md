# Repo Index Generator Design

## Objective

Replace the hand-written `docs/generated/repo-index.json` with a generated file produced from stable repository structure.

## Affected Modules

- `scripts/`
- `docs/generated/`
- `Makefile`

## Approach

Add a small Python script that scans a limited, stable set of repository facts and emits `docs/generated/repo-index.json`.

The script will:

- use the repository root as the default scan target
- enumerate top-level areas from a curated list of known directories
- emit stable document entrypoints referenced by `AGENTS.md`
- emit key file metadata for the main entrypoint and core files
- keep the JSON schema compatible with the current file shape

It will not:

- parse imports or build a dependency graph
- infer architecture rules from source
- scan arbitrary deep subtrees

## Validation Strategy

- run the generator script directly
- validate the output JSON parses successfully
- run `make check-harness`
- run `go test ./...`
- run `make build`

## Rollback Plan

- remove the generator script
- remove the `Makefile` target
- restore the previous hand-maintained `docs/generated/repo-index.json`
