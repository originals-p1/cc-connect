# Repo Index Freshness Check Design

## Objective

Enforce that `docs/generated/repo-index.json` stays in sync with the Python generator by making repository checks fail when the file is stale.

## Affected Modules

- `scripts/generate_repo_index.py`
- `scripts/test_generate_repo_index.py`
- `scripts/check_harness.sh`

## Approach

Add a `--check` mode to the generator script.

The generator will:

- build the same JSON payload used for file generation
- compare the expected serialized output against the current output file
- exit `0` when the file is current
- exit non-zero when the file is missing or stale
- print a clear remediation message instructing the user to run `make generate-repo-index`

Then wire that check into `scripts/check_harness.sh`.

## Validation Strategy

- run the Python test script and verify that generation and check mode both work
- run `make check-harness`
- run `go test ./...`
- run `make build`

## Rollback Plan

- remove `--check` support from the generator
- remove the extra assertions from the Python test script
- remove the generator freshness step from `check_harness.sh`
