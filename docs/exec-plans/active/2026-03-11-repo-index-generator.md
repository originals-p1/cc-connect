# Repo Index Generator Execution Plan

## Objective

Add a Python-based generator for `docs/generated/repo-index.json` and switch the repository to using generated structured metadata.

## Affected Modules

- `scripts/generate_repo_index.py`
- `docs/generated/repo-index.json`
- `Makefile`

## Approach

Implement a small Python script with explicit metadata tables for stable repository facts and limited filesystem scanning for existence checks. Generate the JSON file in one pass and expose the workflow through a Make target.

## Validation Strategy

- run the generator script and ensure it writes the output file
- parse the generated JSON with Python
- run `make check-harness`
- run `go test ./...`
- run `make build`

## Rollback Plan

- delete `scripts/generate_repo_index.py`
- remove the `generate-repo-index` target from `Makefile`
- restore the previous contents of `docs/generated/repo-index.json`

## Steps

1. Create `scripts/generate_repo_index.py`.
2. Add a `generate-repo-index` target to `Makefile`.
3. Regenerate `docs/generated/repo-index.json` using the script.
4. Run validation commands.
