# Apple Notes checklist checkboxes not exposed in accessibility tree

## Problem

When creating a checklist in Apple Notes, the checkbox elements themselves are not exposed in the macOS accessibility tree. This makes it impossible to interact with them using `desktop-cli`'s text/ID-based commands.

**Reproduction steps:**
1. Created a new note in Apple Notes with `cmd+n`
2. Added title "Shopping List"
3. Created checklist with `shift+cmd+l` and added 3 items
4. Attempted to find checkbox elements with `desktop-cli read --app "Notes" --roles "chk" --flat`
5. Result: `elements: []` (no checkboxes found)
6. Verified with `desktop-cli screenshot-coords --app "Notes"` - no bounding boxes shown around the visible checkboxes

The checkboxes are visible in screenshots but completely absent from the accessibility tree. The checklist text appears merged together (e.g., "Buy milkBuy bread") in a single text element.

## Proposed Fix

Add a fallback strategy for clicking elements that are visually present but not exposed in the accessibility tree:

1. **Enhanced screenshot-coords command** - Add a `--find-visual` flag that uses vision analysis to locate UI elements (like checkboxes) that aren't in the accessibility tree. Could return suggested click coordinates.

   Example:
   ```bash
   desktop-cli screenshot-coords --app "Notes" --find-visual "checkbox near Buy milk" --output /tmp/coords.png
   ```

   Response could include suggested click coordinates:
   ```yaml
   ok: true
   visual_matches:
     - description: "checkbox near Buy milk"
       x: 534
       y: 118
       confidence: 0.95
   ```

2. **Smart text-based clicking** - Enhance `click --text` to look for interactive elements near the matched text when the text itself isn't clickable. For example, `click --text "Buy milk" --app "Notes" --near left` could click the checkbox to the left of the text.

3. **Documentation update** - Add Apple Notes checklist checkboxes to the Known Limitations section in SKILL.md, explaining that coordinate-based clicking is currently required as a workaround.

## Reproduction

```bash
# Setup
desktop-cli focus --app "Notes"
desktop-cli type --key "cmd+n"
desktop-cli type --text "Shopping List"
desktop-cli type --key "enter"
desktop-cli type --key "shift+cmd+l"
desktop-cli type --text "Buy milk" --key "enter"
desktop-cli type --text "Buy bread" --key "enter"
desktop-cli type --text "Buy eggs"

# Try to find checkboxes
desktop-cli read --app "Notes" --roles "chk" --flat
# Result: elements: []

# Verify with screenshot
desktop-cli screenshot-coords --app "Notes" --output /tmp/notes.png
# Visual inspection shows checkboxes present but not annotated
```

The only current workaround is to use coordinate-based clicking (`click --x <x> --y <y>`), but this requires manually identifying pixel positions and is fragile across different window sizes/positions.
