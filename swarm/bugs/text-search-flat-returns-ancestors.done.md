# Text search with --flat returns all ancestor elements

## Problem

When using `--text "X" --flat`, the output includes every ancestor group element up to the root window, not just elements whose title/value/description actually match the search text. This produces massive output (250KB+) that is mostly irrelevant.

## Reproduction

```bash
desktop-cli read --app "Google Chrome" --text "Recipients" --flat
```

Returns thousands of elements including every parent group, window, toolbar, etc. — not just elements containing "Recipients" in their text fields.

## Expected Behavior

With `--text` + `--flat`, only elements whose title (`t`), value (`v`), or description (`d`) actually match the search text should be returned. Ancestor/parent elements that don't match should be excluded.

## Impact

This is the biggest friction point when using the CLI as an agent. Agents get flooded with irrelevant elements and waste tokens parsing huge output, when they only need the 1-3 matching elements.
