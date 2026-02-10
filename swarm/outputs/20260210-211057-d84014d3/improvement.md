# Text input fields missing from accessibility tree but still receive focus

## Problem

In the Reminders app "New List" dialog, the text input field next to "Name:" is completely absent from the accessibility tree when running `desktop-cli read --app "Reminders"`. The YAML output shows the "Name:" label as static text and the color checkboxes, but no input field.

Command:
```bash
desktop-cli read --app "Reminders"
```

Output includes:
```
[67|txt.3] txt "Name:" (382,598,45,18) display
[70|colour-blue/red] chk "Red" (437,632,18,18) unchecked
```

But visually (screenshot shows) there is clearly a text input field between these elements. When typing blindly with `desktop-cli type --text "Test List"`, the text IS successfully entered into the field, but the response claims the focused element is:

```yaml
focused:
    i: 9
    r: cell
    d: My Lists, Upgrade Available
    b: [231, 529, 260, 22]
```

This is completely misleading - the actual focused element is the text input field that now contains "Test List", not a cell labeled "My Lists, Upgrade Available".

## Proposed Fix

The `type` command (and potentially `read --focused`) should detect when macOS reports a focused element that doesn't make sense given the context. Specifically:

1. **Enhanced focus detection**: When a text field receives keyboard input but the accessibility API reports focus on a different element, the tool should attempt to find the actual input field receiving the text (perhaps by querying for elements with changed values, or using alternative accessibility queries).

2. **Warning output**: If the focused element reported by macOS doesn't match expected behavior (e.g., typing text but focused element is a non-editable cell), include a warning in the response:
   ```yaml
   ok: true
   action: type
   text: Test List
   focused:
       i: 9
       r: cell
       d: My Lists, Upgrade Available
   warning: "Focused element does not appear to be editable. Text may have been entered into a different field."
   ```

3. **Alternative detection methods**: Implement fallback methods to detect text input fields that aren't properly exposed in the accessibility tree:
   - Query for elements with AXRole "AXTextField" even if they're marked as non-accessible
   - Check for elements with typing focus using AXFocusedUIElement at a lower level
   - Detect elements with selection/insertion point attributes

## Reproduction

1. Open Reminders app: `desktop-cli open --app "Reminders"`
2. Click "Add List" button: `desktop-cli click --text "Add List" --app "Reminders"`
3. Read UI: `desktop-cli read --app "Reminders"` → No input field in output
4. Read with input role filter: `desktop-cli read --app "Reminders" --roles "input"` → Empty output
5. Type text: `desktop-cli type --text "Test List"` → Succeeds but reports wrong focused element
6. Screenshot to verify: `desktop-cli screenshot --app "Reminders" --output /tmp/check.png` → Shows text was entered in a field that's not in the accessibility tree
