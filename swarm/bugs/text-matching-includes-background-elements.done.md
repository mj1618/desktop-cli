# Text matching includes background/unrelated elements when dialog is open

## Summary

When using `--target` or `--text` to find elements, the text search matches against the entire accessibility tree including background elements that aren't part of the current dialog/modal. This causes frequent "multiple elements match" errors when common words like "Subject" appear both in the active compose dialog and in the background inbox.

## Steps to Reproduce

1. Open Gmail in Chrome with inbox visible
2. Click Compose to open the compose dialog
3. Run: `desktop-cli type --target "Subject" --app "Google Chrome" --text "Test"`

## Expected Behavior

Should match only the Subject input field in the compose dialog (the frontmost/active UI).

## Actual Behavior

Matches 17+ elements including:
- `id=3369 role=input desc="Subject"` (the actual compose Subject field)
- `id=364 role=chk title="unread, me , Test Subject , 20:59 , Test body."` (inbox row)
- `id=420 role=chk title="unread, me , test subject , 20:48 , test body."` (inbox row)
- Multiple `role=txt` elements from inbox email previews

## Workaround

Using `--roles "input"` narrows it down successfully, but this requires the agent to know the element's role in advance.

## Suggested Improvement

Consider one or more of:
1. A `--scope-id <id>` flag to limit text matching to descendants of a specific element (e.g., the compose dialog container)
2. Prioritize elements in the frontmost/focused dialog when multiple matches exist
3. A `--exact` flag for exact title/description match rather than substring matching
4. Automatically prefer elements closer to the focused element in the tree hierarchy

This would significantly reduce the back-and-forth needed when interacting with apps that have modals/dialogs overlaying content.
