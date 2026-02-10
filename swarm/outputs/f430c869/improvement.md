# Agent format shows elements with zero-height bounds

## Problem

When using `read --format agent` on Google Maps search results, many text elements are shown with bounds that have 0 height, like `(1008,1235,117,0)`. These elements appear to be rendered off-screen or in a virtualized list, and are not actually visible to the user (confirmed by screenshot comparison).

Example output:
```
[315] txt "Stella's Coffee Pies Pastries||Stella's Best Coffee" (1008,1235,216,0) display
[319] txt " " (1144,1235,4,0) display
[320] txt "$1–20" (1148,1235,40,0) display
```

All these elements have height=0 in their bounds. A screenshot shows they are not visible on screen - they're below the fold in a scrollable list.

## Proposed Fix

Add a `--visible-only` flag (or make it the default for agent format) that filters out elements with zero-width or zero-height bounds. These elements are not actually visible or interactable, and including them:
- Wastes tokens for AI agents
- Creates confusion about what's actually on screen
- Makes the output less useful for agents that need to understand the current UI state

Alternative: Add a warning in the output when elements with 0-dimension bounds are present, e.g.:
```
# Warning: 23 elements have zero-dimension bounds (likely off-screen/virtualized)
```

## Reproduction

1. `desktop-cli focus --app "Google Chrome"`
2. Navigate to google.com/maps
3. Search for "coffee shops near Sydney Opera House"
4. `desktop-cli read --app "Google Chrome" --format agent`
5. Observe many elements with bounds like `(x,y,w,0)` where height is 0
6. Compare with `desktop-cli screenshot --app "Google Chrome"` to see they're not visible
