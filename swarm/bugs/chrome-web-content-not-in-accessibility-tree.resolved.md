# Chrome web content not exposed in accessibility tree

## Summary

When using `desktop-cli read` on Google Chrome, the accessibility tree only contains browser chrome elements (toolbar, tabs, address bar, bookmarks bar, window control buttons) but **no web page content** at all. No DOM elements from the loaded web page (buttons, links, inputs, text) are returned regardless of depth or role filters used.

## Steps to reproduce

1. Open Google Chrome with any web page (e.g. Gmail)
2. Run: `desktop-cli read --app "Google Chrome" --depth 15 --flat`
3. Observe that only browser UI elements are returned, not web page content

## Expected behavior

Web page content should be included in the accessibility tree, similar to how Safari and other browsers expose their DOM through the macOS accessibility API.

## Actual behavior

Only Chrome browser chrome elements are returned:
- Window control buttons (close, minimize, zoom)
- Back/Forward/Reload buttons
- Address bar
- Bookmarks bar
- Tab strip

The content area (group element containing the web view) has no children in the accessibility tree.

## Impact

This makes it impossible to:
- Find and click buttons on web pages (e.g. Gmail's "Compose" or "Send")
- Read text content from web pages
- Fill in form fields by finding them in the accessibility tree
- Interact with any web page element by text or ID

## Workarounds used

- Used macOS `open -a "Google Chrome" <url>` to navigate (since URL bar typing also had issues)
- Used Gmail's compose URL with query parameters to pre-fill email fields: `?view=cm&fs=1&to=...&su=...&body=...`
- Used keyboard shortcuts (`Cmd+Enter`) instead of clicking buttons
- Relied on window title changes to confirm actions succeeded

## Resolution

Fixed by setting the `AXEnhancedUserInterface` attribute to `true` on the application's `AXUIElementRef` before reading the accessibility tree. Chrome (and all Chromium-based browsers) lazily activate their accessibility tree — they only expose web page content when an assistive technology signals its presence via this attribute.

The fix is applied in all three C files that create app-level accessibility elements:
- `accessibility.c` (`ax_read_elements`) — the read path
- `action.c` (`ax_perform_action`) — the action path
- `set_value.c` (`ax_set_value`) — the set-value path

Each checks whether the attribute is already set to avoid a redundant 200ms activation delay on subsequent calls. The first interaction with Chrome incurs a one-time 200ms wait to allow Chrome to build its accessibility tree.
