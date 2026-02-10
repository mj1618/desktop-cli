# Click --near flag clicks wrong location for Apple Notes checkboxes

## Problem

When attempting to check off a checklist item in Apple Notes using the `--near` flag, the click happened at completely wrong coordinates (1115, 576) instead of near the checkbox next to "Task 1: Buy groceries" text.

Command used:
```bash
desktop-cli click --text "Task 1: Buy groceries" --near --app "Notes"
```

Response:
```yaml
ok: true
action: click
x: 1115
y: 576
button: left
count: 1
```

The actual checkbox is at approximately x=534, y=118 (to the left of the task text), but the `--near` flag clicked far to the right and below the text instead.

The checkboxes in Apple Notes are not exposed in the accessibility tree at all (as documented in SKILL.md), so `--near` is supposed to be the workaround. But it's clicking at the wrong location, making it unreliable for this use case.

## Proposed Fix

The `--near` flag should use better heuristics for finding nearby interactive elements:

1. **Search in a radius**: When no interactive element is found in the accessibility tree near the text, search for clickable UI elements in a small radius around the text element (e.g., within 50px)

2. **Check to the left first**: For checklist patterns, checkboxes are typically to the LEFT of text labels. The algorithm should search left of the text element first before trying other directions.

3. **Fallback to bounding box edges**: If no interactive element is found in the accessibility tree, use the text element's bounding box and click at common checkbox positions relative to it:
   - Left edge minus offset (for checkboxes to the left of text)
   - Right edge plus offset (for toggle switches to the right)
   - Top-left corner (for some UI patterns)

4. **Add --near-direction flag**: Allow users to specify search direction: `--near-direction left|right|above|below` to help the algorithm find the right element when the default heuristic fails.

## Reproduction

1. Open Apple Notes
2. Create a new note with checklist items (use the checklist button in toolbar)
3. Try to check off the first item using:
   ```bash
   desktop-cli click --text "Task 1: Buy groceries" --near --app "Notes"
   ```
4. Observe that the click happens at wrong coordinates and checkbox doesn't get checked
5. Verify checkbox is still unchecked with a screenshot:
   ```bash
   desktop-cli screenshot --app "Notes" --output /tmp/check.png
   ```
