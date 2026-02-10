# Feature: Verified Actions with Automatic Retry and Fallback

## Priority: HIGH (eliminates silent failures — the invisible time sink)

## Problem

Actions silently fail more often than agents realize. The tool reports `ok: true` because the CLI command executed, but the UI didn't actually respond:

1. **Click doesn't register**: Coordinate-based clicks can miss due to Retina scaling issues, element overlap, or the element shifting between tree-read and click execution.
   - From the test output: *"The `click --text` command resolves to the correct button coordinates but the coordinate-based click doesn't register on Calculator (display stays at 0). This is a pre-existing coordinate/Retina scaling issue."*

2. **Type into wrong field**: Focus was on a different element than expected. Text goes somewhere unintended.

3. **Action on stale element**: Element ID from a previous read no longer exists or has moved.

When an action silently fails, the agent doesn't know. It continues with the next step, which also fails because the prerequisite action didn't happen. The agent burns 3-5 more commands discovering and recovering from the original failure.

**Time cost of a single silent failure**: ~15-20 seconds (agent does 3-5 more commands before realizing, then must backtrack and retry).

## What to Build

### 1. `--verify` Flag on Action Commands

Add `--verify` to `click`, `type`, `action`, and `set-value` that automatically checks if the action had the intended effect:

```bash
# Click with verification: confirms the button was pressed
desktop-cli click --text "Submit" --app "Safari" --verify

# Type with verification: confirms the value was set
desktop-cli type --target "Name" --app "Safari" --text "John" --verify

# Set-value with verification: confirms the value changed
desktop-cli set-value --id 42 --value "hello" --app "Safari" --verify
```

### 2. Verification Strategies Per Action Type

**Click verification:**
- Re-read the target element after clicking
- Check: element state changed (e.g., selected/focused toggled), OR element disappeared (navigated away), OR new elements appeared (dialog opened)
- If none of the above: retry with `action --text` (accessibility press instead of coordinate click)

**Type verification:**
- Re-read the target/focused element after typing
- Check: element's value now contains the typed text
- If not: try `set-value` as fallback (direct value injection)

**Action verification:**
- Re-read the target element after action
- Check: element state changed or expected side effect occurred
- Action is already the most reliable method — if it fails, report clearly

**Set-value verification:**
- Re-read the element after setting
- Check: value matches what was set
- If not: try `type` as fallback (keystroke simulation)

### 3. Auto-Retry with Fallback Chain

When verification fails, try the next strategy in the fallback chain before reporting failure:

```
click --text "Submit"
  → verify: did UI change?
    → YES: success
    → NO: retry with action --text "Submit" (accessibility press)
      → verify: did UI change?
        → YES: success
        → NO: retry with click at (x+1, y+1) (offset click for overlap)
          → verify: did UI change?
            → YES: success
            → NO: report failure with details
```

```
type --target "Name" --text "John"
  → verify: does value contain "John"?
    → YES: success
    → NO: retry with set-value --text "Name" --value "John"
      → verify: does value contain "John"?
        → YES: success
        → NO: report failure with details
```

### 4. Response Format

```yaml
# Success on first attempt:
ok: true
action: click
verified: true
target: { i: 89, r: btn, t: "Submit" }

# Success after retry:
ok: true
action: click
verified: true
retried: true
retry_reason: "click did not change UI state, retried with accessibility action"
target: { i: 89, r: btn, t: "Submit" }

# Verification failed after all retries:
ok: false
action: click
verified: false
error: "action did not produce expected UI change after 2 retries"
target: { i: 89, r: btn, t: "Submit" }
attempts:
  - method: click
    result: "no state change detected"
  - method: action
    result: "no state change detected"
```

### 5. Verification Delay

Some UI changes take time (animations, network requests). Add `--verify-delay <ms>` to wait before checking:

```bash
# Wait 500ms after click before verifying (for page transitions)
desktop-cli click --text "Submit" --app "Safari" --verify --verify-delay 500
```

Default: 100ms (enough for most synchronous UI updates).

### 6. Implementation

```go
func clickWithVerify(provider *platform.Provider, target *model.Element, elements []model.Element, opts ClickOpts) (*ClickResult, error) {
    // Snapshot pre-action state
    preState := snapshotElementState(target, elements)

    // Attempt 1: coordinate click
    err := provider.Inputter.Click(target.CenterX(), target.CenterY(), opts.Button)
    time.Sleep(opts.VerifyDelay)

    // Verify
    postElements, _ := provider.Reader.ReadElements(readOpts)
    postState := snapshotElementState(findElementByID(postElements, target.ID), postElements)

    if stateChanged(preState, postState) {
        return &ClickResult{OK: true, Verified: true}, nil
    }

    // Attempt 2: accessibility action
    err = provider.ActionPerformer.PerformAction(target, "press")
    time.Sleep(opts.VerifyDelay)
    // ... verify again ...

    // All retries exhausted
    return &ClickResult{OK: false, Verified: false, Error: "no state change"}, nil
}

func stateChanged(pre, post *ElementSnapshot) bool {
    // Any of these indicate success:
    return post == nil ||                    // element disappeared (navigation)
        pre.Value != post.Value ||           // value changed
        pre.Focused != post.Focused ||       // focus changed
        pre.Selected != post.Selected ||     // selection changed
        pre.ChildCount != post.ChildCount || // children changed (dialog opened)
        pre.Title != post.Title              // title changed
}
```

## Files to Modify

- `cmd/click.go` — Add `--verify` flag and verification logic
- `cmd/typecmd.go` — Add `--verify` flag and verification logic
- `cmd/action.go` — Add `--verify` flag and verification logic
- `cmd/setvalue.go` — Add `--verify` flag and verification logic
- `cmd/helpers.go` — Add shared verification/snapshot/retry helpers
- `README.md` — Document `--verify` flag
- `SKILL.md` — Add verification examples to agent workflow

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `click --text "X" --verify` checks if UI changed after clicking
- [ ] `type --target "X" --text "Y" --verify` checks if value was set
- [ ] `set-value --id N --value "X" --verify` checks if value matches
- [ ] Auto-retry with fallback chain works (click → action → offset click)
- [ ] Response includes `verified: true/false` and retry details
- [ ] `--verify-delay 500` waits before checking
- [ ] Without `--verify`, behavior is unchanged (no extra reads)
- [ ] Verification failures include clear error messages with what was expected vs. observed
- [ ] Max 3 retry attempts (configurable with `--max-retries`)
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Performance cost of --verify**: Adds 1 tree read per attempt (~100-200ms). For the common success case (1 attempt), that's ~200ms. For retry cases, up to ~600ms. This is FAR cheaper than the 15-20 seconds an agent wastes recovering from a silent failure.
- **State change detection**: The `stateChanged` heuristic is deliberately broad — ANY change indicates the action had an effect. False positives (unrelated change coincidentally happened) are rare and harmless (the action probably did work).
- **Element disappearing = success**: If the clicked element no longer exists after clicking, it likely navigated away or closed a dialog. This is a success, not a failure.
- **Fallback chain order matters**: `click` (coordinate) is tried first because it's the most natural interaction. `action` (accessibility API) is the fallback because it's more reliable but less faithful to real user interaction. Offset click handles edge cases where the element has a transparent overlay stealing clicks.
- **`--verify` as default (future)**: Could make verification the default and add `--no-verify` to opt out. But start with opt-in to avoid performance regression for agents that don't need it.
- **Interaction with `--post-read`**: When both `--verify` and `--post-read` are used, the post-read happens after verification (using the same tree read that verification already did). No extra cost.
