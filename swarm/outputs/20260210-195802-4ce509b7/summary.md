# Bug Fix: `do --stop-on-error=false` incorrectly reports success when steps fail

## Agent: c670d24a | Task: c8457b4f

## Problem

When running `desktop-cli do --stop-on-error=false` and one or more steps fail:
- `ok` was incorrectly `true` (should be `false`)
- `completed` counted all steps including failed ones (should only count successful ones)

## Root Cause

In `cmd/do.go`, the `lastErr` variable was only set inside the `if stopOnError` branch. When `stopOnError=false`, `lastErr` remained empty. Then:
- `if lastErr == ""` → `completed = len(results)` overwrote the correct count with total count
- `allOK := lastErr == ""` was always `true`

## Fix

Added a `hasFailure` boolean that tracks whether any step failed, independently of `lastErr`:

1. `hasFailure` is set to `true` whenever a step fails (both in the `stopOnError` and `!stopOnError` paths)
2. `completed` is only overwritten with `len(results)` when `!hasFailure` (was `lastErr == ""`)
3. `allOK` is computed as `!hasFailure` (was `lastErr == ""`)

## Files Changed

- `cmd/do.go` — Fixed the batch result computation logic
- `cmd/do_test.go` — Added 5 unit tests covering all scenarios

## Tests Added

- `TestDoResult_AllSuccess` — All steps succeed → `ok=true, completed=3`
- `TestDoResult_StopOnError_FailAtStep2` — Fail at step 2 with stop-on-error=true → `ok=false, completed=1`
- `TestDoResult_ContinueOnError_FailAtStep2` — The exact bug scenario: 3 steps, step 2 fails, stop-on-error=false → `ok=false, completed=2`
- `TestDoResult_ContinueOnError_AllFail` — All steps fail → `ok=false, completed=0`
- `TestDoResult_ContinueOnError_MultipleFails` — 5 steps, 2 fail → `ok=false, completed=3`

## Validation

- `go build ./...` — passes
- `go test ./...` — all tests pass (no regressions)
