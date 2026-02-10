# Feature: `do` Command — Batch Multiple Actions in One Call

## Priority: HIGH (biggest single speedup for agent workflows)

## Problem

Every agent interaction with desktop-cli requires a full LLM round-trip between actions. A typical 8-step form fill looks like:

```
LLM thinks (3s) → cli click (0.2s) → LLM thinks (3s) → cli type (0.1s) → LLM thinks (3s) → ...
```

The agent spends ~24 seconds on LLM inference and ~1.2 seconds on actual CLI work. The tool is fast; the back-and-forth is slow.

When an agent already knows the full sequence of steps (fill a form, navigate a menu, enter a calculation), it still has to execute them one-by-one with a round-trip between each. This is the single biggest performance bottleneck.

## What to Build

### 1. Command Definition — `cmd/do.go` (new file)

A `do` command that accepts a YAML list of actions on stdin and executes them sequentially, returning per-step results.

```
desktop-cli do [flags] < steps.yaml

Flags:
  --app <name>          Default app for all steps (can be overridden per-step)
  --stop-on-error       Stop execution on first error (default: true)
  --format <format>     Output format: yaml, json (default: yaml)
```

### 2. Input Format

Steps are provided as a YAML list on stdin. Each step is a command name with its flags as a map:

```yaml
- click: { text: "Full Name" }
- type: { text: "John Doe" }
- type: { key: "tab" }
- type: { text: "john@example.com" }
- type: { key: "tab" }
- type: { text: "Hello world" }
- click: { text: "Submit" }
- wait: { for-text: "Success", timeout: 5 }
```

Supported step types (maps directly to existing commands):
- `click`: { text, id, x, y, roles, button, double, near, exact, scope-id }
- `type`: { text, key, target, id, roles, delay }
- `action`: { text, id, action, roles, exact, scope-id }
- `set-value`: { text, id, value, attribute, roles }
- `scroll`: { direction, amount, text, id, x, y }
- `wait`: { for-text, for-role, for-id, gone, timeout, interval }
- `focus`: { app, window, pid, window-id }
- `read`: { text, roles, depth, flat, prune, scope-id, format, focused }
- `sleep`: { ms } (simple delay between steps)

Each step can optionally override `app`:
```yaml
- click: { text: "File", app: "Finder" }
- click: { text: "New Window", app: "Finder" }
- type: { text: "document.txt", app: "TextEdit" }
```

### 3. Output Format

Return a YAML list with one result per step. Each result includes the step index, the action performed, and the result from the underlying command:

```yaml
ok: true
action: do
steps: 8
results:
  - step: 1
    ok: true
    action: click
    target: { i: 42, r: input, t: "Full Name", b: [100,100,200,30] }
  - step: 2
    ok: true
    action: type
    text: "John Doe"
  - step: 3
    ok: true
    action: key
    key: tab
    focused: { i: 43, r: input, t: "Email", b: [100,140,200,30] }
  - step: 4
    ok: true
    action: type
    text: "john@example.com"
  - step: 5
    ok: true
    action: key
    key: tab
  - step: 6
    ok: true
    action: type
    text: "Hello world"
  - step: 7
    ok: true
    action: click
    target: { i: 89, r: btn, t: "Submit", b: [100,200,100,40] }
  - step: 8
    ok: true
    action: wait
    elapsed: "1.2s"
    match: 'text="Success"'
display:
  - i: 91
    r: txt
    v: "Form submitted successfully"
    b: [100,250,300,20]
```

On error with `--stop-on-error` (default), stop at the failing step and return partial results:

```yaml
ok: false
action: do
steps: 8
completed: 5
error: 'step 6: no element matches text "Nonexistent"'
results:
  - step: 1
    ok: true
    ...
  - step: 5
    ok: true
    ...
  - step: 6
    ok: false
    error: 'no element matches text "Nonexistent"'
```

### 4. Implementation Approach

The `do` command should reuse the existing command logic, NOT shell out to itself. Each step should call the same internal functions that the individual commands use:

```go
func runDo(cmd *cobra.Command, args []string) error {
    // 1. Parse YAML steps from stdin
    // 2. Get default --app from flags
    // 3. For each step:
    //    a. Determine command type (click, type, action, etc.)
    //    b. Build the appropriate options struct
    //    c. Call the internal handler (same code path as the standalone command)
    //    d. Collect the result
    //    e. If error and --stop-on-error, break
    // 4. Output combined results
}
```

Key: refactor existing command RunE functions to extract the core logic into reusable internal functions. For example, `click` currently has everything in `runClick()` — extract the logic into a `doClick(opts ClickOptions) (ClickResult, error)` that both `runClick` and `runDo` can call.

### 5. Usage Examples

```bash
# Form fill — 8 steps in 1 CLI call instead of 8
desktop-cli do --app "Safari" <<'EOF'
- click: { text: "Full Name" }
- type: { text: "John Doe" }
- type: { key: "tab" }
- type: { text: "john@example.com" }
- click: { text: "Submit" }
- wait: { for-text: "Thank you", timeout: 10 }
EOF

# Calculator — type full expression
desktop-cli do --app "Calculator" <<'EOF'
- action: { text: "C" }
- type: { text: "347*29+156=" }
EOF

# Multi-app workflow
desktop-cli do <<'EOF'
- focus: { app: "Safari" }
- click: { text: "Address", app: "Safari" }
- type: { text: "https://example.com", key: "enter" }
- wait: { for-text: "Example Domain", app: "Safari", timeout: 10 }
- read: { app: "Safari", format: "agent" }
EOF

# With sleep for animations
desktop-cli do --app "System Settings" <<'EOF'
- click: { text: "General" }
- sleep: { ms: 500 }
- click: { text: "About" }
- sleep: { ms: 500 }
- read: { format: "agent" }
EOF
```

## Files to Create

- `cmd/do.go` — New `do` command implementation

## Files to Modify

- `cmd/click.go` — Extract core logic into reusable function
- `cmd/typecmd.go` — Extract core logic into reusable function
- `cmd/action.go` — Extract core logic into reusable function
- `cmd/wait.go` — Extract core logic into reusable function
- `cmd/scroll.go` — Extract core logic into reusable function
- `cmd/setvalue.go` — Extract core logic into reusable function
- `cmd/read.go` — Extract core logic into reusable function
- `README.md` — Add `do` command documentation
- `SKILL.md` — Add `do` command to quick reference and agent workflow

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `desktop-cli do --help` shows usage and flags
- [ ] Piping a YAML step list via stdin executes all steps sequentially
- [ ] Each step result is captured and returned in the output
- [ ] `--stop-on-error` (default) halts on first failure and returns partial results
- [ ] `--stop-on-error=false` continues past failures, marking failed steps
- [ ] `--app` flag applies as default to all steps without explicit `app`
- [ ] Per-step `app` override works
- [ ] `sleep` step type pauses for the specified milliseconds
- [ ] Display elements are collected once at the end (not per-step) to reduce output
- [ ] Empty stdin returns a clear error message
- [ ] Invalid step type returns a clear error message
- [ ] README.md and SKILL.md updated with `do` command examples

## Implementation Notes

- **Refactoring existing commands**: The main challenge is extracting reusable logic from RunE functions. Each command currently reads flags directly from cobra — the refactored version should accept an options struct instead. The RunE function becomes a thin wrapper that reads flags → builds options → calls the internal function.
- **Stdin parsing**: Use `yaml.NewDecoder(os.Stdin)` to parse the step list. Each step is a map with one key (the command name) and a value map of flags.
- **Display elements**: Collect display elements only once after all steps complete (using the last step's app context), not per-step. This avoids redundant tree reads.
- **Error context**: On failure, include the step number and the original error message so agents know exactly which step failed and why.
- **Tree caching opportunity**: Since consecutive steps often target the same app/window, the tree read for element resolution could be cached within a `do` execution. If step N reads the tree for app X and step N+1 also targets app X (and no action was performed between that would change the tree), reuse the cached tree. This is an optimization for later — start with simple per-step tree reads.
- **No new dependencies**: Everything is built on existing internal functions. The YAML parsing uses the same `gopkg.in/yaml.v3` already in go.mod.
