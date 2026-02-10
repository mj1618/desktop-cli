# Feature: Conditional and Branching Steps in `do` Command

## Problem

AI agents using the `do` batch command must currently assume a linear, deterministic UI flow. In practice, desktop UIs are non-deterministic:

- A cookie consent dialog may or may not appear
- A login prompt may show up if the session expired
- A confirmation dialog may require dismissal before continuing
- Different OS versions show different prompts
- Web pages load at variable speeds — an element may or may not exist yet

When any step fails in a `do` sequence, the agent must either:
1. Stop entirely (`--stop-on-error=true`) and make a new LLM round-trip to assess what happened
2. Continue blindly (`--stop-on-error=false`) and hope subsequent steps still make sense

Both options waste tokens and time. The agent can't express "click Accept if it exists, otherwise continue" — a pattern needed in nearly every real-world automation workflow.

## Solution

Add conditional step types to the `do` command that enable branching without LLM round-trips:

### New Step Types

#### `if-exists` — Conditional execution based on element presence

```yaml
- if-exists: { text: "Accept Cookies", app: "Chrome" }
  then:
    - click: { text: "Accept Cookies" }
```

#### `if-exists` with `else`

```yaml
- if-exists: { text: "Sign In", roles: "btn" }
  then:
    - click: { text: "Sign In" }
    - wait: { for-text: "Dashboard", timeout: 10 }
  else:
    - read: { format: "agent" }
```

#### `if-focused` — Branch based on which element has focus

```yaml
- if-focused: { roles: "input" }
  then:
    - type: { text: "search query", key: "enter" }
  else:
    - click: { text: "Search", roles: "input" }
    - type: { text: "search query", key: "enter" }
```

#### `try` — Execute steps and silently continue on error

```yaml
- try:
    - click: { text: "Dismiss" }
    - wait: { for-text: "Dismiss", gone: true, timeout: 2 }
- click: { text: "Continue" }
```

This replaces the global `--stop-on-error=false` with per-section error handling.

### Full Example: Robust Web Navigation

```yaml
desktop-cli do --app "Chrome" <<'EOF'
# Dismiss cookie banner if present
- try:
    - click: { text: "Accept", roles: "btn" }
    - sleep: { ms: 300 }

# Handle possible login redirect
- if-exists: { text: "Sign In", roles: "btn" }
  then:
    - click: { text: "Email", roles: "input" }
    - type: { text: "user@example.com", key: "tab" }
    - type: { text: "password123", key: "enter" }
    - wait: { for-text: "Dashboard", timeout: 15 }

# Main workflow
- click: { text: "Settings" }
- wait: { for-text: "Preferences", timeout: 5 }
- click: { text: "Notifications" }
EOF
```

### Response Format

Conditional steps report which branch was taken:

```yaml
steps:
  - step: 1
    type: if-exists
    condition: { text: "Accept Cookies" }
    matched: true
    branch: then
    substeps:
      - step: 1.1
        type: click
        ok: true
        x: 500
        y: 300
  - step: 2
    type: try
    ok: true
    substeps:
      - step: 2.1
        type: click
        ok: false
        error: "no element found matching text 'Dismiss'"
    # try block succeeded even though substep failed
```

## Implementation Plan

1. **Modify `cmd/do.go`** — extend step parsing:
   - `if-exists` step: uses existing `resolveElementByText` to check element presence
   - `if-focused` step: uses existing `readFocusedElement` to check focus state
   - `try` step: executes substeps, catches errors, always continues
   - Each conditional step contains `then`/`else` arrays of regular steps (recursive)

2. **YAML parsing** — the step structure changes from flat list to potentially nested:
   - Current: `[]StepConfig` where each is `map[string]interface{}`
   - New: same, but `if-exists`, `if-focused`, and `try` entries have `then`, `else`, or inline substep arrays
   - Regular steps remain unchanged — fully backward compatible

3. **Step execution** — refactor `executeStep` to handle:
   - Condition evaluation (element lookup without action)
   - Branch selection (then/else)
   - Recursive substep execution with scoped error handling
   - Step numbering uses dot notation (1.1, 1.2, etc.)

4. **Tests** — add test cases for:
   - `if-exists` with element found → executes `then`
   - `if-exists` with element not found → executes `else`
   - `if-exists` without `else` → skips silently
   - `try` with all steps succeeding
   - `try` with failing steps → continues and reports ok
   - Nested conditionals (if-exists inside try)
   - Backward compatibility (existing step configs unchanged)

5. **Update docs** — add conditional examples to README.md, SKILL.md

## Value

- **Eliminates 2-5 LLM round-trips** per workflow that encounters variable UI states
- **Enables robust automation** — agents can express "handle this if it appears" patterns
- **Backward compatible** — existing `do` YAML sequences work unchanged
- **Low latency** — condition checks use the same accessibility APIs already in use (no new dependencies)
- **Composable** — `try`, `if-exists`, `if-focused` can be combined freely

## Complexity

Medium — requires extending the YAML step parser and adding recursive substep execution, but all condition-checking primitives already exist in the codebase (element resolution, focus detection, role filtering).

## Dependencies

None — builds on existing `do` command infrastructure and helper functions.
