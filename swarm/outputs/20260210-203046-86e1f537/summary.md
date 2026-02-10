# Assert Command Implementation Summary

## Task: `assert` Command and Batch Step

### Files Created
- **`cmd/assert.go`** — New `assert` command with full property assertion support
- **`cmd/assert_test.go`** — 12 test cases covering all assertion types

### Files Modified
- **`cmd/do.go`** — Added `executeAssert()` function and `"assert"` to step dispatcher
- **`README.md`** — Added assert command documentation with examples and response format
- **`SKILL.md`** — Added assert quick reference, batch step example, and agent workflow entry

### Implementation Details

The `assert` command provides structured pass/fail validation of UI state:

**Element targeting:**
- `--text` with `--roles`, `--exact`, `--scope-id` (same as click/action)
- `--id` for direct element targeting
- `--app`, `--window`, `--pid`, `--window-id` for scoping

**Property assertions:**
- `--value` — exact value match
- `--value-contains` — case-insensitive substring match on value
- `--checked` / `--unchecked` — selected/checked state
- `--disabled` / `--enabled` — enabled state
- `--is-focused` — keyboard focus
- `--gone` — element does NOT exist

**Timing:**
- `--timeout` — poll until condition met or timeout (0 = single check)
- `--interval` — polling interval in ms (default 500)

**Exit behavior:**
- Exit code 0 on pass, 1 on fail (enables `&&` chaining)
- Structured YAML output with `pass: true/false`, element info, and error messages

**Batch integration:**
- Works as a `do` step: `- assert: { text: "Success", timeout: 5 }`
- Supports all assertion flags as step params

### Test Results
- `go build ./...` — passes
- `go test ./...` — all tests pass (12 new assert tests + all existing tests)
