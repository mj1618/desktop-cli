# Calculator type command doesn't work as documented

## Problem

The SKILL.md documentation states:

> For Calculator: `type --app "Calculator" --text "347*29+156="` types the full expression in 1 command (instead of 11 individual button presses)

However, when running this command:

```bash
desktop-cli type --app "Calculator" --text "347*29+156="
```

The command returns success:

```yaml
ok: true
action: type
text: 347*29+156=
display:
    - i: 9
      r: txt
      v: ‎2‎+‎2
      b: [710, 731, 37, 26]
    - i: 11
      r: txt
      v: ‎4
      b: [729, 761, 19, 36]
      primary: true
```

But Calculator still shows "2+2=4" (the previous calculation), not "347×29+156" and its result.

The issue is that Calculator doesn't have a text input field - it only has individual button elements (btn role). The `type` command appears to do nothing when there's no focusable input field to type into.

## Proposed Fix

The `type` command should detect when targeting Calculator and automatically translate the text into a sequence of button clicks:

1. Parse the input text character by character (e.g., "347*29+156=")
2. Map each character to its corresponding button:
   - Digits "0"-"9" → button with that text
   - "*" → "Multiply" button
   - "/" → "Divide" button
   - "+" → "Add" button
   - "-" → "Subtract" button
   - "=" → "Equals" button
   - "." → "Point" button
3. Click each button in sequence using the existing button-clicking infrastructure

This would make the documented behavior actually work. Alternatively, remove this claim from the documentation if Calculator automation isn't intended.

## Reproduction

1. Open Calculator app
2. Run: `desktop-cli type --app "Calculator" --text "347*29+156="`
3. Take screenshot: `desktop-cli screenshot --app "Calculator" --output /tmp/calc.png`
4. Observe that Calculator does NOT show the typed expression - it shows whatever was there before
5. Compare with reading the UI: `desktop-cli read --app "Calculator" --format agent` - shows only buttons, no input field
