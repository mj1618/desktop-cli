# Fill Command Implementation Summary

## Task: `compound-fill-command`
## Agent: b50ad524

## What Was Built

Implemented the `fill` command — a compound form-filling command that sets multiple form fields in a single CLI call, reading the UI tree only once.

## Files Created
- `cmd/fill.go` — New fill command with ~380 lines covering:
  - `--field "Label=value"` flag (repeatable) for specifying fields
  - `--field "id:42=value"` syntax for ID-based targeting
  - `--method set-value|type` for choosing between direct value setting and keystroke simulation
  - `--submit "Submit"` to click a submit button after filling
  - `--tab-between` for Tab-key navigation between fields
  - YAML stdin input for many fields
  - `--post-read` support for capturing UI state after filling
  - Per-field error reporting (individual field failures don't block others)
  - `executeFill()` function for use as a `do` batch step

## Files Modified
- `cmd/helpers.go` — Added `resolveElementByTextFromTree()` helper that resolves elements from a pre-read tree (avoiding redundant tree reads)
- `cmd/do.go` — Added `fill` as a supported step type in the batch command
- `README.md` — Added fill command documentation, feature bullet point, and updated `do` step type list
- `SKILL.md` — Added fill quick reference, updated batch step list, and added fill to agent workflow

## Key Design Decisions
1. **Single tree read**: The fill command's primary optimization — reads the tree once and resolves all fields from the same snapshot, avoiding N separate tree reads for N fields
2. **`resolveElementByTextFromTree` helper**: Extracted text matching logic that works on a pre-read tree, reusable by other commands
3. **Default method `set-value`**: Direct accessibility API value setting is instant and doesn't require focus management; `type` mode available for apps that need keystroke events
4. **Cmd+A before type**: When using `--method type`, selects all before typing to replace existing content
5. **Graceful submit fallback**: If submit element not found in original tree, re-reads in case form changed after filling

## Verification
- `go build ./...` — passes
- `go test ./...` — all tests pass
- `desktop-cli fill --help` — shows correct usage and flags
