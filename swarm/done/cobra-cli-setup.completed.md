# Feature: Cobra CLI Framework Setup with All Command Stubs

## Priority: CRITICAL (Phase 1, Step 1 — blocks all other work)

## Problem

The codebase is a bare scaffold. `main.go` just prints "desktop-cli" and there are zero dependencies. Every planned command (read, list, click, type, focus, scroll, drag, screenshot) needs the Cobra CLI framework in place before any can be implemented.

## What to Build

### 1. Add Cobra dependency

```bash
go get github.com/spf13/cobra@latest
```

### 2. Create `cmd/root.go`

- Root command `desktop-cli` with long description
- `--version` flag wired to `internal/version/version.go`
- Global flags: none yet (placeholder for future `--format` etc.)

### 3. Create command stubs with all planned flags

Each command file should register the command with Cobra, define all flags from PLAN.md, and print a "not yet implemented" JSON error to stderr with exit code 1. This lets agents discover the full CLI surface immediately.

Create these files:

| File | Command | Key Flags |
|------|---------|-----------|
| `cmd/read.go` | `read` | `--app`, `--window`, `--window-id`, `--pid`, `--depth`, `--roles`, `--visible-only`, `--bbox`, `--compact`, `--pretty` |
| `cmd/list.go` | `list` | `--apps`, `--windows`, `--pid`, `--app` |
| `cmd/click.go` | `click` | `--id`, `--x`, `--y`, `--button`, `--double`, `--app`, `--window` |
| `cmd/typecmd.go` | `type` | `--text`, `--key`, `--delay`, `--id`, `--app`, `--window` |
| `cmd/focus.go` | `focus` | `--app`, `--window`, `--window-id`, `--pid` |
| `cmd/scroll.go` | `scroll` | `--direction`, `--amount`, `--x`, `--y`, `--id`, `--app`, `--window` |
| `cmd/drag.go` | `drag` | `--from-x`, `--from-y`, `--to-x`, `--to-y`, `--from-id`, `--to-id`, `--app`, `--window` |
| `cmd/screenshot.go` | `screenshot` | `--window`, `--app`, `--output`, `--format`, `--quality`, `--scale` |

### 4. Update `main.go`

Replace the `fmt.Println` with a call to `cmd.Execute()`.

### 5. Stub error output format

Each stub command's `RunE` should output:
```json
{"error": "command not yet implemented", "command": "<name>"}
```
and return exit code 1. This gives agents a machine-readable signal.

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `desktop-cli --help` shows all 8 subcommands
- [ ] `desktop-cli read --help` shows all flags from PLAN.md
- [ ] `desktop-cli list --help` shows all flags from PLAN.md
- [ ] `desktop-cli click --help` shows all flags from PLAN.md
- [ ] `desktop-cli type --help` shows all flags from PLAN.md
- [ ] `desktop-cli focus --help` shows all flags from PLAN.md
- [ ] `desktop-cli scroll --help` shows all flags from PLAN.md
- [ ] `desktop-cli drag --help` shows all flags from PLAN.md
- [ ] `desktop-cli screenshot --help` shows all flags from PLAN.md
- [ ] Running any command outputs a JSON error and exits with code 1
- [ ] `desktop-cli --version` prints version info
- [ ] README.md updated with any relevant changes
- [ ] SKILL.md updated with any relevant changes

## Files to Create/Modify

- `go.mod` — add cobra dependency
- `main.go` — wire to cmd.Execute()
- `cmd/root.go` — root command + version
- `cmd/read.go` — read command stub
- `cmd/list.go` — list command stub
- `cmd/click.go` — click command stub
- `cmd/typecmd.go` — type command stub (avoid Go keyword)
- `cmd/focus.go` — focus command stub
- `cmd/scroll.go` — scroll command stub
- `cmd/drag.go` — drag command stub
- `cmd/screenshot.go` — screenshot command stub

## Notes

- Use `typecmd.go` filename to avoid collision with Go's `type` keyword
- All flag names and types must match PLAN.md exactly so downstream features can assume consistent flag interfaces
- The `type` command should accept text as a positional argument too (in addition to `--text`)

## Completion Notes (Agent 8ab98af1)

All acceptance criteria met:
- Added `github.com/spf13/cobra@v1.10.2` dependency
- Created `cmd/root.go` with root command, version flag wired to `internal/version/version.go`, and `notImplemented()` helper
- Created all 8 command stubs (`cmd/read.go`, `cmd/list.go`, `cmd/click.go`, `cmd/typecmd.go`, `cmd/focus.go`, `cmd/scroll.go`, `cmd/drag.go`, `cmd/screenshot.go`) with all flags matching PLAN.md
- Updated `main.go` to call `cmd.Execute()`
- `type` command accepts text as positional argument via `cobra.MaximumNArgs(1)`
- Each stub outputs JSON error to stderr and exits with code 1
- `go build ./...` succeeds, `go test ./...` passes (including new `cmd/root_test.go`)
- `desktop-cli --version` prints version info
- README.md and SKILL.md already document the full CLI surface — no changes needed
