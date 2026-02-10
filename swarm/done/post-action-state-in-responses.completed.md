# Feature: Include Post-Action UI State in Responses

## Priority: MEDIUM (saves 1 LLM round-trip per verify step)

## Problem

After performing an action (click, type, action, set-value), agents frequently need to see what changed in the UI. The current flow is:

```bash
# Step 1: Click a button
desktop-cli click --text "Submit" --app "Safari"
# Returns: target element info + display elements

# Step 2: Agent must call read to see what happened
desktop-cli read --app "Safari" --format agent
# Returns: full UI state showing new elements (success message, new page, etc.)
```

Step 2 costs a full LLM round-trip (~3-5 seconds) just to see the result. The tool already includes `display` elements and `target`/`focused` info — but it doesn't show what NEW elements appeared or what the overall interactive state looks like after the action.

## What to Build

### 1. `--post-read` Flag on Action Commands

Add a `--post-read` flag to `click`, `type`, `action`, and `set-value` that automatically appends a compact read of the UI state after the action completes:

```bash
desktop-cli click --text "Submit" --app "Safari" --post-read
```

Output:

```yaml
ok: true
action: click
target:
    i: 89
    r: btn
    t: "Submit"
    b: [200, 400, 100, 32]
state: |
    # Contact Form - Safari (pid: 5678)
    [91] txt "Thank you for your submission" (100,250,300,20)
    [92] lnk "Return to home" (100,290,150,18)
    [93] btn "Close" (100,330,80,32)
```

The `state` field contains the same output as `read --format agent` — a compact one-line-per-element list of the current UI. This gives the agent the full picture in a single response.

### 2. Configurable Post-Read Options

The `--post-read` flag accepts optional modifiers via `--post-read-opts`:

```bash
# Default: agent format, interactive elements only
desktop-cli click --text "Submit" --app "Safari" --post-read

# Scoped to a container (e.g., after clicking within a modal)
desktop-cli click --text "OK" --app "Safari" --post-read --post-read-scope 156

# With specific roles
desktop-cli click --text "Submit" --app "Safari" --post-read --post-read-roles "btn,lnk,input,txt"

# With depth limit
desktop-cli click --text "Submit" --app "Safari" --post-read --post-read-depth 3
```

### 3. Optional Delay Before Post-Read

Some actions trigger UI changes that take a moment (animations, network requests). Add `--post-read-delay <ms>` to wait before reading:

```bash
# Wait 500ms after click for page transition, then read
desktop-cli click --text "Submit" --app "Safari" --post-read --post-read-delay 500
```

Default: 0ms (read immediately after action). Most accessibility tree updates are synchronous, so no delay is needed for simple interactions.

### 4. Implementation

In each action command's RunE function, after performing the action:

```go
if postRead {
    if postReadDelay > 0 {
        time.Sleep(time.Duration(postReadDelay) * time.Millisecond)
    }
    readOpts := platform.ReadOptions{
        App:    appName,
        Window: window,
        PID:    pid,
        Depth:  postReadDepth,  // default 0 (unlimited)
        Roles:  postReadRoles,  // default nil (all)
    }
    elements, err := provider.Reader.ReadElements(readOpts)
    if err == nil {
        // Format as agent output and include in response as "state" field
        state := output.FormatAgent(elements, appName, pid, windowTitle)
        result.State = state
    }
}
```

## Files to Modify

- `cmd/click.go` — Add `--post-read` flag and post-action read logic
- `cmd/typecmd.go` — Add `--post-read` flag and post-action read logic
- `cmd/action.go` — Add `--post-read` flag and post-action read logic
- `cmd/setvalue.go` — Add `--post-read` flag and post-action read logic
- `cmd/helpers.go` — Shared helper for post-read logic to avoid duplication
- `README.md` — Document `--post-read` flag
- `SKILL.md` — Add `--post-read` examples to agent workflow

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `--post-read` flag works on `click`, `type`, `action`, and `set-value`
- [ ] Post-read output uses agent format (compact, one-line-per-element)
- [ ] `--post-read-delay 500` waits 500ms before reading
- [ ] `--post-read-scope <id>` limits read to descendants of element
- [ ] `--post-read-roles "btn,lnk"` filters to specific roles
- [ ] `--post-read-depth 3` limits tree traversal depth
- [ ] Post-read failures don't cause the action itself to fail (best-effort)
- [ ] Without `--post-read`, behavior is unchanged (no extra read)
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Agent format only**: The post-read state is always in agent format for compactness. Agents that want full YAML/JSON can do a separate `read` call.
- **Best-effort**: If the post-read fails (e.g., app closed after clicking a close button), the action result is still returned successfully — the `state` field is simply omitted.
- **Interaction with `display` elements**: The `display` field and `state` field serve different purposes. `display` shows read-only text values (like a calculator display). `state` shows the full interactive element list. When `--post-read` is used, `display` elements are included in the `state` output (marked with `display` flag), so the separate `display` field can be omitted to avoid duplication.
- **Token cost tradeoff**: Including full state adds ~50-200 tokens per response. This is far cheaper than an additional LLM round-trip (~1000+ tokens for the read call + response). Worth it when the agent needs to decide what to do next.
- **Future: auto post-read**: Could make `--post-read` the default for all action commands, with `--no-post-read` to opt out. But start with opt-in to avoid breaking existing workflows.
