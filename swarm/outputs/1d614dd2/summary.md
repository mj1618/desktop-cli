# Task: macos-click-and-type-commands (Agent 080be5ef)

## Status: COMPLETED

## What was done

### 1. Added TypeText and KeyCombo to `internal/platform/darwin/inputter.go`
- Added C functions `cg_type_char()` and `cg_key_combo()` for CGEvent-based keyboard simulation
- Added `-framework Carbon` to LDFLAGS for virtual key code constants
- Implemented `TypeText()` using `CGEventKeyboardSetUnicodeString` for full Unicode support
- Implemented `KeyCombo()` with modifier key parsing (cmd, shift, ctrl, alt/opt)
- Added complete key code mapping for a-z, 0-9, function keys F1-F12, arrow keys, and special keys (return, tab, space, delete, escape, home, end, pageup, pagedown)
- Added `parseKeyCombo()` helper to resolve key names and modifiers

### 2. Wired `type` command in `cmd/typecmd.go`
- Replaced `notImplemented("type")` stub with full `runType()` implementation
- Supports `--text` flag and positional text argument
- Supports `--key` for key combinations (e.g., "cmd+c", "ctrl+shift+t")
- Supports `--delay` for inter-keystroke delay in milliseconds
- Supports `--id` + `--app` to click-focus an element before typing
- Returns structured YAML result on success
- Proper error messages for missing args, missing --app with --id, element not found

### 3. Updated README.md and SKILL.md
- Expanded type command documentation with all supported usage patterns

## Files Modified
- `internal/platform/darwin/inputter.go` - Added TypeText, KeyCombo, key code maps, parseKeyCombo
- `cmd/typecmd.go` - Full type command implementation
- `README.md` - Expanded type command docs
- `SKILL.md` - Expanded type command docs

## Verification
- `go build ./...` succeeds
- `go test ./...` passes (all packages)

## Notes
- Other agents had already created: inputter.go (Click/MoveMouse), cmd/helpers.go (findElementByID), cmd/click.go (wired), init.go (Inputter registered)
- Another agent also added Scroll implementation while this work was in progress - no conflicts
