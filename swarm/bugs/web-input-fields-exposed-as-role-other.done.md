# Web app input fields exposed as role "other" instead of "input"

## Problem

In Chrome, some input-like fields (e.g. Gmail's "To" recipients field) are exposed with role `other` instead of `input` in the accessibility tree. This means `--roles "input"` filtering misses them, and `wait --for-role "input"` times out.

## Reproduction

```bash
# This times out — the To field has role "other", not "input"
desktop-cli wait --app "Google Chrome" --for-text "To" --for-role "input" --timeout 10

# The actual element:
# i: 3340, r: other, d: "To recipients"
```

## Expected Behavior

This is fundamentally a Chrome/web accessibility issue, not a desktop-cli bug. However, the CLI could help by:
- Documenting this limitation in SKILL.md (note that web app fields may not have role "input")
- Potentially adding a meta-role like `--roles "interactive"` that matches input, other, list, etc. — any element that is likely to accept user input

## Impact

Agents that filter by `--roles "input"` to find form fields will miss web app fields that Chrome exposes as `other`. This leads to failed waits and unnecessary read/search cycles.
