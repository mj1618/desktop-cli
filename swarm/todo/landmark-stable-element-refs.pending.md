# Feature: Landmark-Based Stable Element References

## Priority: HIGH (eliminates re-reads needed to rediscover elements)

## Problem

Element IDs are assigned sequentially during each `read` call and are **not stable across reads**. The same "Submit" button might be `id=89` on one read and `id=91` on the next (because an element was added/removed earlier in the tree).

This forces a wasteful pattern:

```bash
# Read 1: discover the form
desktop-cli read --app "Safari" --format agent
# [42] input "Full Name" ...
# [89] btn "Submit" ...

# Fill the form (tree changes as values are set)
desktop-cli type --target "Full Name" --app "Safari" --text "John"

# Read 2: must re-read because IDs may have shifted
desktop-cli read --app "Safari" --format agent
# [42] input "Full Name" val="John" ...
# [90] btn "Submit" ...       ← ID CHANGED from 89 to 90!

# Now must use the new ID
desktop-cli click --id 90 --app "Safari"
```

Agents can't reliably plan multi-step workflows using IDs because the IDs shift between reads. They must either:
- Use `--text` targeting for every action (slower, ambiguous)
- Re-read before every `--id` action (wasteful)

## What to Build

### 1. Content-Based Stable References

Generate a stable string reference for each element based on its semantic identity — role, label, and position in the tree hierarchy:

```
toolbar > input "Search"
dialog > btn "OK"
menubar > menu "File" > menuitem "New Window"
window > group > form > input "Email"
```

These references persist across reads as long as the element's semantic identity doesn't change. An input labeled "Email" in a form inside a window will always have the same reference, regardless of how many elements are added/removed elsewhere in the tree.

### 2. `ref` Field in Output

Include the stable reference alongside the numeric ID in all output formats:

**Agent format:**
```
[42|toolbar/search] input "Search" (200,50,800,30)
[89|dialog/ok-btn] btn "OK" (400,500,100,32)
[90|form/email] input "Email" val="john@example.com" (100,140,200,30)
```

**YAML format:**
```yaml
- i: 42
  ref: "toolbar/search"
  r: input
  t: "Search"
  b: [200, 50, 800, 30]
```

### 3. `--ref` Flag for Targeting

Allow using stable references in commands instead of numeric IDs:

```bash
# By stable reference (persists across reads):
desktop-cli click --ref "dialog/ok-btn" --app "Safari"
desktop-cli type --ref "form/email" --app "Safari" --text "john@example.com"
desktop-cli action --ref "toolbar/search" --app "Safari"
desktop-cli set-value --ref "form/email" --value "john@example.com" --app "Safari"

# Still works: by numeric ID (unstable, but fast within same read):
desktop-cli click --id 89 --app "Safari"

# Still works: by text (substring match):
desktop-cli click --text "OK" --app "Safari"
```

### 4. Reference Generation Algorithm

```go
func generateRef(el model.Element, parent string) string {
    // Build segment from role + disambiguating label
    segment := el.Role
    label := bestLabel(el) // title > description > value (first non-empty)
    if label != "" {
        // Slugify: lowercase, replace spaces with hyphens, remove special chars
        slug := slugify(label)
        segment = slug
    }

    // Combine with parent path
    if parent != "" {
        return parent + "/" + segment
    }
    return segment
}

func bestLabel(el model.Element) string {
    if el.Title != "" { return el.Title }
    if el.Description != "" { return el.Description }
    // Don't use Value — it changes (input field content, slider position)
    return ""
}

// Example outputs:
// toolbar/search
// dialog/submit
// menubar/file/new-window
// form/full-name
// form/email
// sidebar/inbox-23288-unread (link with title "Inbox 23288 unread")
```

Handle duplicates by appending a 1-based index:
```
# Two "OK" buttons in the same dialog:
dialog/ok.1
dialog/ok.2
```

### 5. Reference Resolution

When `--ref` is provided, resolve it by:
1. Read the full element tree
2. Generate refs for all elements
3. Find the element whose ref matches the provided value
4. Support partial matching: `--ref "submit"` matches `dialog/form/submit` if unambiguous

```go
func resolveElementByRef(elements []model.Element, ref string) (*model.Element, error) {
    // Generate refs for all elements
    refs := generateAllRefs(elements)

    // Exact match first
    if el, ok := refs[ref]; ok {
        return el, nil
    }

    // Partial match: find refs ending with the provided value
    var matches []*model.Element
    for r, el := range refs {
        if strings.HasSuffix(r, ref) || strings.HasSuffix(r, "/"+ref) {
            matches = append(matches, el)
        }
    }

    if len(matches) == 1 { return matches[0], nil }
    if len(matches) == 0 { return nil, fmt.Errorf("no element matches ref %q", ref) }
    return nil, fmt.Errorf("multiple elements match ref %q: ...", ref)
}
```

### 6. Compact Ref Format

Full paths can be long. Use abbreviations for common ancestry:

```
# Full: window/toolbar/group/btn "Back"
# Compact: toolbar/back

# Full: window/group/scroll/group/list/row/cell/input "Email"
# Compact: list/row.3/email
```

Rules:
- Skip `window` (always implied)
- Skip anonymous `group` elements (no label)
- Skip `scroll` containers
- Keep `toolbar`, `dialog`, `menu`, `list`, `form` as landmarks
- Keep the element's own label as the final segment

### 7. Usage Examples

```bash
# First read: discover elements with refs
desktop-cli read --app "Safari" --format agent
# [42|toolbar/address] input "Address" val="https://example.com" (200,50,800,30)
# [89|form/name] input "Full Name" (100,100,200,30)
# [90|form/email] input "Email" (100,140,200,30)
# [91|form/submit] btn "Submit" (100,200,100,40)

# Now act using refs (stable across reads):
desktop-cli type --ref "form/name" --app "Safari" --text "John Doe"
desktop-cli type --ref "form/email" --app "Safari" --text "john@example.com"
desktop-cli click --ref "form/submit" --app "Safari"

# Partial ref (unambiguous):
desktop-cli click --ref "submit" --app "Safari"
```

## Files to Create

- `internal/model/refs.go` — Reference generation algorithm
- `internal/model/refs_test.go` — Unit tests for ref generation

## Files to Modify

- `internal/model/element.go` — Add `Ref` field to Element struct
- `internal/output/output.go` — Include refs in all output formats
- `cmd/helpers.go` — Add `resolveElementByRef()`, add `--ref` flag support
- `cmd/click.go` — Add `--ref` flag
- `cmd/typecmd.go` — Add `--ref` flag
- `cmd/action.go` — Add `--ref` flag
- `cmd/setvalue.go` — Add `--ref` flag
- `cmd/scroll.go` — Add `--ref` flag
- `cmd/drag.go` — Add `--ref` flags (from-ref, to-ref)
- `README.md` — Document refs system
- `SKILL.md` — Add ref examples

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `read --format agent` includes refs in output: `[42|toolbar/search]`
- [ ] `read --format yaml` includes `ref` field on each element
- [ ] Same element produces same ref across consecutive reads
- [ ] `click --ref "form/submit"` resolves and clicks the element
- [ ] `type --ref "form/email" --text "..."` resolves and types into the element
- [ ] Partial ref matching works: `--ref "submit"` matches `form/submit`
- [ ] Duplicate labels get indexed: `dialog/ok.1`, `dialog/ok.2`
- [ ] Anonymous groups are skipped in ref paths
- [ ] `--ref` and `--id` can both be used (but not together)
- [ ] `--ref` and `--text` can both be used (but not together)
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Stability guarantee**: Refs are stable as long as the element's role, label, and tree position don't change. Adding/removing a sibling won't change a ref (unlike numeric IDs). But renaming a button or moving it to a different container will change its ref.
- **Not a cache**: Refs are regenerated on each read. They're deterministic (same tree → same refs), so they're effectively stable. No persistence needed.
- **Performance**: Ref generation is O(n) — one pass through the tree. Adds ~1ms to a 200-element tree. Negligible.
- **Ref vs text vs ID**: Three targeting modes, each with tradeoffs:
  - `--id 42` — fastest (no tree walk), but unstable across reads
  - `--text "Submit"` — most natural, but ambiguous when multiple elements match
  - `--ref "form/submit"` — stable and precise, but requires knowing the ref from a prior read
- **Interaction with `--scope-id`**: `--ref` can be combined with `--scope-id` to search only within a container. But refs already encode hierarchy, so `--ref "dialog/submit"` implicitly scopes to the dialog.
- **Agent workflow change**: Agents would read once with `--format agent`, note the refs for elements they'll interact with, then use `--ref` for all subsequent commands. No re-reads needed unless the UI changes dramatically (page navigation, new dialog).
