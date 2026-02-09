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

## Notes

This may be a Chrome-specific limitation. Chrome may require `--force-renderer-accessibility` flag or explicit accessibility API activation to expose web content through the macOS accessibility tree. It would be worth investigating whether:
- Safari exposes web content correctly
- Chrome with `--force-renderer-accessibility` works
- There's a programmatic way to trigger Chrome's accessibility mode
