# Smart Context Detection — Implementation Summary

## Task: smart-context-defaults
## Agent: 9573f833
## Status: Completed

## What Was Implemented

Smart context detection that automatically applies optimal defaults based on the runtime context, eliminating common agent mistakes and reducing flag verbosity.

### Features Implemented

1. **Auto agent format for piped output** — When stdout is piped (agent context), `--format agent` is automatically used instead of `--format yaml`. This saves 20-30x tokens on every read call.

2. **Auto-prune for web content** — When the element tree contains a "web" role (AXWebArea), `--prune` is automatically enabled. This removes empty group/other elements that clutter web page trees, reducing output 5-8x.

3. **Auto role expansion for web content** — When `--roles "input"` is specified and web content is detected, "other" is automatically added to the role list. Chrome exposes some web input fields as role "other" instead of "input".

4. **`--raw` flag** — Disables all smart defaults for explicit control.

5. **`smart_defaults` response field** — Shows which smart defaults were applied, providing transparency so agents understand the output context.

### Files Created
- `internal/model/profiles.go` — `HasWebContent()` and `ExpandRolesForWeb()` functions
- `internal/model/profiles_test.go` — Tests for web detection and role expansion

### Files Modified
- `cmd/root.go` — Added `--raw` flag, auto-format detection based on piped output, default format changed from "yaml" to "" (auto-detect)
- `cmd/read.go` — Integrated smart defaults: auto-prune on web content, auto role expansion, smart_defaults tracking
- `internal/output/output.go` — Added `RawMode`, `IsOutputPiped()`, `SmartDefaults` field to ReadResult/ReadFlatResult
- `README.md` — Documented smart defaults behavior
- `SKILL.md` — Updated read examples to reflect simpler usage with smart defaults

### Validation
- `go build ./...` — passes
- `go test ./...` — all tests pass (including new profiles_test.go)
- Backwards compatible: explicit flags always override smart defaults
