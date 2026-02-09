# Feature: Implement `set-value` Command for Direct Attribute Value Setting

## Priority: HIGH (Phase 4 — direct AX value setting for reliable element manipulation)

## Problem

Agents using `desktop-cli` can see element values (via the `v` field in `read` output — e.g. text field contents, slider positions, checkbox states), but there's no way to **set** those values directly. Currently, to fill a text field, agents must:

1. `click --id <id>` to focus the element
2. `type --key "cmd+a"` to select all existing text
3. `type --text "new value"` to type the new text character by character

This keystroke-simulation approach has significant limitations:

1. **Slow for long text** — Typing 500 characters takes 500 keystroke events. Setting the value directly is instant.
2. **Unreliable for rich text fields** — Key combos like Cmd+A may not work in all apps or field types.
3. **Cannot set non-text values** — Sliders, steppers, progress indicators, and other numeric value elements can't be set by typing.
4. **Race conditions** — Between clicking to focus and typing, another element might steal focus.
5. **No way to set checkbox/toggle state** — The `action press` command toggles a checkbox, but agents can't set it to a specific state (on/off) without first reading the current state.

The macOS Accessibility API provides `AXUIElementSetAttributeValue()` which sets attribute values directly on the AX element reference. For `kAXValueAttribute`, this directly updates text field contents, slider positions, and similar value-holding elements. For `kAXSelectedAttribute` and `kAXFocusedAttribute`, it can set selection and focus state.

## What to Build

### 1. Add ValueSetter Interface — `internal/platform/platform.go`

Add a new interface for setting accessibility attribute values on elements:

```go
// ValueSetter sets accessibility attribute values directly on UI elements.
type ValueSetter interface {
    // SetValue sets the value attribute on an element identified
    // by its sequential ID within the given read scope.
    SetValue(opts SetValueOptions) error
}
```

### 2. Add SetValueOptions — `internal/platform/types.go`

```go
// SetValueOptions configures which element to set a value on and what value to set.
type SetValueOptions struct {
    App      string // Scope to application
    Window   string // Scope to window
    WindowID int    // Scope to window by system ID
    PID      int    // Scope to process
    ID       int    // Element ID (from read output)
    Value    string // The value to set (text for text fields, number for sliders, "true"/"false" for checkboxes)
    Attribute string // AX attribute to set (default: "value" → kAXValueAttribute)
}
```

### 3. Add ValueSetter to Provider — `internal/platform/provider.go`

Add the `ValueSetter` field to the `Provider` struct:

```go
type Provider struct {
    Reader          Reader
    Inputter        Inputter
    WindowManager   WindowManager
    Screenshotter   Screenshotter
    ActionPerformer ActionPerformer
    ValueSetter     ValueSetter
}
```

### 4. C Implementation — `internal/platform/darwin/set_value.h` + `set_value.c`

**`set_value.h`:**
```c
#ifndef SET_VALUE_H
#define SET_VALUE_H

#include <ApplicationServices/ApplicationServices.h>

// Set a string value on the element at the given traversal index.
// attributeName is the AX attribute (e.g. "AXValue", "AXSelected", "AXFocused").
// Returns 0 on success, -1 on failure.
int ax_set_value(pid_t pid, const char* windowTitle, int windowID,
                 int elementIndex, const char* attributeName, const char* value);

#endif
```

**`set_value.c` implementation approach:**

1. Get the app's AX reference: `AXUIElementCreateApplication(pid)`
2. Get the target window (matching the same logic as `ax_read_elements` — by windowTitle substring or windowID)
3. Traverse the element tree in the **exact same order** as `ax_read_elements` to find the element at the given sequential index (ID)
4. Determine the value type needed:
   - For `kAXValueAttribute` on text fields: create a `CFStringRef` from the value string
   - For `kAXValueAttribute` on sliders/steppers: parse the string as a number and create a `CFNumberRef`
   - For `kAXSelectedAttribute` / `kAXFocusedAttribute`: parse "true"/"false" and create a `kCFBooleanTrue`/`kCFBooleanFalse`
5. Call `AXUIElementSetAttributeValue(element, attribute, cfValue)`
6. Return success/failure

**Critical: Type detection.** The C code should check the element's `kAXRoleAttribute` to determine the correct `CFTypeRef` to create:
- `AXTextField`, `AXTextArea`, `AXComboBox` → `CFStringRef`
- `AXSlider`, `AXStepper`, `AXProgressIndicator` → `CFNumberRef` (float)
- `AXCheckBox`, `AXRadioButton` → For the "AXValue" attribute, these use `CFNumberRef` with 0/1
- Default → Try `CFStringRef` first

**Alternatively (simpler):** Query the element's `kAXValueAttribute` type using `AXUIElementCopyAttributeValue()` first, then create the new value with the same `CFTypeID`. This auto-detects the correct type.

**Traversal reuse.** The traversal function should be extracted from `accessibility.c` into a shared helper that both `ax_read_elements` and `ax_set_value` (and the future `ax_perform_action`) can use. However, to minimize scope of this task, the traversal can be duplicated initially and refactored later. The key requirement is that traversal order is **identical** to `ax_read_elements` so element IDs match between `read` and `set-value`.

### 5. Go Wrapper — `internal/platform/darwin/value_setter.go`

```go
//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework Foundation
#include "set_value.h"
#include <stdlib.h>
*/
import "C"

import (
    "fmt"
    "unsafe"

    "github.com/mj1618/desktop-cli/internal/platform"
)

type DarwinValueSetter struct {
    reader *DarwinReader
}

func NewValueSetter(reader *DarwinReader) *DarwinValueSetter {
    return &DarwinValueSetter{reader: reader}
}

func (s *DarwinValueSetter) SetValue(opts platform.SetValueOptions) error {
    if opts.ID <= 0 {
        return fmt.Errorf("--id is required")
    }
    if opts.Value == "" {
        return fmt.Errorf("--value is required")
    }

    // Resolve the target window (same pattern as click/type/scroll commands)
    windows, err := s.reader.ListWindows(platform.ListOptions{
        App: opts.App,
        PID: opts.PID,
    })
    if err != nil {
        return fmt.Errorf("failed to list windows: %w", err)
    }
    if len(windows) == 0 {
        return fmt.Errorf("no matching window found")
    }

    win := windows[0]

    // Determine AX attribute name (default: AXValue)
    attribute := "AXValue"
    switch opts.Attribute {
    case "", "value":
        attribute = "AXValue"
    case "selected":
        attribute = "AXSelected"
    case "focused":
        attribute = "AXFocused"
    default:
        // Allow passing raw AX attribute names like "AXValue" directly
        attribute = opts.Attribute
    }

    cWindowTitle := (*C.char)(nil)
    if opts.Window != "" {
        cWindowTitle = C.CString(opts.Window)
        defer C.free(unsafe.Pointer(cWindowTitle))
    }

    cAttribute := C.CString(attribute)
    defer C.free(unsafe.Pointer(cAttribute))

    cValue := C.CString(opts.Value)
    defer C.free(unsafe.Pointer(cValue))

    rc := C.ax_set_value(C.pid_t(win.PID), cWindowTitle, C.int(opts.WindowID),
        C.int(opts.ID), cAttribute, cValue)
    if rc != 0 {
        return fmt.Errorf("failed to set %s=%q on element %d", attribute, opts.Value, opts.ID)
    }

    return nil
}
```

### 6. Command Definition — `cmd/setvalue.go` (new file)

```go
package cmd

import (
    "fmt"

    "github.com/mj1618/desktop-cli/internal/output"
    "github.com/mj1618/desktop-cli/internal/platform"
    "github.com/spf13/cobra"
)

type SetValueResult struct {
    OK        bool   `yaml:"ok"`
    Action    string `yaml:"action"`
    ID        int    `yaml:"id"`
    Value     string `yaml:"value"`
    Attribute string `yaml:"attribute"`
}

var setValueCmd = &cobra.Command{
    Use:   "set-value",
    Short: "Set the value of a UI element directly",
    Long: `Set an accessibility attribute value directly on a UI element by ID.

This sets the element's value without simulating keystrokes or mouse events.
Common use cases:
  - Set text field contents instantly (faster than type for long text)
  - Set slider positions to specific values
  - Set checkbox/toggle state without toggling

The --attribute flag controls which attribute to set:
  value     - Element value: text content, slider position, etc. (default)
  selected  - Selection state (true/false)
  focused   - Focus state (true/false)

Unlike 'type', this does NOT simulate keystrokes — it sets the value directly
via the accessibility API, which is faster and more reliable.`,
    RunE: runSetValue,
}

func init() {
    rootCmd.AddCommand(setValueCmd)
    setValueCmd.Flags().Int("id", 0, "Element ID from read output (required)")
    setValueCmd.Flags().String("value", "", "Value to set (required)")
    setValueCmd.Flags().String("attribute", "value", "Attribute to set: value (default), selected, focused")
    setValueCmd.Flags().String("app", "", "Scope to application")
    setValueCmd.Flags().String("window", "", "Scope to window")
    setValueCmd.Flags().Int("window-id", 0, "Scope to window by system ID")
    setValueCmd.Flags().Int("pid", 0, "Scope to process by PID")
    setValueCmd.MarkFlagRequired("id")
    setValueCmd.MarkFlagRequired("value")
}

func runSetValue(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.ValueSetter == nil {
        return fmt.Errorf("set-value not supported on this platform")
    }

    id, _ := cmd.Flags().GetInt("id")
    value, _ := cmd.Flags().GetString("value")
    attribute, _ := cmd.Flags().GetString("attribute")
    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")
    windowID, _ := cmd.Flags().GetInt("window-id")
    pid, _ := cmd.Flags().GetInt("pid")

    if appName == "" && window == "" && windowID == 0 && pid == 0 {
        return fmt.Errorf("--app, --window, --window-id, or --pid is required to scope the element lookup")
    }

    opts := platform.SetValueOptions{
        App:       appName,
        Window:    window,
        WindowID:  windowID,
        PID:       pid,
        ID:        id,
        Value:     value,
        Attribute: attribute,
    }

    err = provider.ValueSetter.SetValue(opts)
    if err != nil {
        return err
    }

    return output.PrintYAML(SetValueResult{
        OK:        true,
        Action:    "set-value",
        ID:        id,
        Value:     value,
        Attribute: attribute,
    })
}
```

### 7. Register ValueSetter — `internal/platform/darwin/init.go`

Update the init function to also create and register the ValueSetter:

```go
func init() {
    platform.NewProviderFunc = func() (*platform.Provider, error) {
        reader := NewReader()
        inputter := NewInputter()
        windowManager := NewWindowManager(reader)
        screenshotter := NewScreenshotter(reader)
        valueSetter := NewValueSetter(reader)
        return &platform.Provider{
            Reader:        reader,
            Inputter:      inputter,
            WindowManager: windowManager,
            Screenshotter: screenshotter,
            ValueSetter:   valueSetter,
        }, nil
    }
}
```

Note: If the `action` command task has already added `ActionPerformer` to init.go by the time this is implemented, include it too.

### 8. Update Documentation

**README.md** — Add a "Set element values" section after "Type text or key combos":

```markdown
### Set element values

```bash
# Set a text field's value directly (instant, no keystroke simulation)
desktop-cli set-value --id 4 --value "hello world" --app "Safari"

# Set a slider to a specific position
desktop-cli set-value --id 12 --value "75" --app "System Settings"

# Clear a text field
desktop-cli set-value --id 4 --value "" --app "Safari"

# Set focus on an element
desktop-cli set-value --id 4 --attribute focused --value "true" --app "Safari"

# Set selection state
desktop-cli set-value --id 8 --attribute selected --value "true" --app "Finder"
```
```

**SKILL.md** — Add set-value to quick reference:

```markdown
### Set element values

```bash
desktop-cli set-value --id 4 --value "hello world" --app "Safari"
desktop-cli set-value --id 12 --value "75" --app "System Settings"
desktop-cli set-value --id 4 --value "" --app "Safari"
desktop-cli set-value --id 4 --attribute focused --value "true" --app "Safari"
```
```

Update the "Agent Workflow" section in both files to mention `set-value`:

```markdown
## Agent Workflow

1. `list --windows` to find the target window
2. `read --app <name> --depth 3 --roles "btn,lnk,input,txt"` to get the element tree as YAML
3. Use the element `i` (id) field with:
   - `set-value --id <id> --value "..." --app <name>` to set text fields, sliders, etc. (preferred for value-holding elements)
   - `action --id <id> --app <name>` to press buttons, toggle checkboxes (if action command exists)
   - `click --id <id> --app <name>` to click at coordinates (fallback)
   - `type --id <id> --app <name> --text "..."` to type into fields (fallback for fields that don't support set-value)
4. `wait --app <name> --for-text "..." --timeout 10` to wait for UI to update
5. Repeat read/act/wait loop as needed
```

## Files to Create

- `internal/platform/darwin/set_value.c` — C implementation using `AXUIElementSetAttributeValue`
- `internal/platform/darwin/set_value.h` — C header
- `internal/platform/darwin/value_setter.go` — Go `DarwinValueSetter` implementing `platform.ValueSetter`
- `cmd/setvalue.go` — New `set-value` command

## Files to Modify

- `internal/platform/platform.go` — Add `ValueSetter` interface
- `internal/platform/types.go` — Add `SetValueOptions` struct
- `internal/platform/provider.go` — Add `ValueSetter` field to `Provider` struct
- `internal/platform/darwin/init.go` — Register `ValueSetter` in provider
- `README.md` — Add set-value command usage section and update agent workflow
- `SKILL.md` — Add set-value command to quick reference and update agent workflow

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (ValueSetter is nil in provider, command returns helpful error)
- [ ] `go test ./...` passes
- [ ] `desktop-cli set-value --help` shows all flags with descriptions
- [ ] `desktop-cli set-value --id 4 --value "test" --app "TextEdit"` sets a text field's value
- [ ] `desktop-cli set-value --id 12 --value "50" --app "System Settings"` sets a slider value
- [ ] `desktop-cli set-value --id 4 --value "" --app "TextEdit"` clears a text field
- [ ] `desktop-cli set-value --id 4 --attribute focused --value "true" --app "TextEdit"` focuses an element
- [ ] `desktop-cli set-value` with no `--id` returns a clear error message
- [ ] `desktop-cli set-value` with no `--value` returns a clear error message
- [ ] `desktop-cli set-value` with no app scope returns a clear error message
- [ ] Invalid element ID returns error: "failed to set ... on element ..."
- [ ] Setting a read-only attribute returns an error from AX API
- [ ] On success: outputs YAML with `ok: true`, attribute name, value, and element ID
- [ ] Element IDs are consistent between `read` and `set-value` (same traversal order)
- [ ] README.md and SKILL.md are updated with set-value command examples and workflow

## Implementation Notes

- **`AXUIElementSetAttributeValue`** is the core macOS API. It takes an `AXUIElementRef`, a `CFStringRef` attribute name, and a `CFTypeRef` value. For text fields, the value is a `CFStringRef`. For numeric elements (sliders), it's a `CFNumberRef`. For boolean attributes (selected, focused), it's `kCFBooleanTrue`/`kCFBooleanFalse`.

- **Type detection strategy:** The C code should query the current value of the attribute using `AXUIElementCopyAttributeValue()` to determine the `CFTypeID`, then create the new value with the matching type. This avoids hardcoding role-to-type mappings. Fallback: try `CFStringRef`, then `CFNumberRef`.

- **Element ID consistency is critical**: Same requirement as the `action` command — the traversal order must match `ax_read_elements`. Reuse the same window resolution logic (`windowTitle`, `windowID`) and traversal from `accessibility.c`.

- **When to use `set-value` vs `type`**: `set-value` is instant and reliable for standard AX-compliant elements. However, some custom-drawn UI or web content may not support `AXUIElementSetAttributeValue` (the `kAXValueAttribute` might be read-only). In those cases, `type` with keystroke simulation is the fallback. Agents should try `set-value` first and fall back to `type` on error.

- **Permission**: Uses the same Accessibility permission as all other commands. No additional permission needed.

- **Memory management**: `CFTypeRef` values created for setting must be properly released after the call. The traversal function's `AXUIElementRef` must also be released.

- **Error handling**: `AXUIElementSetAttributeValue` returns `AXError`. Common errors:
  - `kAXErrorAttributeUnsupported` — element doesn't support setting this attribute
  - `kAXErrorInvalidUIElement` — element reference is stale
  - `kAXErrorCannotComplete` — app is busy or unresponsive
  - `kAXErrorIllegalArgument` — wrong value type for the attribute
  Map these to clear error messages for agents.

- **Empty value**: Setting `--value ""` should work for clearing text fields. The C code should handle empty strings as valid `CFStringRef` values.
