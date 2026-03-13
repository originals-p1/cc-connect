# Planner Dispatch Parse Repair Execution Plan

## Objective

Allow planner-driven `/task` dispatch to retry once when a planner response
contains malformed extracted JSON and fails `json.Unmarshal`, instead of only
retrying when no JSON is found at all.

## Affected Modules

- `core/engine.go`
- `core/multi_agent_test.go`

## Approach

1. Keep the current extraction and validation flow intact.
2. Broaden the repair-retry trigger to include malformed JSON parse failures.
3. Add a regression test that reproduces the `invalid character 'p' looking for beginning of value` case.

## Validation Strategy

- `go test ./core/...`

## Rollback Plan

- Revert the retry condition change in `core/engine.go`
- Remove the malformed JSON regression test
