# type command target response returns incomplete element data

## Summary

When using `type --target` or `type --id`, the response includes a `target` object, but it's missing key fields (title, value, description) and reports an incorrect role. This makes it impossible to verify input was entered correctly without doing a separate `read`.

## Steps to Reproduce

1. Open Gmail compose in Chrome
2. Run: `desktop-cli type --target "Subject" --roles "input" --app "Google Chrome" --text "Test Subject"`

## Expected Response

```yaml
ok: true
action: type
text: Test Subject
target:
    i: 3369
    r: input
    t: Subject
    v: Test Subject
    d: Subject
    b: [1144, 1023, 97, 20]
```

## Actual Response

```yaml
ok: true
action: type
text: Test Subject
target:
    i: 3369
    r: group
    b: [1144, 1023, 97, 20]
```

Missing: `t` (title), `v` (value), `d` (description). Role reported as `group` instead of `input`.

## Same issue with type --id

When typing into the To recipients field:
```
desktop-cli type --id 3342 --app "Google Chrome" --text "matt@supplywise.app" --key "enter"
```

The error listing identified the element as `id=3342 role=other desc="To recipients"`, but the response returned `r: group` with no `t`, `v`, or `d` fields.

## Why This Matters

The SKILL.md documentation says "Responses automatically include target/focused element info — no separate read needed after typing." But because the target data is incomplete, agents can't verify input was entered correctly and may need to do a follow-up `read` anyway, defeating the purpose of including target info in the response.

Note: typing without `--target` (unfocused typing) correctly returns complete `focused` element data including the value.
