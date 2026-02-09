# Enhancement: `type --key "tab"` Should Report Which Element Received Focus

## Priority: MEDIUM (Closely related to "type returns target element info" — may be implemented together)

## Problem

When an agent presses Tab to navigate between form fields:

```bash
desktop-cli type --key "tab"
```

The response is simply:

```yaml
ok: true
action: key
key: tab
```

The agent has no idea where focus moved to. Did Tab go to the Subject field? The Body field? Some hidden UI element? The agent must issue a separate `read` command and scan for `f: true` to find out — an expensive and wasteful operation.

This is especially problematic for form navigation workflows (e.g., filling out email compose fields) where the agent uses Tab to move between To, Subject, and Body. Each Tab press requires a follow-up read to verify the navigation worked, doubling the number of CLI calls.

## What to Build

### 1. After Key Press, Read Back the Focused Element

When `type --key` is used for navigation keys (Tab, Shift+Tab, Enter, Escape, arrow keys), the response should include the element that now has focus:

```yaml
ok: true
action: key
key: tab
focused:
  i: 3463
  r: input
  d: Subject
  b: [550, 792, 421, 20]
```

### 2. Scope

This applies specifically to `type --key` commands. Navigation keys where focus feedback is most valuable:
- `tab`, `shift+tab` — form field navigation
- `enter` — may trigger actions or move focus
- `escape` — may close dialogs or move focus
- Arrow keys — may navigate within lists, menus, tabs

For regular `type --text` commands, the target element info (covered in the sibling task) is more relevant.

### 3. Implementation

After sending the key event:
1. Brief delay (50-100ms) to let the UI update focus
2. Quick read of the app's element tree, filtered to find `f: true`
3. Include that element in the response as `focused`
4. If no focused element is found, omit the `focused` field

## Files to Modify

- `cmd/typecmd.go` — Add focused element lookup after key press
- `README.md` — Document enhanced key response format
- `SKILL.md` — Update examples

## Acceptance Criteria

- [ ] `desktop-cli type --key "tab"` response includes the newly focused element
- [ ] `desktop-cli type --key "shift+tab"` response includes the newly focused element
- [ ] Focus info includes at minimum: id, role, title/description, bounds
- [ ] When no element has focus after key press, the `focused` field is omitted
- [ ] Regular `type --text` is unaffected (or has its own target element — see sibling task)
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] README.md and SKILL.md updated

## Implementation Notes

- This task overlaps significantly with the "type and action return target element info" task. They should likely be implemented together, with `type --key` returning `focused` and `type --text`/`type --target` returning `target`.
- The focus read after key press should be fast — use a shallow depth and filter to interactive roles (`input`, `btn`, `txt`, `lnk`, `chk`, `radio`, `tab`, `menuitem`) to minimize traversal time.
- A small delay after the key event is important because macOS accessibility focus updates are asynchronous. 50-100ms should be sufficient.
- Consider making this opt-out with `--no-feedback` if the extra read adds latency for agents that don't need it (e.g., rapid-fire key sequences).
