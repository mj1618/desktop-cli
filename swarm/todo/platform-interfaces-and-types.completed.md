# Feature: Platform Abstraction Interfaces and Core Types

## Priority: CRITICAL (Phase 1 — blocks all command implementations)

## Problem

All 8 CLI commands are stubs that return "not yet implemented". Before any command can be wired up, the project needs:

1. **Platform interfaces** (`internal/platform/platform.go`) — `Reader`, `Inputter`, `WindowManager` interfaces that define the contract all backends must fulfill
2. **Core model types** (`internal/model/`) — `Element`, `Window`, `Bounds`, and related structs with compact JSON tags matching the schema in PLAN.md
3. **Filter types** (`internal/model/filter.go`) — `ReadOptions`, `ListOptions`, `FocusOptions` that capture the CLI flags in a structured way
4. **JSON output utilities** (`internal/output/json.go`) — Serialization with compact mode, pretty-print support, and omit-empty behavior for token efficiency

Without these, every future command implementation will have to invent its own types, leading to inconsistency.

## What to Build

### 1. `internal/platform/platform.go` — Core Interfaces

```go
package platform

import "github.com/mj1618/desktop-cli/internal/model"

// Reader reads the UI element tree from the OS accessibility layer.
type Reader interface {
    // ReadElements returns the element tree for the specified target.
    ReadElements(opts ReadOptions) ([]model.Element, error)

    // ListWindows returns all windows, optionally filtered.
    ListWindows(opts ListOptions) ([]model.Window, error)
}

// Inputter simulates mouse and keyboard input.
type Inputter interface {
    Click(x, y int, button MouseButton, count int) error
    MoveMouse(x, y int) error
    Scroll(x, y int, dx, dy int) error
    Drag(fromX, fromY, toX, toY int) error
    TypeText(text string, delayMs int) error
    KeyCombo(keys []string) error
}

// WindowManager manages window focus and positioning.
type WindowManager interface {
    FocusWindow(opts FocusOptions) error
    GetFrontmostApp() (string, int, error)
}

// MouseButton represents a mouse button.
type MouseButton int

const (
    MouseLeft MouseButton = iota
    MouseRight
    MouseMiddle
)

// ParseMouseButton converts a string flag value to MouseButton.
func ParseMouseButton(s string) (MouseButton, error) { ... }
```

### 2. `internal/platform/types.go` — Option Structs

```go
package platform

// ReadOptions controls what elements to read and how to filter them.
type ReadOptions struct {
    App         string   // Filter by application name
    Window      string   // Filter by window title substring
    WindowID    int      // Filter by system window ID (0 = unset)
    PID         int      // Filter by process ID (0 = unset)
    Depth       int      // Max traversal depth (0 = unlimited)
    Roles       []string // Only include these roles (empty = all)
    VisibleOnly bool     // Only include visible elements
    BBox        *Bounds  // Only include elements within this bounding box (nil = no filter)
    Compact     bool     // Use compact output format
}

// ListOptions controls window/app listing.
type ListOptions struct {
    Apps bool   // List applications instead of windows
    PID  int    // Filter by PID
    App  string // Filter by app name
}

// FocusOptions specifies what to focus.
type FocusOptions struct {
    App      string
    Window   string
    WindowID int
    PID      int
}

// Bounds represents a screen rectangle.
type Bounds struct {
    X, Y, Width, Height int
}

// ParseBBox parses a "x,y,w,h" string into a Bounds.
func ParseBBox(s string) (*Bounds, error) { ... }
```

### 3. `internal/model/element.go` — Element Struct

The Element struct must use compact single-letter JSON keys as defined in PLAN.md, and use `omitempty` to skip default/empty values for token efficiency.

```go
package model

// Element represents a UI element in the accessibility tree.
type Element struct {
    ID          int        `json:"i"`                    // Sequential integer ID
    Role        string     `json:"r"`                    // Abbreviated role code
    Title       string     `json:"t,omitempty"`          // Visible label / title
    Value       string     `json:"v,omitempty"`          // Current value
    Description string     `json:"d,omitempty"`          // Accessibility description
    Bounds      [4]int     `json:"b"`                    // [x, y, width, height]
    Focused     bool       `json:"f,omitempty"`          // Has keyboard focus
    Enabled     *bool      `json:"e,omitempty"`          // nil or true = enabled (omit); false = disabled (include)
    Selected    bool       `json:"s,omitempty"`          // Is selected
    Children    []Element  `json:"c,omitempty"`          // Child elements
    Actions     []string   `json:"a,omitempty"`          // Available actions
}
```

**Note on `Enabled`**: Per PLAN.md, `e` should be omitted when the element IS enabled (the default). Using `*bool` allows distinguishing "not set / enabled" (nil, omitted) from "disabled" (false, included).

### 4. `internal/model/window.go` — Window Struct

```go
package model

// Window represents an application window.
type Window struct {
    App     string `json:"app"`
    PID     int    `json:"pid"`
    Title   string `json:"title"`
    ID      int    `json:"id"`
    Bounds  [4]int `json:"bounds"`
    Focused bool   `json:"focused,omitempty"`
}
```

### 5. `internal/model/roles.go` — Role Mapping

Map macOS accessibility role strings to compact role abbreviations as defined in PLAN.md.

```go
package model

// RoleMap maps macOS AXRole values to compact role codes.
var RoleMap = map[string]string{
    "AXButton":      "btn",
    "AXStaticText":  "txt",
    "AXLink":        "lnk",
    "AXImage":       "img",
    "AXTextField":   "input",
    "AXTextArea":    "input",
    "AXCheckBox":    "chk",
    "AXRadioButton": "radio",
    "AXMenu":        "menu",
    "AXMenuBar":     "menu",
    "AXMenuItem":    "menuitem",
    "AXTabGroup":    "tab",
    "AXList":        "list",
    "AXTable":       "list",
    "AXRow":         "row",
    "AXCell":        "cell",
    "AXGroup":       "group",
    "AXSplitGroup":  "group",
    "AXScrollArea":  "scroll",
    "AXToolbar":     "toolbar",
    "AXWebArea":     "web",
    "AXWindow":      "window",
}

// MapRole converts a raw accessibility role to a compact code.
func MapRole(axRole string) string {
    if short, ok := RoleMap[axRole]; ok {
        return short
    }
    return "other"
}
```

### 6. `internal/model/filter.go` — Element Filtering

```go
package model

import "github.com/mj1618/desktop-cli/internal/platform"

// FilterElements applies ReadOptions filters (roles, bbox, visible-only)
// to a slice of elements, returning only matching elements. Operates on
// the already-built tree; depth filtering should happen during traversal.
func FilterElements(elements []Element, opts platform.ReadOptions) []Element { ... }
```

**Note**: This introduces a circular import risk (`model` importing `platform`). To avoid this, filter options could be defined in `model` or a shared `types` package. Implementer should resolve this — a simple approach is to put `ReadOptions` in `platform` and have `filter.go` accept individual filter params instead of the full `ReadOptions` struct.

### 7. `internal/output/json.go` — JSON Serialization

```go
package output

import "github.com/mj1618/desktop-cli/internal/model"

// ReadResult is the top-level JSON output of the `read` command.
type ReadResult struct {
    App      string          `json:"app,omitempty"`
    PID      int             `json:"pid,omitempty"`
    Window   string          `json:"window,omitempty"`
    TS       int64           `json:"ts"`
    Elements []model.Element `json:"elements"`
}

// PrintJSON serializes v to stdout as JSON.
// If pretty is true, uses indentation; otherwise single-line.
func PrintJSON(v interface{}, pretty bool) error { ... }
```

### 8. Unit Tests

- `internal/model/element_test.go` — Test JSON marshaling: verify short keys, omitempty behavior, Enabled nil vs false
- `internal/model/roles_test.go` — Test MapRole for all known roles plus unknown fallback to "other"
- `internal/platform/types_test.go` — Test ParseBBox and ParseMouseButton
- `internal/output/json_test.go` — Test PrintJSON compact vs pretty output

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `internal/platform/platform.go` defines `Reader`, `Inputter`, `WindowManager` interfaces
- [ ] `internal/platform/types.go` defines `ReadOptions`, `ListOptions`, `FocusOptions`, `Bounds`, `MouseButton`
- [ ] `internal/model/element.go` defines `Element` with compact JSON tags matching PLAN.md schema
- [ ] `internal/model/window.go` defines `Window` struct
- [ ] `internal/model/roles.go` defines role mapping with `MapRole()` function
- [ ] `internal/output/json.go` provides `PrintJSON()` utility
- [ ] JSON marshaling of an Element produces compact keys (`i`, `r`, `t`, `v`, `d`, `b`, `f`, `e`, `s`, `c`, `a`)
- [ ] Omitempty works correctly: empty title omitted, enabled=true omitted, disabled=false included
- [ ] Unit tests cover JSON serialization, role mapping, bbox parsing, mouse button parsing
- [ ] No circular import issues between packages
- [ ] README.md updated if relevant
- [ ] SKILL.md updated if relevant

## Files to Create

- `internal/platform/platform.go` — Interfaces
- `internal/platform/types.go` — Option structs, Bounds, MouseButton
- `internal/platform/types_test.go` — Tests for ParseBBox, ParseMouseButton
- `internal/model/element.go` — Element struct
- `internal/model/element_test.go` — JSON marshaling tests
- `internal/model/window.go` — Window struct
- `internal/model/roles.go` — Role mapping
- `internal/model/roles_test.go` — Role mapping tests
- `internal/model/filter.go` — Element filtering
- `internal/output/json.go` — JSON serialization
- `internal/output/json_test.go` — Output tests

## Notes

- The `Enabled` field on Element needs special handling: PLAN.md says to omit `e` when element IS enabled (the common case). Using `*bool` with `omitempty` achieves this — nil (omitted) means enabled, `*false` (included as `"e":false`) means disabled.
- Keep the `model` and `platform` packages free of circular imports. Filter functions should accept primitive filter params, not the full ReadOptions struct from platform.
- All types should be designed so the macOS darwin backend (future) can populate them directly from CGo results.
- Role abbreviations must exactly match PLAN.md table.

## Completion Notes (Agent b98dbeae)

All acceptance criteria met:

### Files Created
- `internal/platform/platform.go` — `Reader`, `Inputter`, `WindowManager` interfaces
- `internal/platform/types.go` — `ReadOptions`, `ListOptions`, `FocusOptions`, `Bounds`, `MouseButton`, `ParseBBox()`, `ParseMouseButton()`
- `internal/platform/types_test.go` — Tests for ParseBBox (valid, with spaces, invalid) and ParseMouseButton (valid, case-insensitive, invalid)
- `internal/model/element.go` — `Element` struct with compact single-letter JSON keys (`i`, `r`, `t`, `v`, `d`, `b`, `f`, `e`, `s`, `c`, `a`) and omitempty
- `internal/model/element_test.go` — Tests for JSON keys, omitempty behavior, Enabled nil vs false, children, round-trip
- `internal/model/window.go` — `Window` struct with JSON tags
- `internal/model/roles.go` — `RoleMap` and `MapRole()` for all 22 known AX roles
- `internal/model/roles_test.go` — Tests for all known roles and unknown fallback to "other"
- `internal/model/filter.go` — `FilterElements()` with role and bbox filtering (recursive), avoids circular imports by accepting primitive params
- `internal/model/filter_test.go` — Tests for no filters, role filter, bbox filter, recursive children, bounds intersection
- `internal/output/json.go` — `ReadResult` struct and `PrintJSON()` (compact/pretty modes, no HTML escaping)
- `internal/output/json_test.go` — Tests for compact output, pretty output, omitempty on ReadResult

### Design Decisions
- Circular import avoided: `filter.go` accepts `[]string` roles and `*[4]int` bbox instead of `platform.ReadOptions`
- `Enabled` uses `*bool` with omitempty: nil (omitted) = enabled, &false (included) = disabled
- `boundsIntersect` is an unexported helper in the model package for bbox filtering
- `PrintJSON` uses `json.Encoder` with `SetEscapeHTML(false)` to avoid escaping `<`, `>`, `&` in element text

### Verification
- `go build ./...` — passes
- `go test ./...` — all 28 tests pass across 5 packages
- No circular imports between `model`, `platform`, and `output` packages
- README.md and SKILL.md already document the JSON schema and role abbreviations — no changes needed
