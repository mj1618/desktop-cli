# Click Send Requires Role Disambiguation

## Issue
When trying to click the Send button in Gmail compose form, the command `desktop-cli click --text "Send"` fails with:
```
Error: multiple elements match text "Send" — use --id, --exact, or --scope-id to narrow:
  id=3433 role=btn desc="Send (⌘Enter)"
  id=3434 role=other desc="More send options"
```

Had to use workaround: `desktop-cli click --text "Send" --roles "btn"`

## Problem
- The "More send options" button (role=other) shouldn't match a "Send" text search
- Or the tool should preferentially pick the primary action (role=btn) when disambiguating
- This requires users to specify `--roles "btn"` even though they obviously want the button, not the "more options" element

## Efficiency Impact
- Requires an extra --roles flag that users shouldn't need to think about
- Makes the command harder to use when both elements match similar text
- The tool could be smarter about picking the primary action element

## Steps to Reproduce
1. Open Gmail in Chrome
2. Click Compose
3. Fill in To, Subject, Body fields
4. Try: `desktop-cli click --text "Send" --app "Google Chrome"` (fails)
5. Need to use: `desktop-cli click --text "Send" --roles "btn" --app "Google Chrome"` (works)
