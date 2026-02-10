# Misleading focused element in `type` command response

## Problem

When using the `type` command without a `--target` or `--id` flag, the response includes a `focused` field that often shows an incorrect or unrelated element instead of the actual element that received the input.

Example - typing "Test List" in Reminders' "New List" dialog:

```bash
desktop-cli type --text "Test List" --app "Reminders"
```

Response:
```yaml
ok: true
action: type
text: Test List
focused:
    i: 9
    r: cell
    d: My Lists, Upgrade Available
    b:
        - 231
        - 529
        - 260
        - 22
```

The text was actually typed correctly into the "Name:" input field in the dialog, but the response claims focus is on the "My Lists" cell in the sidebar. This is misleading - it makes it appear that:
1. The typing went to the wrong place
2. Focus somehow jumped to an unrelated element
3. The command may have failed

The same issue occurred when typing "First reminder" - it created the reminder correctly but reported focus on the wrong cell.

## Proposed Fix

The `type` command should capture and report the correct focused element in two ways:

1. **Before typing** - capture which element currently has focus (the element that will receive the text)
2. **After typing** - re-check the focused element to see if it changed (e.g., after pressing Enter, focus might move to the next field)

The response should accurately reflect:
- What element received the text (either by reading AXFocusedUIElement before typing, or by checking which element's value changed)
- What element has focus after the action completes

Alternative: If the focused element cannot be reliably determined, omit the `focused` field entirely rather than returning incorrect information. Incorrect data is worse than no data.

## Reproduction

1. Open Reminders app: `desktop-cli open --app "Reminders"`
2. Click through welcome screens and click "Add List"
3. Type a list name: `desktop-cli type --text "Test List" --app "Reminders"`
4. Observe the response shows `focused` as an unrelated cell, even though the text was correctly typed into the name input field
5. Click OK, then type a reminder: `desktop-cli type --text "First reminder"`
6. Again observe the response shows incorrect focused element (sidebar cell) even though the reminder was successfully created
