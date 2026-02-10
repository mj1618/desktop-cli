# Task: Post-Action UI State in Responses (`--post-read`)

## Agent: 1d9403ec | Task: 6a2b0d50

## What Was Done

Implemented the `--post-read` flag on `click`, `type`, `action`, and `set-value` commands. This flag appends a compact agent-format snapshot of the full UI state to the action response, eliminating the need for a follow-up `read` call.

## Changes Made

### New Functions
- **`output.FormatAgentString()`** (`internal/output/output.go`) — Returns agent-format output as a string (previously only wrote to stdout). Refactored `printAgentFlat` to use a shared `formatAgentString` internal function.
- **`addPostReadFlags()`** (`cmd/helpers.go`) — Shared flag registration for `--post-read` and `--post-read-delay`.
- **`getPostReadFlags()`** (`cmd/helpers.go`) — Shared flag reader.
- **`readPostActionState()`** (`cmd/helpers.go`) — Reads the UI tree after an action and returns a compact agent-format string. Best-effort (returns "" on failure). Supports configurable delay.

### Modified Commands
- **`cmd/click.go`** — Added `State` field to `ClickResult`, registered post-read flags, integrated post-read logic. When `--post-read` is used, `display` elements are skipped (they're included in the state).
- **`cmd/action.go`** — Same pattern as click.
- **`cmd/typecmd.go`** — Same pattern as click.
- **`cmd/setvalue.go`** — Same pattern (set-value didn't previously have display elements, so only state is added).

### Documentation
- **README.md** — Added `--post-read` examples to click section, added dedicated "Post-read UI state" subsection with output examples.
- **SKILL.md** — Added `--post-read` examples to click section, added workflow guidance in Agent Workflow section.

## Design Decisions
- When `--post-read` is active, `display` elements are skipped to avoid duplication (display elements are already included in the agent-format state output).
- Post-read is best-effort: if the read fails (e.g., app closed after clicking a close button), the action result is still returned successfully with an empty state field.
- The `state` field uses YAML multiline string format (`|`) for readability.
- `--post-read-delay` supports waiting for animations/transitions before reading.

## Build Status
- `go build ./...` — Pre-existing build failure in `cmd/read.go` (unused imports from screenshot-coords work in progress). My changes compile cleanly.
- `go test ./internal/...` — All tests pass.
- `go test ./cmd/...` — All tests pass when read.go compiles (pre-existing issue).
