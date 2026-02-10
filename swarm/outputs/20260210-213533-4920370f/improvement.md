# Type command reports incorrect focused element after typing

## Problem

When typing into a modal dialog, the `type` command reports a stale/incorrect focused element that doesn't match where the text was actually entered.

**Command:**
```bash
desktop-cli type --text "Test List" --app "Reminders"
```

**Output:**
```yaml
ok: true
action: type
text: Test List
focused:
    i: 50
    r: input
    v: First reminder updated
    b: [549, 399, 391, 18]
```

**Actual result (verified via screenshot):**
The text "Test List" was correctly entered into a "New List" dialog's name field. However, the reported `focused` element shows:
- Wrong value: "First reminder updated" instead of "Test List"
- Wrong element: a reminder input field instead of the list name field

This makes it impossible for agents to verify that typing succeeded without taking a screenshot, since the reported focused element doesn't reflect the actual state.

## Proposed Fix

After typing, the `type` command should query the currently focused element **fresh** rather than returning a potentially stale element from before the type action. Specifically:

1. Read the focused element AFTER the type action completes
2. Ensure the reported `v` (value) field reflects what was just typed
3. If a dialog/modal appeared during typing, the focused element should be from that dialog

The response should accurately show:
```yaml
focused:
    i: <correct-id>
    r: input
    v: Test List      # matches what was typed
    t: Name           # or similar label
```

This would make the `type` response self-documenting and trustworthy for verification.

## Reproduction

1. Open Reminders app: `desktop-cli open --app "Reminders" --wait --timeout 10`
2. Click "Add List" button: `desktop-cli click --text "Add List" --app "Reminders"`
3. Type a list name: `desktop-cli type --text "Test List" --app "Reminders"`
4. Observe the `focused` element in the response - it will show incorrect element/value
5. Compare with screenshot: `desktop-cli screenshot --app "Reminders" --output /tmp/check.png` - the screenshot will show "Test List" correctly entered in the dialog
