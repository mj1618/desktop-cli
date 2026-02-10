# Smart role disambiguation for numeric/single-character text matches

## Problem

When clicking Calculator buttons (or similar UI elements with single-character labels), text-based clicking requires explicit `--roles "btn"` to disambiguate between the display text and button text. This creates excessive friction for common workflows.

Example workflow: entering "347" requires:
```bash
desktop-cli click --text "3" --roles "btn" --app "Calculator"  # ERROR without --roles
desktop-cli click --text "4" --roles "btn" --app "Calculator"  # ERROR without --roles
desktop-cli click --text "7" --roles "btn" --app "Calculator"  # ERROR without --roles
```

Error without `--roles`:
```
Error: multiple elements match text "3" — use --id, --exact, or --scope-id to narrow:
  id=9 role=txt
  id=26 role=btn desc="3"
```

The display shows "347" as text elements, and each digit button exists as a btn element. Every numeric click requires disambiguation.

## Proposed Fix

Implement smart role prioritization in text-based targeting: when multiple elements match the text, prefer interactive elements (btn, lnk, input, etc.) over static text elements (txt, statictext).

Specifically:
1. When `--text` matches multiple elements and `--roles` is NOT specified
2. If matches include BOTH interactive elements (btn/lnk/input/etc) AND static text elements (txt)
3. Automatically filter to interactive elements only
4. Only show the disambiguation error if multiple interactive elements match, or if ALL matches are the same role category

This makes `desktop-cli click --text "3" --app "Calculator"` work without `--roles "btn"`, while still catching ambiguous cases like two "Submit" buttons.

Add a `--prefer-role` flag for explicit control: `--prefer-role "btn"` would prefer buttons over links when both match.

## Reproduction

```bash
# Open Calculator app
desktop-cli focus --app "Calculator"

# This currently FAILS without --roles:
desktop-cli click --text "3" --app "Calculator"

# This currently works but is verbose:
desktop-cli click --text "3" --roles "btn" --app "Calculator"

# With the fix, the first command should work automatically
```

Try entering any multi-digit number in Calculator — each digit requires `--roles "btn"` disambiguation.
