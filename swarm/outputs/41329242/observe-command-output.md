# Observe Command Implementation - Complete

## Files Created
- `internal/model/diff.go` — `UIChange` struct, `DiffElements()`, `diffProperties()` functions
- `internal/model/diff_test.go` — 11 unit tests covering: no changes, added, removed, changed, empty inputs, all-new, all-removed, bounds/selected/description changes, no-diff
- `cmd/observe.go` — New `observe` command with JSONL streaming output

## Files Modified
- `README.md` — Added "Observe UI changes" section with usage examples
- `SKILL.md` — Added `observe` to quick reference and agent workflow

## Validation
- `go build ./...` — passes
- `go test ./...` — all tests pass
- `desktop-cli observe --help` — shows all flags correctly

## Command Features
- Streams UI diffs as JSONL (one JSON object per line)
- Events: snapshot, added, removed, changed, error, done
- Flags: --app, --window, --window-id, --pid, --depth, --roles, --interval, --duration, --ignore-bounds, --ignore-focus
- Error resilience: transient read failures emit error events without stopping
- Always JSONL output regardless of --format flag
