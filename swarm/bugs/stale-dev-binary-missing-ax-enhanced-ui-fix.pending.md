# Stale dev binary missing AXEnhancedUserInterface fix

## Summary

The installed `desktop-cli` binary at `/usr/local/bin/desktop-cli` was a stale dev build (`version dev (commit: none, built: unknown)`) that did not include the `AXEnhancedUserInterface` fix for Chrome/Chromium browsers. This caused a complete inability to read web page content from Google Chrome — only browser chrome elements (toolbar, tabs, address bar) were visible in the accessibility tree.

## Steps to reproduce

1. Have a stale `desktop-cli` binary installed (one built before the AXEnhancedUserInterface fix was applied)
2. Open Google Chrome with Gmail
3. Run: `desktop-cli read --app "Google Chrome" --depth 15`
4. Observe that only browser chrome elements are returned — no web page content

## Impact

This made it impossible to interact with any web page elements (buttons, links, inputs, text) in Chrome. The agent had to resort to workarounds:
- Navigating via URL parameters instead of clicking UI elements
- Using keyboard shortcuts (Cmd+Enter) instead of clicking buttons
- Relying on window title changes to infer state

## Resolution

Rebuilding from source (`go build -o desktop-cli .`) and reinstalling fixed the issue. After rebuilding, the full Gmail accessibility tree was visible including Compose button, To/Subject/Body inputs, Send button, and all email content.

## Suggestion

Consider adding a version stamp to dev builds (e.g. git commit hash or build date) so it's easier to tell if the installed binary is current. The `update.sh` script could also verify the build succeeded before replacing the binary.
