# Refactoring Review

## Changes Reviewed
- `README.md` — expanded docs (features, usage, requirements)
- `go.mod` — added cobra dependency
- `main.go` — replaced placeholder with `cmd.Execute()`
- `main_test.go` — updated test name/comment
- `cmd/` package — root + 8 stub subcommands
- `internal/version/` — build-time version vars

## Refactoring Applied

### Fixed `notImplemented` to return errors instead of calling `os.Exit`
**File:** `cmd/root.go`

The `notImplemented` helper was assigned to `RunE` (error-returning) but called `os.Exit(1)` directly, bypassing cobra's error handling. The `return nil` after `os.Exit` was dead code. Replaced with `fmt.Errorf` to let cobra handle the error display and exit properly.

## No Other Issues Found
- Subcommand files follow a consistent, clean pattern
- No dead code or unused imports
- Shared flags (`--app`, `--window`) across commands are not worth extracting yet — descriptions vary slightly and commands are still stubs
