# Clipboard Read/Write Command — Implementation Summary

## Task
Implemented the `clipboard` command with `read`, `write`, `clear`, and `grab` subcommands for desktop-cli.

## Changes Made

### New Files
- **`internal/platform/darwin/clipboard.go`** — macOS clipboard implementation using `pbcopy`/`pbpaste`
- **`internal/platform/darwin/clipboard_test.go`** — 5 tests: round-trip, Unicode, whitespace, clear, empty string
- **`cmd/clipboard.go`** — Cobra command with 4 subcommands: `read`, `write`, `clear`, `grab`

### Modified Files
- **`internal/platform/platform.go`** — Added `ClipboardManager` interface (GetText, SetText, Clear)
- **`internal/platform/provider.go`** — Added `ClipboardManager` field to `Provider` struct
- **`internal/platform/darwin/init.go`** — Wire up `NewClipboard()` in provider factory
- **`README.md`** — Added clipboard section to features list and usage docs
- **`SKILL.md`** — Added clipboard quick reference, updated agent workflow and contenteditable workarounds

## Verification
- `go build ./...` — passes
- `go test ./internal/platform/darwin/ -run TestClipboard` — all 5 tests pass
- Pre-existing test failures in `cmd/helpers_test.go` (FindNearestInteractiveElement) are unrelated

## Command Reference
```bash
desktop-cli clipboard read                          # read clipboard text
desktop-cli clipboard write "text"                  # write to clipboard
desktop-cli clipboard clear                         # clear clipboard
desktop-cli clipboard grab --app "Chrome"           # select-all + copy + read
```
