# Enhancement: `type` and `action` Commands Should Return Target Element Info

## Priority: HIGH (Eliminates most verification `read` calls — major efficiency gain for agents)

## Problem

When an agent runs `desktop-cli type --text "hello"` or `desktop-cli type --key "tab"`, the response is:

```yaml
ok: true
action: type
text: hello
```

There is no information about:
1. **Which element** received the input (role, title, description)
2. **What the element's value is** after the action

This forces agents to make a separate `read` call after every `type` or `action` to verify:
- The text actually went into the right field
- A key press (like Tab) moved focus to the expected element
- A button press actually triggered

In a typical form-filling workflow (e.g., composing an email), this doubles the number of CLI invocations — every `type` call needs a follow-up `read`.

## What to Build

### 1. Include Target Element Info in `type` Response

When `type` is used with `--target`, `--id`, or just bare (typing into the focused element), the response should include information about the element that received the input:

```yaml
ok: true
action: type
text: hello
target:
  i: 42
  r: input
  t: Subject
  v: hello
  b: [550, 792, 421, 20]
```

For `--key` actions, include the currently focused element after the key press:

```yaml
ok: true
action: key
key: tab
focused:
  i: 55
  r: input
  t: Message Body
  b: [550, 830, 421, 200]
```

### 2. Include Target Element Info in `action` Response

Similarly, `desktop-cli action --text "Submit" --app "Safari"` should return:

```yaml
ok: true
action: press
target:
  i: 89
  r: btn
  t: Submit
  b: [200, 400, 100, 32]
```

### 3. Implementation Approach

After performing the type/key/action:
1. If the command targeted a specific element (by `--text`, `--id`, or `--target`), re-read that element and include it in the response
2. For `--key` commands with no target, do a quick read to find the currently focused element (`f: true`) and include it
3. The target element info should use the same compact keys as the `read` command (`i`, `r`, `t`, `v`, `d`, `b`)
4. This should be a lightweight read — only the targeted/focused element, not the full tree

### 4. Update Documentation

Update SKILL.md and README.md to show the enhanced response format.

## Files to Modify

- `cmd/typecmd.go` — Add target/focused element to response after typing
- `cmd/action.go` — Add target element to response after action
- `README.md` — Document enhanced response format
- `SKILL.md` — Update examples

## Acceptance Criteria

- [ ] `desktop-cli type --target "Subject" --app "Chrome" --text "hello"` response includes the target element with its current value
- [ ] `desktop-cli type --key "tab"` response includes the newly focused element
- [ ] `desktop-cli type --id 42 --app "Chrome" --text "hello"` response includes the element info
- [ ] `desktop-cli action --text "Submit" --app "Safari"` response includes the target element
- [ ] Bare `desktop-cli type --text "hello"` (no target) includes the focused element in the response
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] README.md and SKILL.md updated

## Implementation Notes

- The re-read after action should be minimal — ideally just the single element, not the full tree. If the element was found by ID, re-read by ID. If by text, re-read with the same text filter.
- For `--key` commands, finding the focused element requires a read pass looking for `f: true`. Keep the read shallow (depth 2-3) and filter to common interactive roles to keep it fast.
- Consider making this opt-in with a `--verbose` flag if the extra read adds noticeable latency, but it's likely fast enough (<100ms) to always include.
