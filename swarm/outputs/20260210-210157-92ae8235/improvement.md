# Auto-filter zero-dimension elements in text-based targeting

## Problem

When using text-based targeting (e.g. `click --text "Flights"`), the tool includes elements with zero-width or zero-height bounds in the matching candidates, causing ambiguity errors even when only one visible element actually matches.

Example command and error:
```bash
desktop-cli click --text "Flights" --roles "lnk" --app "Google Chrome"
```

Error output shows 20+ matching elements, but most have zero-height bounds:
```
id=92 lnk (417,214,68,48) desc="Flights"  # VISIBLE - this is what I want
id=197 lnk (310,832,228,31) desc="Sydney To London Flights..."  # VISIBLE
id=216 lnk (310,978,474,31) desc="Sydney to London (SYD – LON)..."  # VISIBLE
id=424 lnk (310,1117,652,0) desc="Show flights on Google Flights"  # ZERO HEIGHT - not visible
id=430 lnk (310,1117,460,0) desc="Cheap Flights from Sydney..."  # ZERO HEIGHT - not visible
id=453 lnk (310,1117,493,0) desc="Find Cheap Flights..."  # ZERO HEIGHT - not visible
... (15 more zero-height elements)
```

The majority of "matches" are off-screen/virtualized elements with zero dimensions that aren't actually clickable. This forces the user to add `--id` or use other workarounds when there's really only a few visible candidates.

## Proposed Fix

When performing text-based targeting (`--text` flag on `click`, `action`, `set-value`, `type`, `hover`, etc.), automatically filter out elements with zero width OR zero height BEFORE checking for ambiguity.

Zero-dimension elements are already excluded from agent format output (as documented in SKILL.md line 66: "Elements with zero-width or zero-height bounds (off-screen/virtualized) are automatically excluded"). The same filtering logic should apply to text-based targeting to prevent false ambiguity.

Implementation:
- In `cmd/helpers.go`, modify `findElementByText()` and related targeting functions to filter candidates by `elem.Bounds.Width > 0 && elem.Bounds.Height > 0` before returning matches
- This filtering should happen after role/text matching but before the ambiguity check
- The filtering should NOT apply when using `--id` (since the user explicitly requested a specific element ID)
- Document this behavior in the error message: "X elements match (Y visible)" so users understand why some matches were excluded

## Reproduction

1. Open Chrome and search for "flights from Sydney to London" on google.com
2. Wait for results to load
3. Run: `desktop-cli click --text "Flights" --roles "lnk" --app "Google Chrome"`
4. Observe error about 20+ matches, most with zero-height bounds
5. Expected: Should match only the 2-3 visible "Flights" elements, or auto-select id=92 (the filter tab) since it's the only one that matches the exact single word "Flights" among visible elements
