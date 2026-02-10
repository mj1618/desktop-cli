# Hover Command Implementation

## Task
Implemented the `hover` command that moves the mouse cursor to an element (or coordinates) without clicking. This enables agents to trigger hover-dependent UI like tooltips, row actions, and flyout menus.

## Changes Made

### New Files
- **`cmd/hover.go`** — New command following the `click.go` pattern. Supports:
  - `--text` — Find and hover over element by text (case-insensitive)
  - `--id` — Hover by element ID
  - `--x/--y` — Hover at absolute screen coordinates
  - `--app`, `--window` — Scope element lookups
  - `--roles`, `--exact`, `--scope-id` — Text targeting refinement
  - `--post-read`, `--post-read-delay` — Capture UI state after hovering

### Modified Files
- **`cmd/do.go`** — Added `hover` as a supported step type in the batch `do` command with `executeHover()` function
- **`README.md`** — Added hover command documentation with examples
- **`SKILL.md`** — Added hover command reference and agent workflow guidance

## Key Design Decisions
- **No click**: The entire purpose is mouse movement without clicking (agents already have `click` for that)
- **`--post-read` support**: Critical for hover use cases — agents hover to reveal hidden UI, then read the new state
- **Reused existing infrastructure**: Same text targeting, element resolution, and post-read helpers as `click`
- **Platform support already existed**: `MoveMouse` was already in the `Inputter` interface and macOS implementation (`cg_move_mouse`)

## Testing
- All existing tests pass (`go test ./...` — 6 packages OK)
- Build succeeds (`go build ./...`)
- Command registration verified (`desktop-cli hover --help`)
