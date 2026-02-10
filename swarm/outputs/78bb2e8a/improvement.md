# Include display/value elements in action responses

## Problem

When using `action` command to interact with Calculator (or any app with display elements), the response only includes the target button that was pressed, but NOT the updated display/value that resulted from the action.

Example - pressing the "3" button in Calculator:

```bash
desktop-cli action --text "3" --app "Calculator"
```

Response:
```yaml
ok: true
action: action
id: 24
name: press
target:
    i: 24
    r: btn
    d: "3"
    b: [662, 943, 40, 40]
```

The response tells me which button I pressed, but NOT what the calculator display now shows. To see the result, I must make a follow-up read:

```bash
desktop-cli read --app "Calculator" --format agent
# Shows: [9] txt "5%3" (686,761,62,36) display
```

This doubles the number of round-trips for calculator workflows. For a calculation like "347 * 29 + 156", I need ~20 round trips (10 button presses + 10 reads) instead of ~10.

## Proposed Fix

When `action` or `click` commands are used, automatically include any display/read-only value elements in the response alongside the target element. Specifically:

1. After executing the action, query for elements with the `display` flag (elements marked as display text in agent format, or read-only text elements with values)
2. Include them in the response as a `display` field:

```yaml
ok: true
action: action
id: 24
name: press
target:
    i: 24
    r: btn
    d: "3"
    b: [662, 943, 40, 40]
display:
    - i: 9
      r: txt
      v: "5%3"
      b: [686, 761, 62, 36]
```

This would work for:
- **Calculator**: Include the display text element after each button press
- **Other apps**: Include any prominent read-only value/display elements that changed
- **General case**: Include elements that have the `display` flag or are read-only text with values

Alternatively, add a `--include-display` flag to opt-in to this behavior if including it by default is too expensive.

## Reproduction

1. Open Calculator app
2. Run: `desktop-cli action --text "3" --app "Calculator"`
3. Observe the response only includes the button target, not the display value
4. Must run `desktop-cli read --app "Calculator" --format agent` to see the display shows "3"
5. Repeat for any calculator operation - each button press requires a follow-up read to see the result

This same issue affects any app where actions update display elements (e.g. System Settings sliders, music players showing track info, etc.)
