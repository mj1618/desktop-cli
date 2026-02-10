# Fix: --near prefers content area matches over sidebar/preview matches

## Problem
When using `click --text "Buy groceries" --near --app "Notes"`, the command matched the note preview text in the sidebar instead of the actual checklist item in the main content area. This happened because `resolveElementByText` would either pick the first match or error on ambiguity, and the `--near` proximity search was anchored to whichever match it picked.

## Solution
When `--near` is used, the click command now:

1. **Gets ALL text matches** instead of requiring a single unambiguous match (new `resolveAllTextMatches` function)
2. **Picks the best match for content-area targeting** using `pickBestNearMatch`:
   - Prefers rightmost matches (macOS apps use left sidebar + right content layouts)
   - Among matches at similar X positions (within 50px), falls back to focus proximity
3. **Then finds the nearest interactive element** to the best match

This is applied in both the `click` command and the `do` batch command's click step.

## Files Changed
- `cmd/helpers.go` — Added `resolveAllTextMatches()` and `pickBestNearMatch()` functions
- `cmd/click.go` — Separated `--near` path to use the new multi-match resolution
- `cmd/do.go` — Same fix applied to the batch `click` step
- `cmd/helpers_test.go` — Added 4 tests covering the multi-pane scenario

## Tests Added
- `TestPickBestNearMatch_PrefersContentArea` — Verifies rightmost match wins (sidebar vs content)
- `TestPickBestNearMatch_ContentAreaCheckbox` — End-to-end: text match → best match → nearest checkbox
- `TestPickBestNearMatch_SingleMatch` — Single match returns directly (no regression)
- `TestPickBestNearMatch_SameXUsesFocusProximity` — Same X position falls back to focus proximity

All existing tests continue to pass.
