# Repo Index Freshness Check Execution Plan

## Objective

Make `make check-harness` fail when `docs/generated/repo-index.json` no longer matches the output of the generator script.

## Affected Modules

- `scripts/generate_repo_index.py`
- `scripts/test_generate_repo_index.py`
- `scripts/check_harness.sh`

## Approach

Extend the generator with a `--check` mode so generation and validation use identical logic. Reuse that path from `check_harness.sh` to keep repository validation narrow and deterministic.

## Validation Strategy

- run `python3 scripts/test_generate_repo_index.py`
- run `make check-harness`
- run `go test ./...`
- run `make build`

## Rollback Plan

- delete the `--check` branch from `scripts/generate_repo_index.py`
- revert the extra checks in `scripts/test_generate_repo_index.py`
- remove the repo-index validation call from `scripts/check_harness.sh`

## Steps

1. Add `--check` mode to `scripts/generate_repo_index.py`.
2. Expand `scripts/test_generate_repo_index.py` to cover stale-file detection.
3. Update `scripts/check_harness.sh` to call the generator in check mode.
4. Run validation commands.
