# Reading structured list results requires too many exploration commands

## Problem

When trying to read the first search result from Google Maps, I had to run 4-5 separate commands to locate and read the list items:

```bash
# Command 1: Try to find results (didn't show list items)
desktop-cli read --app "Google Chrome" --text "Results" --flat --prune

# Command 2: Try broader search (found container but not items)
desktop-cli read --app "Google Chrome" --text "coffee" --flat --prune

# Command 3: Finally scope to container ID to see items
desktop-cli read --app "Google Chrome" --scope-id 156 --format agent
```

This pattern repeats for any structured list/results UI:
- Google search results
- Gmail inbox messages
- Calendar events
- Finder file lists
- App store search results

Each time an agent needs to extract items from a list, it must:
1. Take a screenshot to confirm list exists
2. Search for container text/role
3. Identify the container element ID
4. Re-read with `--scope-id` to see children

This is **4x more commands** than necessary and wastes tokens on multiple read outputs.

## Proposed Fix

Add a `--list-items` flag to the `read` command that automatically:
1. Detects repeating sibling elements (common role + similar structure)
2. Returns only the list items, not the container hierarchy
3. Groups items by their common parent

Example usage:
```bash
# Instead of 4 commands, just one:
desktop-cli read --app "Google Chrome" --text "Results" --list-items --format agent

# Output would show just the list items:
[171] lnk "MCA Cafe at Tallawoladah" ...
[204] lnk "Dutch Smuggler Coffee Brewers" ...
[239] lnk "Bennie's Cafe · Bistro" ...
```

Alternative approach: Add a `--depth 1` equivalent that means "direct children only" when combined with `--scope-id` or `--text` targeting.

## Reproduction

```bash
# Open Chrome and navigate to Google Maps
open -a "Google Chrome"
desktop-cli type --text "google.com/maps" --key "enter"
sleep 3

# Search for something with list results
desktop-cli type --target "Search Google Maps" --app "Google Chrome" --text "coffee shops near Sydney Opera House" --key "enter"
sleep 3

# Try to read the list items - requires multiple steps:
desktop-cli read --app "Google Chrome" --text "Results" --flat --prune  # Doesn't show items
desktop-cli read --app "Google Chrome" --scope-id <ID> --format agent   # Need to find ID first
```
