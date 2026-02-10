# Feature: Add `observe` Command for Streaming UI Change Detection

## Priority: HIGH (Phase 4 — enables reactive agent workflows with minimal token usage)

## Problem

Agents frequently need to monitor a UI for changes after performing an action. The current workflow is:

1. `click --id 5 --app "Safari"` to perform an action
2. `wait --for-text "Done" --app "Safari"` to wait for a known condition, OR
3. Repeatedly call `read --app "Safari"` in a polling loop, parsing the full tree each time

This has significant problems for open-ended monitoring:

1. **`wait` only checks for predetermined conditions** — The agent must know exactly what to look for upfront. It can't discover unexpected changes (error dialogs, loading spinners, redirects, new elements appearing).
2. **Repeated `read` is token-expensive** — A typical browser page produces 200+ elements. Re-reading the full tree every 500ms for 10 seconds = 20 full reads = ~40,000 tokens, even if only 2 elements changed.
3. **No change isolation** — Even with `--text` or `--flat` filters, the agent must diff the current output against the previous output to identify what changed. This wastes tokens and agent compute.
4. **No streaming** — Each `read` call is a separate CLI invocation, adding process startup overhead.

The PLAN.md Phase 4 mentions "Add `--watch` mode to `read` (poll and output changes as JSONL)." This feature implements that vision with a focus on **diff-based output** — only emitting what changed between polls, not the full tree.

## What to Build

### 1. New `observe` Command — `cmd/observe.go`

A command that continuously polls the UI tree and emits only the changes (additions, removals, modifications) as a JSONL stream to stdout.

```bash
# Watch Safari for any UI changes, emit diffs
desktop-cli observe --app "Safari"

# Watch for changes to buttons and links only
desktop-cli observe --app "Safari" --roles "btn,lnk"

# Watch a specific window with custom polling interval
desktop-cli observe --app "Chrome" --window "Gmail" --interval 500

# Watch for a limited duration
desktop-cli observe --app "Safari" --duration 10

# Watch at limited depth
desktop-cli observe --app "Safari" --depth 3
```

### 2. Output Format — JSONL (one JSON object per line)

Each line represents an event. Events are emitted **only when changes are detected** — no output when the UI is stable.

#### Event Types

**`snapshot` — Initial state (emitted once at start):**
```json
{"type":"snapshot","ts":1707500000,"count":42}
```
This tells the agent how many elements exist initially, without dumping the full tree. The agent can `read` if it needs the full tree.

**`added` — New element appeared:**
```json
{"type":"added","ts":1707500001,"el":{"i":43,"r":"btn","t":"Submit","b":[400,300,80,32],"a":["press"]},"p":"window > web > form"}
```

**`removed` — Element disappeared:**
```json
{"type":"removed","ts":1707500002,"id":15,"r":"btn","t":"Loading..."}
```

**`changed` — Element's properties changed:**
```json
{"type":"changed","ts":1707500003,"id":5,"changes":{"v":["old value","new value"]}}
```
The `changes` field is a map of field name to `[old, new]` pairs. Only changed fields are included.

**`done` — Observation ended (duration reached or interrupted):**
```json
{"type":"done","ts":1707500010,"elapsed":"10.0s","events":7}
```

#### Why JSONL (not YAML)

JSONL is the standard format for streaming structured data:
- One event per line — easy to parse incrementally (`for line in stream`)
- Compatible with `jq` for real-time filtering: `desktop-cli observe --app "Safari" | jq 'select(.type=="added")'`
- Agents can process events as they arrive without buffering
- YAML is multi-line and unsuitable for streaming

This command always outputs JSONL regardless of the `--format` flag (YAML cannot stream line-by-line).

### 3. Diff Algorithm — `internal/model/diff.go` (new file)

The diff logic compares two element trees and produces a list of changes.

```go
package model

// ChangeType represents the kind of UI change detected.
type ChangeType string

const (
    ChangeAdded   ChangeType = "added"
    ChangeRemoved ChangeType = "removed"
    ChangeChanged ChangeType = "changed"
)

// UIChange represents a single change between two reads.
type UIChange struct {
    Type    ChangeType        `json:"type"`
    TS      int64             `json:"ts"`
    // For added: the full element
    Element *FlatElement      `json:"el,omitempty"`
    // For added: path in tree
    Path    string            `json:"p,omitempty"`
    // For removed: element identifiers
    ID      int               `json:"id,omitempty"`
    Role    string            `json:"r,omitempty"`
    Title   string            `json:"t,omitempty"`
    // For changed: field diffs
    Changes map[string][2]string `json:"changes,omitempty"`
}

// DiffElements compares two flat element lists and returns the changes.
// Elements are matched by their ID (sequential traversal index).
// prev and curr should be flattened from the same app/window scope.
func DiffElements(prev, curr []FlatElement) []UIChange {
    prevMap := make(map[int]FlatElement, len(prev))
    for _, el := range prev {
        prevMap[el.ID] = el
    }
    currMap := make(map[int]FlatElement, len(curr))
    for _, el := range curr {
        currMap[el.ID] = el
    }

    var changes []UIChange
    now := time.Now().Unix()

    // Check for added and changed elements
    for _, el := range curr {
        prevEl, existed := prevMap[el.ID]
        if !existed {
            changes = append(changes, UIChange{
                Type:    ChangeAdded,
                TS:      now,
                Element: &el,
                Path:    el.Path,
            })
            continue
        }
        // Check for property changes
        diffs := diffProperties(prevEl, el)
        if len(diffs) > 0 {
            changes = append(changes, UIChange{
                Type:    ChangeChanged,
                TS:      now,
                ID:      el.ID,
                Changes: diffs,
            })
        }
    }

    // Check for removed elements
    for _, el := range prev {
        if _, exists := currMap[el.ID]; !exists {
            changes = append(changes, UIChange{
                Type:  ChangeRemoved,
                TS:    now,
                ID:    el.ID,
                Role:  el.Role,
                Title: el.Title,
            })
        }
    }

    return changes
}

// diffProperties compares two elements and returns changed fields.
// Returns a map of field name to [old, new] string values.
func diffProperties(prev, curr FlatElement) map[string][2]string {
    diffs := make(map[string][2]string)

    if prev.Title != curr.Title {
        diffs["t"] = [2]string{prev.Title, curr.Title}
    }
    if prev.Value != curr.Value {
        diffs["v"] = [2]string{prev.Value, curr.Value}
    }
    if prev.Role != curr.Role {
        diffs["r"] = [2]string{prev.Role, curr.Role}
    }
    if prev.Description != curr.Description {
        diffs["d"] = [2]string{prev.Description, curr.Description}
    }
    if prev.Bounds != curr.Bounds {
        diffs["b"] = [2]string{
            fmt.Sprintf("%v", prev.Bounds),
            fmt.Sprintf("%v", curr.Bounds),
        }
    }
    if prev.Focused != curr.Focused {
        diffs["f"] = [2]string{
            fmt.Sprintf("%v", prev.Focused),
            fmt.Sprintf("%v", curr.Focused),
        }
    }
    if prev.Selected != curr.Selected {
        diffs["s"] = [2]string{
            fmt.Sprintf("%v", prev.Selected),
            fmt.Sprintf("%v", curr.Selected),
        }
    }

    if len(diffs) == 0 {
        return nil
    }
    return diffs
}
```

**Matching strategy:** Elements are matched by ID (traversal index). This is imperfect when the UI structure changes significantly (IDs shift), but it's good enough for monitoring incremental changes (which is the primary use case). When the entire tree restructures, many adds/removes will be emitted, which correctly signals a major UI change to the agent.

### 4. Command Implementation — `cmd/observe.go`

```go
package cmd

import (
    "encoding/json"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/mj1618/desktop-cli/internal/model"
    "github.com/mj1618/desktop-cli/internal/platform"
    "github.com/spf13/cobra"
)

var observeCmd = &cobra.Command{
    Use:   "observe",
    Short: "Watch for UI changes and stream diffs as JSONL",
    Long: `Continuously poll the UI element tree and emit changes (added, removed, modified elements) as JSONL to stdout.

Each line is a JSON object representing one change event. No output is emitted when the UI is stable.
This is far more token-efficient than repeatedly calling 'read' — only changes are reported.

Output is always JSONL regardless of the --format flag.

Use Ctrl+C or --duration to stop observing.`,
    RunE: runObserve,
}

func init() {
    rootCmd.AddCommand(observeCmd)
    observeCmd.Flags().String("app", "", "Scope to application (required)")
    observeCmd.Flags().String("window", "", "Scope to window by title substring")
    observeCmd.Flags().Int("window-id", 0, "Scope to window by system ID")
    observeCmd.Flags().Int("pid", 0, "Scope to process by PID")
    observeCmd.Flags().Int("depth", 0, "Max depth to traverse (0 = unlimited)")
    observeCmd.Flags().String("roles", "", "Comma-separated roles to include (e.g. \"btn,input\")")
    observeCmd.Flags().Int("interval", 1000, "Polling interval in milliseconds (default: 1000)")
    observeCmd.Flags().Int("duration", 0, "Max seconds to observe (0 = until Ctrl+C, default: 0)")
    observeCmd.Flags().Bool("ignore-bounds", false, "Ignore element position changes (reduces noise from layout shifts)")
    observeCmd.Flags().Bool("ignore-focus", false, "Ignore focus changes (reduces noise from focus cycling)")
}

func runObserve(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.Reader == nil {
        return fmt.Errorf("reader not available on this platform")
    }

    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")
    windowID, _ := cmd.Flags().GetInt("window-id")
    pid, _ := cmd.Flags().GetInt("pid")
    depth, _ := cmd.Flags().GetInt("depth")
    rolesStr, _ := cmd.Flags().GetString("roles")
    intervalMs, _ := cmd.Flags().GetInt("interval")
    durationSec, _ := cmd.Flags().GetInt("duration")
    ignoreBounds, _ := cmd.Flags().GetBool("ignore-bounds")
    ignoreFocus, _ := cmd.Flags().GetBool("ignore-focus")

    if appName == "" && pid == 0 && windowID == 0 {
        return fmt.Errorf("--app, --pid, or --window-id is required to scope observation")
    }

    var roles []string
    if rolesStr != "" {
        for _, r := range strings.Split(rolesStr, ",") {
            roles = append(roles, strings.TrimSpace(r))
        }
    }

    readOpts := platform.ReadOptions{
        App:      appName,
        Window:   window,
        WindowID: windowID,
        PID:      pid,
        Depth:    depth,
        Roles:    roles,
    }

    enc := json.NewEncoder(os.Stdout)
    enc.SetEscapeHTML(false)

    interval := time.Duration(intervalMs) * time.Millisecond
    var deadline time.Time
    if durationSec > 0 {
        deadline = time.Now().Add(time.Duration(durationSec) * time.Second)
    }
    start := time.Now()

    // Initial read to establish baseline
    elements, err := provider.Reader.ReadElements(readOpts)
    if err != nil {
        return fmt.Errorf("initial read failed: %w", err)
    }
    prevFlat := model.FlattenElements(elements)

    // Emit snapshot event
    enc.Encode(map[string]interface{}{
        "type":  "snapshot",
        "ts":    time.Now().Unix(),
        "count": len(prevFlat),
    })

    eventCount := 0

    // Poll loop
    for {
        if durationSec > 0 && time.Now().After(deadline) {
            break
        }

        time.Sleep(interval)

        elements, err := provider.Reader.ReadElements(readOpts)
        if err != nil {
            // Emit error event but continue observing
            enc.Encode(map[string]interface{}{
                "type":  "error",
                "ts":    time.Now().Unix(),
                "error": err.Error(),
            })
            continue
        }

        currFlat := model.FlattenElements(elements)
        changes := model.DiffElements(prevFlat, currFlat)

        // Filter out ignored change types
        for _, change := range changes {
            if change.Type == model.ChangeChanged {
                if ignoreBounds {
                    delete(change.Changes, "b")
                }
                if ignoreFocus {
                    delete(change.Changes, "f")
                }
                if len(change.Changes) == 0 {
                    continue
                }
            }
            enc.Encode(change)
            eventCount++
        }

        prevFlat = currFlat
    }

    // Emit done event
    elapsed := time.Since(start)
    enc.Encode(map[string]interface{}{
        "type":    "done",
        "ts":      time.Now().Unix(),
        "elapsed": fmt.Sprintf("%.1fs", elapsed.Seconds()),
        "events":  eventCount,
    })

    return nil
}
```

### 5. Diff Logic Unit Tests — `internal/model/diff_test.go` (new file)

```go
package model

import "testing"

func TestDiffElements_NoChanges(t *testing.T) {
    elements := []FlatElement{
        {Element: Element{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}}, Path: "window"},
    }
    changes := DiffElements(elements, elements)
    if len(changes) != 0 {
        t.Errorf("expected no changes, got %d", len(changes))
    }
}

func TestDiffElements_Added(t *testing.T) {
    prev := []FlatElement{
        {Element: Element{ID: 1, Role: "btn", Title: "OK"}, Path: "window"},
    }
    curr := []FlatElement{
        {Element: Element{ID: 1, Role: "btn", Title: "OK"}, Path: "window"},
        {Element: Element{ID: 2, Role: "btn", Title: "Cancel"}, Path: "window"},
    }
    changes := DiffElements(prev, curr)
    if len(changes) != 1 {
        t.Fatalf("expected 1 change, got %d", len(changes))
    }
    if changes[0].Type != ChangeAdded {
        t.Errorf("expected added, got %s", changes[0].Type)
    }
    if changes[0].Element.Title != "Cancel" {
        t.Errorf("expected Cancel, got %s", changes[0].Element.Title)
    }
}

func TestDiffElements_Removed(t *testing.T) {
    prev := []FlatElement{
        {Element: Element{ID: 1, Role: "btn", Title: "OK"}, Path: "window"},
        {Element: Element{ID: 2, Role: "btn", Title: "Loading..."}, Path: "window"},
    }
    curr := []FlatElement{
        {Element: Element{ID: 1, Role: "btn", Title: "OK"}, Path: "window"},
    }
    changes := DiffElements(prev, curr)
    if len(changes) != 1 {
        t.Fatalf("expected 1 change, got %d", len(changes))
    }
    if changes[0].Type != ChangeRemoved {
        t.Errorf("expected removed, got %s", changes[0].Type)
    }
    if changes[0].ID != 2 {
        t.Errorf("expected ID 2, got %d", changes[0].ID)
    }
}

func TestDiffElements_Changed(t *testing.T) {
    prev := []FlatElement{
        {Element: Element{ID: 1, Role: "input", Title: "Search", Value: ""}, Path: "window"},
    }
    curr := []FlatElement{
        {Element: Element{ID: 1, Role: "input", Title: "Search", Value: "hello"}, Path: "window"},
    }
    changes := DiffElements(prev, curr)
    if len(changes) != 1 {
        t.Fatalf("expected 1 change, got %d", len(changes))
    }
    if changes[0].Type != ChangeChanged {
        t.Errorf("expected changed, got %s", changes[0].Type)
    }
    if changes[0].Changes["v"][1] != "hello" {
        t.Errorf("expected new value 'hello', got %s", changes[0].Changes["v"][1])
    }
}

func TestDiffProperties_MultipleDiffs(t *testing.T) {
    prev := FlatElement{Element: Element{ID: 1, Role: "input", Title: "Name", Value: "old", Focused: false}}
    curr := FlatElement{Element: Element{ID: 1, Role: "input", Title: "Name", Value: "new", Focused: true}}
    diffs := diffProperties(prev, curr)
    if len(diffs) != 2 {
        t.Errorf("expected 2 diffs (v, f), got %d", len(diffs))
    }
    if diffs["v"][0] != "old" || diffs["v"][1] != "new" {
        t.Errorf("unexpected value diff: %v", diffs["v"])
    }
}
```

## Files to Create

- `internal/model/diff.go` — `UIChange` struct and `DiffElements`/`diffProperties` functions
- `internal/model/diff_test.go` — Unit tests for diff logic
- `cmd/observe.go` — New `observe` command

## Files to Modify

- `README.md` — Add "Observe UI changes" section with usage examples
- `SKILL.md` — Add `observe` to quick reference and agent workflow

## Dependencies

- None — uses existing `read` infrastructure and standard library `encoding/json`

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `desktop-cli observe --help` shows all flags with descriptions
- [ ] `desktop-cli observe --app "Finder"` starts observing and emits a `snapshot` event
- [ ] When a UI element appears, an `added` event is emitted with the element's properties and path
- [ ] When a UI element disappears, a `removed` event is emitted with the element's ID, role, and title
- [ ] When an element's value/title/state changes, a `changed` event is emitted with only the changed fields
- [ ] No output is emitted when the UI is stable (no spurious events)
- [ ] `--interval 200` increases polling frequency to 200ms
- [ ] `--duration 5` stops observing after 5 seconds and emits a `done` event
- [ ] `--roles "btn,input"` only monitors buttons and input fields
- [ ] `--depth 3` limits traversal depth
- [ ] `--ignore-bounds` suppresses position-change events
- [ ] `--ignore-focus` suppresses focus-change events
- [ ] Output is JSONL (one JSON object per line), regardless of `--format` flag
- [ ] Each event line is valid JSON parseable by `jq`
- [ ] Error events are emitted for transient read failures without stopping observation
- [ ] `--app`, `--pid`, or `--window-id` is required (clear error if none given)
- [ ] `DiffElements` has unit tests: no changes, added, removed, changed, multiple diffs
- [ ] `diffProperties` correctly compares all element fields: title, value, role, description, bounds, focused, selected
- [ ] README.md documents `observe` command with examples
- [ ] SKILL.md includes `observe` in quick reference and agent workflow

## Implementation Notes

- **Always JSONL output.** YAML is not suitable for streaming — it's multi-line and requires document separators. The `observe` command ignores the `--format` flag and always outputs JSONL. This is documented in the `--help` text.

- **Element matching by ID.** The diff algorithm matches elements by their traversal-order ID. This works well for incremental changes (buttons appearing, text changing, focus moving) but can produce many add/remove events when the tree structure significantly changes (e.g., page navigation). This is correct behavior — a major tree restructure IS a significant change that agents should know about.

- **Reuses FlattenElements.** The diff operates on flattened element lists (not trees). This simplifies comparison — flat lists are easy to index by ID and compare element-by-element. The `FlattenElements` function from `internal/model/flatten.go` is reused.

- **No Ctrl+C handling needed.** When `--duration` is 0 (observe forever), the user sends SIGINT (Ctrl+C) to stop. Go's default signal handling exits cleanly. The `done` event won't be emitted in this case, but that's fine — agents that use `observe` with a duration will get the `done` event.

- **Polling vs. AX notifications.** macOS provides `AXObserverCreate` for real-time AX notifications. This would be more efficient than polling but adds significant complexity (callback management, run loop integration). Polling is simple, portable, and sufficient. The `--interval` flag lets agents trade off latency vs. CPU. Default 1000ms is conservative.

- **Token efficiency.** For a typical interaction: agent clicks a button, UI shows a dialog with 5 new elements. With `observe`, the agent receives ~5 lines of JSONL (~250 tokens). With repeated `read`, the agent would receive the full 200-element tree each time (~2000 tokens per read). For a 10-second observation at 500ms intervals, that's 250 vs 40,000 tokens — a **160x reduction**.

- **Error resilience.** If a single read fails (e.g., app is temporarily unresponsive), the command emits an `error` event and continues polling. It does NOT exit or lose the previous baseline state.

- **Noise reduction flags.** `--ignore-bounds` and `--ignore-focus` help reduce noisy events. Layout shifts (bounds changes) and focus cycling are common during normal interaction and often irrelevant to the agent's goal. These flags let agents opt out of these change categories.
