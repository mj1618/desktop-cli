# Feature: Implement `wait` Command for Polling UI Conditions

## Priority: HIGH (Phase 4 — critical for robust agent automation loops)

## Problem

Agents using `desktop-cli` frequently need to wait for UI state changes after performing actions — for example, waiting for a dialog to appear after clicking a button, waiting for a loading spinner to disappear, or waiting for specific text to appear in a text field. Currently, agents must implement their own polling loops:

```bash
# Current agent pattern: wasteful, verbose, error-prone
for i in $(seq 1 30); do
  result=$(desktop-cli read --app "Safari" --depth 3 --roles "btn")
  if echo "$result" | grep -q "Submit"; then
    break
  fi
  sleep 1
done
```

This pattern is:
1. **Token-wasteful** — every poll returns the full element tree, consuming agent context
2. **Verbose** — agents must write multi-line shell loops for every wait
3. **Fragile** — agents often pick wrong timeouts, miss edge cases, or fail silently
4. **Slow** — fixed sleep intervals add unnecessary latency

A dedicated `wait` command solves all of these by polling internally and only returning when the condition is met (or timeout is reached).

## What to Build

### 1. Command Definition — `cmd/wait.go` (new file)

Create a new `wait` command that polls the UI element tree until a specified condition is met.

```
desktop-cli wait [flags]

Flags:
  --app <name>          Scope to application
  --window <title>      Scope to window
  --for-text <text>     Wait until an element with this title/value text appears
  --for-role <role>     Wait until an element with this role appears (e.g. "btn", "input")
  --for-id <id>         Wait until an element with this ID exists
  --gone               Invert: wait until the condition is NO LONGER true
  --timeout <seconds>   Max time to wait (default: 30)
  --interval <ms>       Polling interval in milliseconds (default: 500)
  --depth <n>           Max depth to traverse (default: 0 = unlimited)
  --roles <roles>       Comma-separated roles to filter during read
```

#### Usage Examples

```bash
# Wait for a "Submit" button to appear (up to 30s)
desktop-cli wait --app "Safari" --for-text "Submit" --for-role "btn"

# Wait for a dialog to disappear
desktop-cli wait --app "Safari" --for-text "Loading..." --gone

# Wait for any input field to appear, with custom timeout
desktop-cli wait --app "Chrome" --for-role "input" --timeout 10

# Wait for element ID 5 to exist (after a page transition)
desktop-cli wait --app "Safari" --for-id 5 --timeout 15

# Fast polling for time-sensitive waits
desktop-cli wait --app "Safari" --for-text "Done" --interval 200
```

### 2. Implementation — `cmd/wait.go`

The command polls `provider.Reader.ReadElements()` in a loop, checking the returned elements against the wait condition each iteration.

```go
package cmd

import (
    "fmt"
    "strings"
    "time"

    "github.com/mj1618/desktop-cli/internal/model"
    "github.com/mj1618/desktop-cli/internal/output"
    "github.com/mj1618/desktop-cli/internal/platform"
    "github.com/spf13/cobra"
)

// WaitResult is the YAML output of a wait command.
type WaitResult struct {
    OK       bool   `yaml:"ok"`
    Action   string `yaml:"action"`
    Elapsed  string `yaml:"elapsed"`
    Match    string `yaml:"match,omitempty"`
    TimedOut bool   `yaml:"timed_out,omitempty"`
}

var waitCmd = &cobra.Command{
    Use:   "wait",
    Short: "Wait for a UI condition to be met",
    Long:  "Poll the UI element tree until a specified condition is met or timeout is reached.",
    RunE:  runWait,
}

func init() {
    rootCmd.AddCommand(waitCmd)
    waitCmd.Flags().String("app", "", "Scope to application")
    waitCmd.Flags().String("window", "", "Scope to window")
    waitCmd.Flags().Int("pid", 0, "Filter to a specific process by PID")
    waitCmd.Flags().Int("window-id", 0, "Filter to a specific window by system window ID")
    waitCmd.Flags().String("for-text", "", "Wait for element with this title or value text (substring match)")
    waitCmd.Flags().String("for-role", "", "Wait for element with this role (e.g. btn, input, lnk)")
    waitCmd.Flags().Int("for-id", 0, "Wait for element with this ID to exist")
    waitCmd.Flags().Bool("gone", false, "Invert: wait until the condition is NO LONGER true")
    waitCmd.Flags().Int("timeout", 30, "Max seconds to wait (default: 30)")
    waitCmd.Flags().Int("interval", 500, "Polling interval in milliseconds (default: 500)")
    waitCmd.Flags().Int("depth", 0, "Max depth to traverse (0 = unlimited)")
    waitCmd.Flags().String("roles", "", "Comma-separated roles to filter during read")
}

func runWait(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.Reader == nil {
        return fmt.Errorf("reader not available on this platform")
    }

    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")
    pid, _ := cmd.Flags().GetInt("pid")
    windowID, _ := cmd.Flags().GetInt("window-id")
    forText, _ := cmd.Flags().GetString("for-text")
    forRole, _ := cmd.Flags().GetString("for-role")
    forID, _ := cmd.Flags().GetInt("for-id")
    gone, _ := cmd.Flags().GetBool("gone")
    timeoutSec, _ := cmd.Flags().GetInt("timeout")
    intervalMs, _ := cmd.Flags().GetInt("interval")
    depth, _ := cmd.Flags().GetInt("depth")
    rolesStr, _ := cmd.Flags().GetString("roles")

    // Validate: at least one condition must be specified
    if forText == "" && forRole == "" && forID == 0 {
        return fmt.Errorf("specify at least one condition: --for-text, --for-role, or --for-id")
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
        PID:      pid,
        WindowID: windowID,
        Depth:    depth,
        Roles:    roles,
    }

    timeout := time.Duration(timeoutSec) * time.Second
    interval := time.Duration(intervalMs) * time.Millisecond
    deadline := time.Now().Add(timeout)
    start := time.Now()

    for {
        elements, err := provider.Reader.ReadElements(readOpts)
        if err != nil {
            // Transient errors during polling are OK — keep trying until timeout
            if time.Now().After(deadline) {
                return fmt.Errorf("timeout after %s (last error: %w)", timeout, err)
            }
            time.Sleep(interval)
            continue
        }

        matched := checkWaitCondition(elements, forText, forRole, forID)

        // If --gone flag: we want the condition to NOT be true
        conditionMet := matched
        if gone {
            conditionMet = !matched
        }

        if conditionMet {
            elapsed := time.Since(start)
            matchDesc := describeCondition(forText, forRole, forID, gone)
            return output.PrintYAML(WaitResult{
                OK:      true,
                Action:  "wait",
                Elapsed: fmt.Sprintf("%.1fs", elapsed.Seconds()),
                Match:   matchDesc,
            })
        }

        if time.Now().After(deadline) {
            elapsed := time.Since(start)
            matchDesc := describeCondition(forText, forRole, forID, gone)
            return output.PrintYAML(WaitResult{
                OK:       false,
                Action:   "wait",
                Elapsed:  fmt.Sprintf("%.1fs", elapsed.Seconds()),
                Match:    matchDesc,
                TimedOut: true,
            })
        }

        time.Sleep(interval)
    }
}

// checkWaitCondition checks if any element in the tree matches the wait criteria.
func checkWaitCondition(elements []model.Element, forText, forRole string, forID int) bool {
    for _, elem := range elements {
        if matchesCondition(elem, forText, forRole, forID) {
            return true
        }
        if checkWaitCondition(elem.Children, forText, forRole, forID) {
            return true
        }
    }
    return false
}

// matchesCondition checks if a single element matches all specified criteria.
// When multiple criteria are given, ALL must match (AND logic).
func matchesCondition(elem model.Element, forText, forRole string, forID int) bool {
    if forID > 0 && elem.ID != forID {
        return false
    }
    if forRole != "" && elem.Role != forRole {
        return false
    }
    if forText != "" {
        // Substring match on title or value (case-insensitive)
        textLower := strings.ToLower(forText)
        titleMatch := strings.Contains(strings.ToLower(elem.Title), textLower)
        valueMatch := strings.Contains(strings.ToLower(elem.Value), textLower)
        descMatch := strings.Contains(strings.ToLower(elem.Description), textLower)
        if !titleMatch && !valueMatch && !descMatch {
            return false
        }
    }
    return true
}

// describeCondition returns a human-readable description of what was waited for.
func describeCondition(forText, forRole string, forID int, gone bool) string {
    parts := []string{}
    if forRole != "" {
        parts = append(parts, fmt.Sprintf("role=%s", forRole))
    }
    if forText != "" {
        parts = append(parts, fmt.Sprintf("text=%q", forText))
    }
    if forID > 0 {
        parts = append(parts, fmt.Sprintf("id=%d", forID))
    }
    desc := strings.Join(parts, " ")
    if gone {
        desc += " (gone)"
    }
    return desc
}
```

### 3. Register Command — `cmd/root.go`

No changes needed — the command registers itself via `init()` in `cmd/wait.go`, following the same pattern as all other commands.

### 4. Update Documentation

**README.md** — Add a "Wait for UI conditions" section after "Focus a window":

```markdown
### Wait for UI conditions

```bash
# Wait for a "Submit" button to appear
desktop-cli wait --app "Safari" --for-text "Submit" --for-role "btn"

# Wait for a loading indicator to disappear
desktop-cli wait --app "Safari" --for-text "Loading..." --gone

# Wait for any input field to appear with custom timeout
desktop-cli wait --app "Chrome" --for-role "input" --timeout 10

# Wait for element ID 5 to exist
desktop-cli wait --app "Safari" --for-id 5

# Fast polling for time-sensitive waits
desktop-cli wait --app "Safari" --for-text "Done" --interval 200
```
```

**SKILL.md** — Add wait examples to the quick reference:

```markdown
### Wait for UI conditions

```bash
desktop-cli wait --app "Safari" --for-text "Submit" --for-role "btn"
desktop-cli wait --app "Safari" --for-text "Loading..." --gone
desktop-cli wait --app "Chrome" --for-role "input" --timeout 10
desktop-cli wait --app "Safari" --for-id 5
```
```

Also update the "Agent Workflow" section in SKILL.md to show `wait` in the typical loop:

```markdown
## Agent Workflow

1. `list --windows` to find the target window
2. `read --app <name> --depth 3 --roles "btn,lnk,input,txt"` to get the element tree as YAML
3. Use the element `i` (id) field to `click --id <id>` or `type --id <id> --text "..."`
4. `wait --app <name> --for-text "..." --timeout 10` to wait for UI to update
5. Repeat read/act/wait loop as needed
```

## Files to Create

- `cmd/wait.go` — New wait command implementation

## Files to Modify

- `README.md` — Add wait command usage section and update agent workflow example
- `SKILL.md` — Add wait command to quick reference and update agent workflow

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go test ./...` passes
- [ ] `desktop-cli wait --help` shows all flags with descriptions
- [ ] `desktop-cli wait --app "Finder" --for-role "btn" --timeout 5` returns quickly if buttons exist, or times out after 5s
- [ ] `desktop-cli wait --app "Finder" --for-text "nonexistent" --timeout 3` times out after 3s and outputs `timed_out: true`
- [ ] `desktop-cli wait --app "Finder" --for-text "sometext" --gone --timeout 3` waits until no element contains "sometext"
- [ ] `desktop-cli wait` (no conditions) returns a clear error message
- [ ] Combined conditions work: `--for-text "Submit" --for-role "btn"` matches only buttons with "Submit" text
- [ ] Text matching is case-insensitive and checks title, value, and description fields
- [ ] `--interval 100` increases polling frequency
- [ ] On success: outputs YAML with `ok: true`, elapsed time, and condition description
- [ ] On timeout: outputs YAML with `ok: false`, `timed_out: true`, elapsed time, and condition description
- [ ] Transient read errors during polling don't cause immediate failure (retries until timeout)
- [ ] README.md and SKILL.md are updated with wait command examples

## Implementation Notes

- **Pure Go, no CGo needed**: This command only uses `provider.Reader.ReadElements()` which already exists. No new platform interface or C code required.
- **Output on timeout is NOT an error**: The command outputs a YAML result with `ok: false` and `timed_out: true` rather than returning a Go error. This lets agents parse the output consistently regardless of outcome. The exit code should still be non-zero on timeout so shell scripts can use `&&` chaining.
- **Exit code on timeout**: Return `fmt.Errorf(...)` wrapping the YAML output isn't right — instead, print the YAML, then return an error like `return fmt.Errorf("timed out waiting for condition")` AFTER printing. Or better: use `cmd.SilenceErrors = true` and `os.Exit(1)` after printing the timeout result. Follow whichever error pattern the other commands use.
  - ACTUALLY: The simplest approach is to just return an error on timeout (after printing the YAML result). The error will be printed by cobra, and the process exits non-zero. This matches the existing command patterns.
- **Condition AND logic**: When multiple conditions are given (e.g., `--for-text "Submit" --for-role "btn"`), ALL must match on the SAME element. This is more useful than OR logic because it lets agents be precise.
- **`--gone` flag**: Inverts the condition — waits until NO element matches. Useful for waiting for loading spinners, dialogs, or transient UI to disappear.
- **Case-insensitive text matching**: Uses `strings.Contains(strings.ToLower(...), ...)` for robustness. Agents shouldn't need to worry about exact casing.
- **Substring matching**: `--for-text "Sub"` matches elements with title "Submit", "Subscribe", etc. This is intentional — exact match would be fragile and break on localization or minor text changes.
- **Transient error resilience**: Read errors during polling (e.g., app temporarily unresponsive) don't cause immediate failure. The command retries until timeout, since the issue often resolves itself.
- **Default timeout of 30s**: Long enough for most UI transitions (page loads, dialogs). Agents should specify shorter timeouts when they expect fast responses.
- **Default interval of 500ms**: Balance between responsiveness and CPU usage. 200ms is fine for fast polling; 1000ms is fine for slow operations. The accessibility API calls are cheap (~50ms each), so even 200ms intervals won't cause excessive CPU.
