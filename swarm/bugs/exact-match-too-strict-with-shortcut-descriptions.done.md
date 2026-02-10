# --exact flag too strict on descriptions containing keyboard shortcuts

## Problem

Using `--exact` with `--text "Send"` fails to match an element whose description is `"Send ‪(⌘Enter)‬"`. Many UI elements (especially buttons) include keyboard shortcut hints appended to their description, so exact matching on the full description string is overly strict.

## Reproduction

```bash
# This fails — no element found
desktop-cli action --text "Send" --roles "btn" --app "Google Chrome" --exact

# This works — substring match
desktop-cli action --text "Send" --roles "btn" --app "Google Chrome"
```

The Send button's description is `"Send ‪(⌘Enter)‬"`, so exact match fails.

## Expected Behavior

`--exact` should either:
- Match against each text field (title, value, description) independently, trimming parenthetical suffixes like `(⌘Enter)`, OR
- Do word-boundary / starts-with matching rather than full-string equality, OR
- At minimum, document this limitation so agents know to avoid `--exact` on buttons

## Impact

Agents use `--exact` to avoid false positives (e.g. matching "Send" but not "Sender"), but it's unreliable for buttons and links that include shortcut hints. This forces agents to fall back to substring matching and deal with potential ambiguity.
