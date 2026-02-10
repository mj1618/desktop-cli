# Click/action/type commands return excessive display text

## Problem

When executing `click`, `action`, or `type` commands with `--app` specified, the response includes ALL display text elements (`r: txt` with values) from the entire application window. In apps like Notes with long lists of content, this results in massive outputs that consume tokens unnecessarily.

**Example:** Clicking the "New Note" button in Notes:

```bash
desktop-cli click --id 2130 --app "Notes"
```

Returns 98.9KB of output containing hundreds of note titles, timestamps, and snippets from the notes list sidebar - none of which are relevant to creating a new note.

The current behavior attempts to be helpful by including "display elements" (like Calculator results), but it's too broad. It returns display text from the entire app rather than limiting to contextually relevant elements.

## Proposed Fix

Limit display text in command responses to contextually relevant elements only:

1. **For Calculator-like apps**: Return display text that's visually prominent or marked as "primary result" (current behavior works well here)

2. **For other apps**: Apply heuristics to filter display text:
   - Only include display elements within the same container as the target element (e.g., same dialog, same panel)
   - Exclude display text from sidebars, lists, or scroll areas that aren't the direct focus of the action
   - Add a `--scope-id` parameter to commands to explicitly limit display text to descendants of a specific container
   - Add a flag like `--no-display` to disable display text collection entirely for performance
   - Consider limiting by proximity: only return display text within N pixels of the clicked element

3. **Alternative approach**: Only return display text when it's likely to be a "result" of the action:
   - Use font size as a heuristic (large text = likely a result display)
   - Use role/attribute heuristics (elements marked as "status" or "alert")
   - Only return display text that changed after the action

4. **Size limit**: Cap display text output at a reasonable size (e.g., 10KB or first 20 elements) with a warning if truncated

## Reproduction

1. Open Notes app with several notes in the sidebar
2. Run: `desktop-cli click --id 2130 --app "Notes"` (New Note button)
3. Observe that the response includes 98.9KB of output with hundreds of irrelevant note titles and snippets from the sidebar
4. Compare to Calculator where display text is relevant: `desktop-cli type --app "Calculator" --text "2+2="` returns only the expression and result (compact and useful)
