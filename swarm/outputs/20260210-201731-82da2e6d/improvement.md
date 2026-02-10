# Click/action response display elements are often irrelevant

## Problem

When performing a `click` or `action` command, the response includes a `display` field with up to 20 "display elements" (read-only text with values). However, these display elements are frequently **irrelevant** to the action performed, showing unrelated UI elements like sidebar items instead of the content that actually changed.

Example: When clicking a column header in Finder to sort a list:

```bash
desktop-cli click --id 1543 --app "Finder"  # Click "Date Modified" column header
```

Response shows 20 sidebar elements (Favourites, AirDrop, Applications, code, supplywise, Desktop, etc.) instead of:
- The sorted list items
- The column header that was clicked
- Any indication the sort order changed

This forces agents to:
1. Take a screenshot to verify the action worked
2. Take another screenshot or read to see what changed
3. Waste tokens on irrelevant display element data

The display elements seem to be collected indiscriminately from the window rather than being scoped to the relevant context (the element clicked or the area affected by the action).

## Proposed Fix

Improve display element selection logic to prioritize **contextually relevant** elements:

1. **Target element context** — Include the clicked/acted-upon element and its immediate siblings/children
2. **Changed elements** — Detect which elements changed after the action (if possible) and include those
3. **Focused area** — Prioritize elements near the click coordinates or in the active content area
4. **Exclude irrelevant sidebars** — Deprioritize static sidebar/navigation elements unless that's what was clicked

Alternatively:
- Add a `--post-read` flag (similar to existing `--post-read` for `click`) that returns full UI state after the action instead of guessing which display elements matter
- Add a `--scope-id` concept to display collection: only collect display elements that are descendants of the clicked element or a specified container
- Change the default to collect fewer (5-10) display elements but make them more relevant

The goal: **display elements should help the agent understand what changed**, not show random sidebar items.

## Reproduction

```bash
# Open Finder to Applications folder in list view
open -a Finder
desktop-cli focus --app "Finder"
desktop-cli click --text "Applications" --app "Finder"
desktop-cli type --key "cmd+2"  # Switch to list view

# Click column header to sort
desktop-cli read --app "Finder" --text "Date Modified" --flat
# Outputs: [1543] btn "Date Modified" (954,251,181,28)

desktop-cli click --id 1543 --app "Finder"
```

**Expected**: Response `display` field shows the sorted list items (Claude, Discord, Desktop, etc.) or the column header itself

**Actual**: Response `display` field shows 20 irrelevant sidebar items (Favourites, AirDrop, Applications, code, supplywise, Desktop, Downloads, matt, iCloud, etc.)

This same issue likely affects many click/action commands where the clicked element is in a content area separate from navigation/sidebar elements.
