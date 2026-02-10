# Bug: Auto Role Expansion for Web Content Is Never Applied

## Summary
The auto role expansion feature (adding "other" to roles when "input" is specified and web content is detected) records a `smart_defaults` message but never actually affects the output. The role expansion is dead code.

## Root Cause
In `cmd/read.go`, the role expansion at line 124 happens **after** `ReadElements()` is called at line 102. The roles are already passed into `ReadOptions` at line 96 and used inside `ReadElements` → `FilterElements` (at `internal/platform/darwin/reader.go:202`). By the time the role expansion modifies the `roles` variable, filtering has already been completed.

## Steps to Reproduce
1. Open Chrome with a page that has input elements exposed as role "other"
2. Run: `desktop-cli read --app "Google Chrome" --roles "input" --format yaml`
3. The `smart_defaults` field may show `auto-expanded roles: added "other" (web input compatibility)` but elements with role "other" are NOT included in the output
4. Compare with: `desktop-cli read --app "Google Chrome" --roles "input,other" --format yaml` — this correctly includes "other" role elements

## Expected Behavior
When `--roles "input"` is specified and web content is detected, elements with role "other" should be included in the output (since Chrome exposes some web inputs as "other").

## Actual Behavior
The `smart_defaults` message is recorded but the role expansion has no effect on the output — the elements are already filtered before expansion occurs.

## Fix
Move the `ExpandRolesForWeb` call to **before** `ReadElements()`, or re-filter elements after expansion. The simplest fix:
1. Call `HasWebContent` by doing a preliminary read without role filtering, or
2. Move the web content detection and role expansion before the `ReadElements` call by splitting the read into two phases, or
3. Apply role filtering as a post-processing step instead of inside `ReadElements`
