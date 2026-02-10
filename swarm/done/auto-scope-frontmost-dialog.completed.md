# Feature: Auto-Scope to Frontmost Dialog/Modal

## Priority: HIGH (eliminates the #1 disambiguation error class)

## Problem

When a modal dialog, sheet, popover, or alert is open, text-based element targeting matches background elements behind the overlay. This is the single most common source of disambiguation errors:

**Gmail compose dialog open:**
```bash
desktop-cli click --text "Subject" --app "Google Chrome"
# Error: multiple elements match text "Subject":
#   id=3369 role=input desc="Subject" (the compose field - CORRECT)
#   id=364 role=chk title="unread, me, Test Subject..." (inbox row - WRONG)
#   id=420 role=chk title="unread, me, test subject..." (inbox row - WRONG)
#   ... 14 more background matches
```

The agent must add `--scope-id` or `--roles "input"` to disambiguate — requiring either a prior `read` to discover the scope ID, or domain knowledge about which role to use. Both add round-trips and friction.

**Calculator (already fixed by `preferInteractiveElements`, but illustrative):**
```bash
desktop-cli click --text "3" --app "Calculator"
# Used to error: matches txt (display) AND btn (button)
```

The focused element is always inside the frontmost context (dialog, popover, main window). The tool already uses focus proximity to narrow matches — but it only helps when the focused element happens to share a deep ancestor with the target. For cases like Gmail compose (focused on body, targeting Subject), focus proximity doesn't help enough because both are in the same dialog.

## What to Build

### 1. Detect Frontmost Context Automatically

When resolving elements by text, detect whether a modal/dialog/sheet/popover is open and auto-scope the search to its subtree:

```go
func resolveElementByText(...) (*model.Element, []model.Element, error) {
    elements, err := provider.Reader.ReadElements(opts)

    // NEW: detect and auto-scope to frontmost overlay
    searchScope := elements
    if scopeID == 0 { // only auto-scope when user hasn't manually scoped
        if overlay := detectFrontmostOverlay(elements); overlay != nil {
            searchScope = overlay.Children
        }
    }

    matches := collectLeafMatches(searchScope, textLower, roleSet, exact)
    // ... rest of disambiguation ...
}
```

### 2. Overlay Detection Heuristic

Detect overlays by looking for common accessibility patterns:

```go
func detectFrontmostOverlay(elements []model.Element) *model.Element {
    // Strategy 1: Look for elements with dialog/sheet/popover roles
    // macOS maps these to: AXSheet, AXDialog, AXPopover, AXMenu
    // Our role mapping: "group" or "other" with specific subrole hints

    // Strategy 2: Find the focused element, then walk up the tree
    // If any ancestor has role "group"/"other" AND is NOT the top-level window,
    // AND it overlaps with (covers) a significant portion of the window,
    // treat it as a dialog/overlay

    // Strategy 3: Check for the "AXDialog" or "AXSheet" subrole directly
    // (requires exposing subrole from the Darwin reader)

    // Return the overlay element, or nil if no overlay detected
}
```

Concrete heuristics (try in order):
1. **AXRole-based**: If any top-level child of the window has role `sheet`, `dialog`, or native subrole `AXSheet`/`AXDialog`, that's the overlay.
2. **Focus-based**: Find the focused element, walk up to its nearest ancestor that is a direct child of the window. If that ancestor is NOT the main content area (heuristic: it appeared more recently, or has fewer siblings), treat it as the overlay context.
3. **Bounds-based**: If a top-level element overlaps the center of the window and has a smaller bounding box than the window (i.e., it's a centered dialog), treat it as an overlay.

### 3. Expose Subrole from macOS Accessibility API

The macOS accessibility API provides a `subrole` attribute (`AXSubrole`) that distinguishes dialogs from regular groups. Current implementation doesn't expose it. Add:

```go
// In internal/model/element.go
type Element struct {
    // ... existing fields ...
    Subrole string `yaml:"sr,omitempty" json:"sr,omitempty"` // AXSubrole (e.g. "AXDialog", "AXSheet")
}
```

```objc
// In internal/platform/darwin/reader.m
NSString *subrole = nil;
AXUIElementCopyAttributeValue(element, kAXSubroleAttribute, (CFTypeRef *)&subrole);
```

This makes dialog detection reliable rather than heuristic-based.

### 4. Opt-Out Flag

Add `--no-auto-scope` to disable the automatic overlay detection for cases where the agent explicitly wants to search the entire tree:

```bash
# Default: auto-scopes to frontmost dialog
desktop-cli click --text "Subject" --app "Chrome"

# Explicit: search entire tree including background
desktop-cli click --text "Subject" --app "Chrome" --no-auto-scope
```

### 5. Indicate Auto-Scoping in Responses

When auto-scoping activates, include it in the response so the agent knows what happened:

```yaml
ok: true
action: click
auto_scoped: "dialog (id=3200)"   # indicates auto-scoping was applied
target:
    i: 3369
    r: input
    d: "Subject"
    b: [550, 792, 421, 20]
```

This transparency helps agents understand why they're getting different results than expected.

### 6. Usage Examples

```bash
# Before (fails — matches background inbox rows):
desktop-cli click --text "Subject" --app "Chrome"

# After (auto-scopes to compose dialog — just works):
desktop-cli click --text "Subject" --app "Chrome"
# Response includes: auto_scoped: "dialog (id=3200)"

# Agent doesn't need to change anything — disambiguation just improves
desktop-cli type --target "To" --app "Chrome" --text "alice@example.com"
desktop-cli action --text "Send" --app "Chrome"
```

## Files to Create

- `internal/model/overlay.go` — Overlay detection logic

## Files to Modify

- `internal/model/element.go` — Add `Subrole` field
- `internal/platform/darwin/reader.go` (or .m/.c) — Read `AXSubrole` attribute
- `cmd/helpers.go` — Add `detectFrontmostOverlay()` and integrate into `resolveElementByText()`
- `cmd/click.go` — Add `--no-auto-scope` flag
- `cmd/typecmd.go` — Add `--no-auto-scope` flag
- `cmd/action.go` — Add `--no-auto-scope` flag
- `cmd/setvalue.go` — Add `--no-auto-scope` flag
- `README.md` — Document auto-scoping behavior
- `SKILL.md` — Update disambiguation guidance

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] When a dialog/sheet is open, `click --text` auto-scopes to the dialog
- [ ] Background elements are excluded from matches when auto-scope is active
- [ ] `--no-auto-scope` disables the behavior
- [ ] Response includes `auto_scoped` field when scoping was applied
- [ ] Subrole field is populated for elements that have one
- [ ] Works for macOS sheets (e.g., Save dialog), alerts, and popovers
- [ ] Falls back to full-tree search when no overlay is detected
- [ ] No regression: existing commands without overlays work identically
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Subrole is the key**: `AXSubrole` is the most reliable signal. `AXDialog`, `AXSheet`, `AXSystemDialog`, `AXFloatingWindow` all indicate overlays. Without subrole, heuristics are fragile.
- **Performance**: Overlay detection adds one tree walk (~O(n) where n = top-level children) before the existing text search. Negligible overhead since the tree is already in memory.
- **False positives**: If auto-scoping incorrectly excludes the target, the agent gets "no element found" and can retry with `--no-auto-scope`. The error message should suggest this: `"no element found matching text "X" in dialog scope — try --no-auto-scope to search entire window"`.
- **Interaction with `--scope-id`**: If the user provides `--scope-id`, auto-scoping is disabled (user explicitly chose a scope).
- **Web page dialogs**: Chrome/web dialogs might NOT have `AXSheet`/`AXDialog` subrole. The bounds-based heuristic (centered element smaller than window) can catch these. But this is lower priority — start with native dialogs.
- **Depends on**: No hard dependencies. Can be implemented independently.
