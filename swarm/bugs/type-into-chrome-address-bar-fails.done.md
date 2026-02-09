# Typing into Chrome address bar does not navigate

## Summary

Using `desktop-cli type` to enter a URL in Chrome's address bar and pressing Enter does not result in navigation. The page remains on "New tab" despite the type and key commands reporting success.

## Steps to reproduce

1. Open Google Chrome with a new tab
2. Run:
   ```bash
   desktop-cli focus --app "Google Chrome"
   desktop-cli type --key "cmd+l"        # focus address bar
   desktop-cli type --text "mail.google.com" --delay 20
   desktop-cli type --key "enter"
   ```
3. Observe that the page stays on "New tab"

Also tried:
- Clicking the address bar by element ID first (`click --id <address-bar-id> --app "Google Chrome"`)
- Using `cmd+a` to select all before typing
- Multiple attempts with varying delays

## Expected behavior

After typing a URL into the focused address bar and pressing Enter, Chrome should navigate to the URL.

## Actual behavior

All commands report `ok: true` but the page remains on "New tab". The window title does not change and the address bar does not show the typed URL on subsequent reads.

## Possible causes

- Chrome's address bar autocomplete/suggestions may be intercepting the typed text
- The `type` command may be sending keystrokes too fast for Chrome's omnibox to process
- The `cmd+l` shortcut may not reliably focus the address bar text input
- Chrome may be handling synthetic key events differently from real keyboard input

## Workaround

Used macOS shell command instead:
```bash
open -a "Google Chrome" "https://mail.google.com"
```

## Resolution

**Root cause**: The `cg_type_char` C function created keyboard events with virtual
key code `0` (the 'a' key) for every character, relying solely on
`CGEventKeyboardSetUnicodeString` to set the actual character. Chrome's omnibox
(and other Chromium-based input fields) inspects the raw virtual key code on the
CGEvent rather than only the Unicode string, so every character was effectively
ignored or misinterpreted.

**Fix**: Added a complete ASCII character-to-virtual-key-code mapping (US layout)
in `internal/platform/darwin/inputter.go`. The `TypeText` function now sets the
correct macOS virtual key code and shift modifier for each character (a-z, A-Z,
0-9, common symbols, whitespace). Non-ASCII characters fall back to key code 0
with Unicode string override. This makes synthetic keyboard events
indistinguishable from real keyboard input for all ASCII text.
