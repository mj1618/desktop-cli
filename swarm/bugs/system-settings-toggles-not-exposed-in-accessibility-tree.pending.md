# System Settings Navigation and Element Discovery Friction

## Problem
When attempting to navigate System Settings and find toggle switches using `desktop-cli read` and `desktop-cli click`, there are several friction points:

1. The search functionality in System Settings doesn't filter navigation results or navigate to matching settings pages when queried (tested with "Dark" and "Appearance" searches)
2. Clicking on settings buttons in the sidebar (via `click --id`) doesn't reliably navigate to show the actual controls on the right content pane
3. When using `read --format agent`, toggle switches (checkboxes/radio buttons) in System Settings return empty results, making it unclear what settings can be toggled
4. The accessibility tree shows buttons and menu items in the sidebar, but the actual toggles and controls in the main content area are not properly exposed or discoverable

Example commands that didn't work as expected:
```bash
desktop-cli click --id 160 --app "System Settings"  # Clicked Date & Time but content didn't load
desktop-cli read --app "System Settings" --roles "chk"  # Returns empty
desktop-cli read --app "System Settings" --roles "radio"  # May also return empty
```

## Proposed Fix
1. Improve accessibility tree exposure for System Settings controls:
   - Ensure toggle switches, checkboxes, and radio buttons in the main content pane are properly discovered by the reader
   - Add descriptive labels/context for each setting option so agents can target them by text

2. Add support for discovering available settings and their toggleable states:
   - Provide a way to query what toggles are available on the current page
   - Include toggle state (on/off) in the output

3. Consider adding a helper command or flag like:
   - `read --app "System Settings" --roles "toggle"` (meta-role like "interactive")
   - A way to search and navigate settings more reliably

## Reproduction
```bash
# Open System Settings
open /System/Applications/System\ Settings.app

# Try to find and interact with toggle switches
desktop-cli focus --app "System Settings"
desktop-cli read --app "System Settings" --roles "chk"  # Returns empty
desktop-cli read --app "System Settings" --roles "radio"  # May be empty

# Try clicking through sidebar to navigate to a settings page
desktop-cli read --app "System Settings" --format agent  # Shows buttons but not content
desktop-cli click --id 160 --app "System Settings"  # Click Date & Time
desktop-cli read --app "System Settings" --format agent  # Still shows sidebar, not settings controls
```
