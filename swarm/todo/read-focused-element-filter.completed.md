# Enhancement: Add `--focused` Filter to `read` Command

## Priority: HIGH (Saves a full tree read + manual scan when agents need to check focus)

## Problem

After pressing Tab, clicking, or performing any navigation action, agents frequently need to know which element currently has focus. Today, this requires:

1. Running a full `desktop-cli read --app "Chrome" --flat` (returns hundreds of elements)
2. Scanning the entire output for the element with `f: true`

This is wasteful — the agent only needs one element, but gets back the entire UI tree. In a complex app like Gmail, this can be 3000+ elements, consuming significant tokens and time.

There is no way to directly ask "what element is currently focused?"

## What to Build

### 1. Add `--focused` Flag to `read` Command

```bash
desktop-cli read --app "Google Chrome" --focused
```

This should return only the currently focused element (the one with `f: true` in the accessibility tree):

```yaml
app: Google Chrome
ts: 1770638906
elements:
    - i: 3463
      r: input
      d: Subject
      v: ""
      b: [550, 792, 421, 20]
      f: true
```

### 2. Combine with Other Filters

The `--focused` flag should work alongside existing filters:

```bash
# Get the focused element, but only if it's an input
desktop-cli read --app "Chrome" --focused --roles "input"

# Get the focused element within a specific bounding box
desktop-cli read --app "Chrome" --focused --bbox "0,600,1000,500"
```

### 3. Implementation Approach

In the read command's element traversal:
1. Walk the full element tree as normal
2. If `--focused` is set, filter to only elements where `Focused == true`
3. Return just that element (or empty if nothing is focused in the scoped area)
4. Works with both tree and `--flat` output modes

### 4. Update Documentation

Update SKILL.md Agent Workflow to recommend `read --focused` after key presses, and add to the Quick Reference.

## Files to Modify

- `cmd/read.go` — Add `--focused` flag, filter logic
- `internal/platform/platform.go` — Add `Focused` field to `ReadOptions` if needed
- `README.md` — Add `--focused` to read command examples
- `SKILL.md` — Add to Quick Reference and Agent Workflow

## Acceptance Criteria

- [ ] `desktop-cli read --app "Finder" --focused` returns only the focused element
- [ ] `desktop-cli read --app "Finder" --focused --flat` returns the focused element in flat mode with path breadcrumb
- [ ] `desktop-cli read --app "Finder" --focused --roles "input"` returns the focused element only if it matches the role filter
- [ ] When no element is focused, returns an empty elements list
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] README.md and SKILL.md updated

## Implementation Notes

- This is a client-side filter on the already-read element tree. No new platform API calls needed.
- The focused element is typically deep in the tree, so the traversal still needs to walk the full depth. The filtering happens on the output side.
- Consider returning the focused element's ancestry path (in flat mode) so the agent knows the context (e.g., "window > group > web > group > input").
