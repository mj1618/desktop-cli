# Calculator display text not visible in agent format

## Problem

When reading Calculator UI with `desktop-cli read --app "Calculator" --format agent`, the output shows all the buttons but doesn't show the display text elements that contain the calculation formula and result:

```
# Calculator

[11] btn "All Clear" (570,805,40,40)
[12] btn "Change Sign" (616,805,40,40)
...
[29] btn "Equals" (708,989,40,40)
```

The formula "347×29+156" and result "10219" are not visible. To read these values, you must either:
1. Use `desktop-cli read --app "Calculator" --roles "txt" --flat` to specifically filter for text elements
2. Fall back to `screenshot` and use vision model to read the display

This makes the agent format less useful for Calculator and similar apps where reading non-interactive display text is essential.

## Proposed Fix

The agent format should include important text elements that display state/output, even if they're not interactive. Specifically:

1. Include role `txt` elements in agent format when they contain meaningful content (non-empty `v` value)
2. Add a flag like `[readonly]` or `[display]` to distinguish non-interactive text from clickable elements
3. For Calculator specifically, the output should show:
   ```
   [9] txt "347×29+156" (624,731,124,26) [display]
   [11] txt "10219" (664,761,83,36) [display]
   [11] btn "All Clear" (570,805,40,40)
   ...
   ```

Alternative approach: Add a `--include-display-text` flag to agent format that includes static text elements with values.

## Reproduction

```bash
desktop-cli focus --app "Calculator"
# Perform any calculation by clicking buttons
desktop-cli click --text "3" --app "Calculator"
desktop-cli click --text "4" --app "Calculator"
desktop-cli click --text "7" --app "Calculator"
desktop-cli click --text "Multiply" --app "Calculator"
desktop-cli click --text "2" --app "Calculator"
desktop-cli click --text "9" --app "Calculator"
desktop-cli click --text "Add" --app "Calculator"
desktop-cli click --text "1" --app "Calculator"
desktop-cli click --text "5" --app "Calculator"
desktop-cli click --text "6" --app "Calculator"
desktop-cli click --text "Equals" --app "Calculator"

# Try to read the result with agent format
desktop-cli read --app "Calculator" --format agent
# Result: Only shows buttons, not the display text

# Compare with text-filtered flat output
desktop-cli read --app "Calculator" --roles "txt" --flat
# Result: Shows the formula and result clearly
```
