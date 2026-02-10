# Auto-Scope to Frontmost Dialog/Modal

**Task**: auto-scope-frontmost-dialog
**Agent**: ecd350b4 | **Task ID**: 41ddd760
**Status**: Completed

## Summary

Implemented automatic scoping of text-based element targeting to the frontmost overlay (dialog, sheet, modal) when one is detected. This eliminates the #1 disambiguation error class where commands like `click --text "OK"` would match buttons in both a dialog and the background window.

## Changes

### New Files
- **`internal/model/overlay.go`** — Overlay detection with 3 strategies:
  1. Subrole-based: detects `AXDialog`, `AXSheet`, `AXSystemDialog`, `AXSystemFloating`, `AXFloatingWindow`
  2. Focus-based: if focused element is inside a non-first child smaller than the window
  3. Bounds-based: centered element smaller than 80% of the window
- **`internal/model/overlay_test.go`** — 10 test cases covering all detection strategies and helpers

### Modified Files
- **`internal/platform/darwin/accessibility.h`** — Added `subrole` field to `AXElementInfo` C struct
- **`internal/platform/darwin/accessibility.c`** — Read `kAXSubroleAttribute` and free on cleanup
- **`internal/model/element.go`** — Added `Subrole` field to `Element` struct
- **`internal/model/flatten.go`** — Added `Subrole` to `FlatElement` and flattening logic
- **`internal/platform/darwin/reader.go`** — Map C subrole string to Go Element.Subrole
- **`cmd/helpers.go`** — Auto-scope logic in `resolveElementByText` and `resolveElementByTextFromTree`; added `--no-auto-scope` flag; extracted `narrowMatches`/`narrowMatchesSimple` helpers; fixed pre-existing nil pointer bug in `readFocusedElement`
- **`README.md`** — Added "Auto-Scope to Frontmost Dialog" documentation section
- **`SKILL.md`** — Added auto-scope note in agent workflow section

## Design Decisions
- Auto-scope tries overlay first, falls back to full tree if no matches found — avoids false negatives
- `--no-auto-scope` flag allows opting out when needed
- Three-tier detection (subrole → focus → bounds) provides reliable detection with graceful degradation
- Subrole data is read from macOS AXSubrole attribute and carried through the full pipeline

## Bug Fix (Pre-existing)
- Fixed nil pointer dereference in `readFocusedElement` when `provider` is nil (not just `provider.Reader`). This was causing `TestDoContextIfFocused_NoProvider` to fail.

## Tests
All tests pass. Build succeeds.
