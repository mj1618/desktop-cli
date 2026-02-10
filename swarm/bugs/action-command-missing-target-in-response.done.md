# Bug: `action` command sometimes returns no `target` in response

## Summary

When using `desktop-cli action --text "Send" --roles "btn" --app "Google Chrome"`, the response is missing the `target` element information that the documentation says should be included.

## Expected Behavior

Per SKILL.md, the `action` command response should include the target element:

```yaml
ok: true
action: action
id: 89
name: press
target:
    i: 89
    r: btn
    t: Submit
    b: [200, 400, 100, 32]
```

## Actual Behavior

The response only contained:

```yaml
ok: true
action: action
id: 3449
name: press
```

No `target` field was returned. This means the agent has no confirmation of what element was actually pressed, and cannot verify the action succeeded without doing a separate `read`.

## Steps to Reproduce

1. Open Gmail compose in Chrome
2. Run: `desktop-cli action --text "Send" --roles "btn" --app "Google Chrome"`
3. Observe the response lacks the `target` field

## Impact

- Agents lose the ability to verify which element was acted on
- Defeats the purpose of "no separate `read` needed after action" documented in SKILL.md
- May indicate the element disappeared (e.g. dialog closed) before the target info could be read back, but if so, the last-known state should still be returned
