# Task: Implement `set-value` Command

## Status: COMPLETED

## What was done

Implemented the `set-value` command for direct AX attribute value setting on UI elements. This allows agents to set text field contents, slider positions, checkbox states, and other element values instantly via the accessibility API instead of simulating keystrokes.

## Files Created
- `internal/platform/darwin/set_value.h` — C header for `ax_set_value()`
- `internal/platform/darwin/set_value.c` — C implementation using `AXUIElementSetAttributeValue()`
- `internal/platform/darwin/value_setter.go` — Go `DarwinValueSetter` implementing `platform.ValueSetter`
- `cmd/setvalue.go` — New `set-value` Cobra command

## Files Modified
- `internal/platform/platform.go` — Added `ValueSetter` interface
- `internal/platform/types.go` — Added `SetValueOptions` struct
- `internal/platform/provider.go` — Added `ValueSetter` field to `Provider`
- `internal/platform/darwin/init.go` — Registered `ValueSetter` in provider
- `cmd/root.go` — Fixed pre-existing initialization cycle (moved `PersistentPreRunE` from var init to `init()` function)
- `README.md` — Added set-value command documentation
- `SKILL.md` — Added set-value to quick reference and updated agent workflow

## Key Implementation Details
- C code auto-detects value type by querying current attribute type (`CFStringRef`, `CFNumberRef`, `CFBooleanRef`)
- Traversal order matches `ax_read_elements` for consistent element IDs
- Supports `--attribute` flag: `value` (default), `selected`, `focused`, or raw AX names
- Error messages are clear for all failure cases (missing flags, invalid element, unsupported platform)

## Build & Test
- `go build ./...` succeeds
- `go test ./...` passes (all 5 test packages)
- `desktop-cli set-value --help` shows all flags correctly
