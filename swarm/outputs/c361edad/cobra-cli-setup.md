# Cobra CLI Setup - Completed

## What was done
- Added `github.com/spf13/cobra@v1.10.2` dependency
- Created `cmd/root.go` with root command, version flag, and `notImplemented()` helper
- Created 8 command stubs with all PLAN.md flags:
  - `cmd/read.go` - 10 flags (app, window, window-id, pid, depth, roles, visible-only, bbox, compact, pretty)
  - `cmd/list.go` - 4 flags (apps, windows, pid, app)
  - `cmd/click.go` - 7 flags (id, x, y, button, double, app, window)
  - `cmd/typecmd.go` - 6 flags + positional arg (text, key, delay, id, app, window)
  - `cmd/focus.go` - 4 flags (app, window, window-id, pid)
  - `cmd/scroll.go` - 7 flags (direction, amount, x, y, id, app, window)
  - `cmd/drag.go` - 8 flags (from-x, from-y, to-x, to-y, from-id, to-id, app, window)
  - `cmd/screenshot.go` - 6 flags (window, app, output, format, quality, scale)
- Updated `main.go` to call `cmd.Execute()`
- Added `cmd/root_test.go` with tests for subcommand registration and version
- All stubs output `{"error":"command not yet implemented","command":"<name>"}` to stderr with exit code 1

## Verification
- `go build ./...` passes
- `go test ./...` passes
- `go vet ./...` passes
- All 8 commands appear in `--help`
- All flags match PLAN.md spec
- `--version` outputs version info
