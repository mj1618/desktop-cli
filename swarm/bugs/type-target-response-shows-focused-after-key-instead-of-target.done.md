# type --target response shows focused element after --key instead of actual target

## Summary

When using `type --target "Subject" --text "Test Subject" --key "tab"`, the response `target` field shows the element that received focus after the key press (Message Body), not the actual element that was targeted and typed into (Subject).

## Steps to Reproduce

1. Open Gmail compose in Chrome
2. Run: `desktop-cli type --target "Subject" --roles "interactive" --app "Google Chrome" --text "Test Subject" --key "tab"`

## Expected Response

The response should show the actual target element (Subject field) with its updated value:

```yaml
ok: true
action: type+key
text: Test Subject
key: tab
target:
    i: 3321
    r: input
    v: Test Subject
    d: Subject
    b: [1073, 591, 568, 20]
```

## Actual Response

The response shows the Message Body element (which received focus after the tab key press) as the "target":

```yaml
ok: true
action: type+key
text: Test Subject
key: tab
target:
    i: 3337
    r: input
    v: |4+
    d: Message Body
    b: [1073, 631, 568, 416]
```

## Impact

- The `target` field in the response is supposed to show "the target element with its updated value" (per docs), but when `--key` is combined with `--target`, it reports the wrong element
- This is confusing for agents because the response suggests the text was typed into the Message Body instead of the Subject field
- An agent relying on the response to verify what happened would get incorrect information
- The actual typing works correctly — "Test Subject" does end up in the Subject field — but the response metadata is wrong

## Root Cause (likely)

The code probably re-reads the focused element after executing both the type and key actions, and returns that as the target. When `--key "tab"` moves focus to a different element, the response incorrectly reports the newly-focused element rather than the original target element. The fix should capture the target element info after typing but before executing the key press, or re-read the original target element after both actions complete.
