# Task: do-conditional-steps

## Summary

Implemented conditional step types (`if-exists`, `if-focused`, `try`) in the `do` batch command, enabling branching and error handling within YAML batch sequences without requiring additional LLM round-trips.

## Changes

### `cmd/do.go`
- Changed YAML parsing from `[]map[string]map[string]interface{}` to `[]map[string]interface{}` to support both regular and conditional step structures
- Added `doContext` struct to manage execution state across recursive calls
- Added `executeSteps` method for recursive substep execution
- Added `parseRegularStep` to extract action/params from non-conditional steps
- Added `parseSubsteps` to convert YAML arrays to step maps
- Added `executeIfExists` — checks element presence via `resolveElementByText`, runs `then`/`else` branch
- Added `executeIfFocused` — checks focused element via `readFocusedElement`, runs `then`/`else` branch
- Added `executeTry` — runs substeps and always reports OK, absorbing errors
- Added `Matched`, `Branch`, `Substeps` fields to `StepResult` for conditional step reporting
- Updated command description and supported step types list

### `cmd/do_test.go`
- Added comprehensive tests: YAML parsing for all conditional types, backward compatibility, `doContext` execution with sleep, try error absorption, if-exists/if-focused fallback to else with no provider, mixed steps, nested conditionals

### `README.md`
- Updated supported step types list to include `if-exists`, `if-focused`, `try`
- Added "Conditional steps" subsection with examples for all three types

### `SKILL.md`
- Updated step types list and added conditional step examples
- Updated agent workflow section to mention conditional steps

## Testing

- All existing tests pass (backward compatible)
- All new tests pass
- Build succeeds with `go build ./...`
- Full test suite: `go test ./...` passes
