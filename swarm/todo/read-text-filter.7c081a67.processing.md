# Feature: Add `--text` Filter to `read` Command for Text-Based Element Search

## Priority: HIGH (reduces agent token usage and simplifies the most common workflow)

## Problem

The most common agent workflow is:
1. `read` the full element tree (potentially hundreds of elements, thousands of tokens)
2. Parse the YAML output to find a specific element by its label text (e.g. "Submit", "Search", "Sign In")
3. Extract the element's `i` (ID) field
4. Use that ID with `click --id`, `type --id`, or `action --id`

Step 2 is wasteful — the agent downloads and parses the entire tree just to find one element. This burns tokens and adds latency, especially for complex UIs like web browsers where the element tree can have hundreds of nodes.

The `wait` command already has `--for-text` which does substring matching against title, value, and description. The `read` command should support the same text filtering, and also support returning results as a flat list rather than a deeply nested tree.

## What to Build

### 1. Add `--text` flag to `read` command — `cmd/read.go`

Add a `--text` flag that filters elements to only those whose title, value, or description contains the given substring (case-insensitive). This works in combination with existing filters (`--roles`, `--bbox`, `--depth`).

```bash
# Find all elements containing "Submit" in their title/value/description
desktop-cli read --app "Safari" --text "Submit"

# Find buttons containing "Save"
desktop-cli read --app "Safari" --text "Save" --roles "btn"

# Find text fields containing "Email"
desktop-cli read --app "Chrome" --text "Email" --roles "input"
```

### 2. Add `--flat` flag to `read` command — `cmd/read.go`

Add a `--flat` flag that outputs matching elements as a flat list rather than a nested tree. This is much easier for agents to parse and significantly reduces token usage by eliminating irrelevant parent/child nesting.

When `--flat` is used:
- All matching elements are returned in a single list (no `c` children field)
- Each element includes a `p` (path) field: a breadcrumb string showing the element's location in the tree (e.g. `"window > toolbar > group"`)
- Elements are ordered by traversal order (same as ID assignment)

```bash
# Flat list of all buttons
desktop-cli read --app "Safari" --roles "btn" --flat

# Find a specific element by text, get it as a flat result
desktop-cli read --app "Safari" --text "Submit" --flat
```

Example output with `--flat`:
```yaml
app: Safari
pid: 1234
ts: 1707500000
elements:
- i: 5
  r: btn
  t: Submit
  b: [400, 300, 80, 32]
  a: [press]
  p: window > web > group > form
- i: 23
  r: btn
  t: Submit Review
  b: [600, 500, 120, 32]
  a: [press]
  p: window > web > group > section
```

Without `--flat`, the same query would return the full nested tree with all parent and sibling elements, consuming far more tokens.

### 3. Add text filtering to `model.FilterElements` — `internal/model/filter.go`

Extend the existing `FilterElements` function (or add a new `FilterByText` function) to support text substring matching.

```go
// FilterByText filters elements to only those whose title, value, or
// description contains the given text (case-insensitive). It recursively
// searches children and returns matching elements.
func FilterByText(elements []Element, text string) []Element {
    if text == "" {
        return elements
    }
    textLower := strings.ToLower(text)
    var result []Element
    for _, el := range elements {
        matched := textMatchesElement(el, textLower)
        // Recursively check children regardless of parent match
        childMatches := FilterByText(el.Children, text)

        if matched {
            filtered := el
            filtered.Children = childMatches
            result = append(result, filtered)
        } else if len(childMatches) > 0 {
            // Include parent with only matching children
            filtered := el
            filtered.Children = childMatches
            result = append(result, filtered)
        }
    }
    return result
}

func textMatchesElement(el Element, textLower string) bool {
    return strings.Contains(strings.ToLower(el.Title), textLower) ||
        strings.Contains(strings.ToLower(el.Value), textLower) ||
        strings.Contains(strings.ToLower(el.Description), textLower)
}
```

### 4. Add tree flattening to `internal/model/flatten.go` (new file)

```go
package model

import "strings"

// FlatElement is an element with a path breadcrumb instead of children.
type FlatElement struct {
    Element
    Path string `yaml:"p,omitempty"`
}

// FlattenElements converts a tree of elements into a flat list,
// optionally filtering by a text predicate. Each element gets a
// path string showing its location in the tree.
func FlattenElements(elements []Element, filterText string) []FlatElement {
    textLower := strings.ToLower(filterText)
    var result []FlatElement
    for _, el := range elements {
        flattenRecursive(el, "", textLower, &result)
    }
    return result
}

func flattenRecursive(el Element, parentPath string, textLower string, result *[]FlatElement) {
    currentPath := el.Role
    if parentPath != "" {
        currentPath = parentPath + " > " + el.Role
    }

    // Check if this element matches (if text filter is active)
    matches := true
    if textLower != "" {
        matches = strings.Contains(strings.ToLower(el.Title), textLower) ||
            strings.Contains(strings.ToLower(el.Value), textLower) ||
            strings.Contains(strings.ToLower(el.Description), textLower)
    }

    if matches {
        flat := FlatElement{
            Element: el,
            Path:    currentPath,
        }
        flat.Children = nil // Remove children from flat output
        *result = append(*result, flat)
    }

    // Always recurse into children
    for _, child := range el.Children {
        flattenRecursive(child, currentPath, textLower, result)
    }
}
```

### 5. Update `ReadOptions` — `internal/platform/types.go`

Add `Text` field to `ReadOptions`:

```go
type ReadOptions struct {
    App         string
    Window      string
    WindowID    int
    PID         int
    Depth       int
    Roles       []string
    VisibleOnly bool
    BBox        *Bounds
    Compact     bool
    Text        string   // Filter by text content (title, value, description)
    Flat        bool     // Return flat list instead of tree
}
```

### 6. Update `output.ReadResult` — `internal/output/yaml.go`

Add a variant or field to support flat element output:

```go
// ReadFlatResult is the output when --flat is used.
type ReadFlatResult struct {
    App      string             `yaml:"app,omitempty"`
    PID      int                `yaml:"pid,omitempty"`
    Window   string             `yaml:"window,omitempty"`
    TS       int64              `yaml:"ts"`
    Elements []model.FlatElement `yaml:"elements"`
}
```

### 7. Wire it up in `cmd/read.go`

In `runRead`, after reading elements:

```go
// Apply text filter
if text != "" {
    elements = model.FilterByText(elements, text)
}

// Output as flat list or tree
if flat {
    flatElements := model.FlattenElements(elements, "")  // text already filtered
    result := output.ReadFlatResult{
        App:      appName,
        PID:      pid,
        TS:       time.Now().Unix(),
        Elements: flatElements,
    }
    return output.PrintYAML(result)
}
```

## Files to Create

- `internal/model/flatten.go` — `FlatElement` struct and `FlattenElements` function
- `internal/model/flatten_test.go` — Tests for flattening logic

## Files to Modify

- `cmd/read.go` — Add `--text` and `--flat` flags, wire up filtering and flattening
- `internal/model/filter.go` — Add `FilterByText` function
- `internal/model/filter_test.go` — Add tests for text filtering
- `internal/platform/types.go` — Add `Text` and `Flat` fields to `ReadOptions`
- `internal/output/yaml.go` — Add `ReadFlatResult` struct
- `README.md` — Add examples for `--text` and `--flat` flags
- `SKILL.md` — Add `--text` and `--flat` to quick reference

## Dependencies

- None (uses existing `read` infrastructure)

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `desktop-cli read --app "Finder" --text "Downloads"` returns only elements containing "Downloads" in title/value/description
- [ ] `desktop-cli read --app "Safari" --text "Submit" --roles "btn"` combines text and role filtering
- [ ] Text matching is case-insensitive
- [ ] Text matches against title, value, AND description (any match counts)
- [ ] `desktop-cli read --app "Finder" --flat` returns all elements as a flat list with path breadcrumbs
- [ ] `desktop-cli read --app "Safari" --text "Submit" --flat` returns flat list of matching elements only
- [ ] Flat output has `p` field with role-based path breadcrumbs (e.g. `window > toolbar > btn`)
- [ ] Flat output omits `c` (children) field
- [ ] `--text` without `--flat` returns the tree structure but pruned to only branches containing matching elements
- [ ] Element IDs in filtered/flat output are the same as in unfiltered output (IDs are assigned during traversal, not after filtering)
- [ ] `--text ""` (empty string) is treated as no filter
- [ ] `--flat` without `--text` returns all elements flattened
- [ ] README.md and SKILL.md updated with new flag examples
- [ ] FilterByText has unit tests covering: exact match, substring, case-insensitive, multi-field match, no match, empty text
- [ ] FlattenElements has unit tests covering: basic flattening, path generation, text filtering, nested elements

## Implementation Notes

- **Text filtering happens AFTER the platform reader returns elements.** The platform reader applies depth, visibility, roles, and bbox filters during traversal. Text filtering is a post-processing step in the command layer, just like the existing role/bbox filtering in `model.FilterElements`. This keeps the platform abstraction clean.
- **Element IDs must be preserved.** IDs are assigned during tree traversal by the platform reader. Text filtering and flattening must NOT reassign IDs — the whole point is that an agent can `read --text "Submit" --flat` to get an element's ID, then immediately use that ID with `click --id` or `action --id`.
- **Flat output is complementary to `--text`.** `--flat` is useful even without `--text` — it gives agents a simple list they can scan sequentially without parsing nested YAML. `--text` is useful even without `--flat` — it prunes the tree to relevant branches. Together they're most powerful: `--text "Submit" --flat` gives the agent exactly the elements it needs in the simplest format.
- **Path breadcrumbs use short role names.** The `p` field uses the same abbreviated role names as the `r` field (btn, txt, lnk, etc.) joined with ` > `. This keeps it compact while giving agents hierarchical context about where the element lives in the UI tree.
- **The `wait` command already has text matching logic.** The `matchesCondition` function in `cmd/wait.go` implements the same case-insensitive substring matching. Consider extracting the matching logic to a shared function in `internal/model/` to avoid duplication. This is optional — the duplication is small.
- **Token savings estimate:** For a typical browser page with 200 elements, `read --roles "btn" --text "Submit" --flat` might return 1-3 elements instead of 200, reducing output from ~2000 tokens to ~50 tokens — a 40x reduction.
