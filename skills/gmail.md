# Gmail (Chrome)

## Compose and send an email

1. Focus Chrome and navigate to Gmail:

```bash
desktop-cli focus --app "Google Chrome"
desktop-cli click --text "Address and search bar" --app "Google Chrome"
desktop-cli type --text "gmail.com"
desktop-cli type --key "enter"
desktop-cli wait --app "Google Chrome" --for-text "Compose" --for-role "btn" --timeout 15
```

2. Open compose and fill fields using Tab to navigate between them:

```bash
desktop-cli click --text "Compose" --roles "btn" --app "Google Chrome"
```

3. The "To" field is focused automatically. Type the recipient then Tab to confirm and advance:
   - **Tip:** The To field has role `other` (not `input`) and description "To recipients" — do not use `--for-role "input"` when waiting for it.
   - **Tip:** Searching `--text "To"` returns massive results because "To" is a common substring. Use `--text "Recipients"` instead to find compose dialog fields efficiently.

```bash
desktop-cli type --text "user@example.com"
desktop-cli type --key "tab"                    # confirm recipient autocomplete
desktop-cli type --key "tab"                    # advance to Subject field
desktop-cli type --text "My subject"
desktop-cli type --key "tab"                    # advance to Message Body
desktop-cli type --text "My message body"
```

4. Send the email:

```bash
desktop-cli click --text "Send" --roles "btn" --app "Google Chrome"
desktop-cli wait --app "Google Chrome" --for-text "Compose: New Message" --gone --timeout 5
```

## Known quirks

- The compose body is a `contenteditable` div. Typed text may not appear in the `v` (value) field via the accessibility tree. For verification, use a screenshot or clipboard check. See the "Contenteditable / Rich-Text Body Fields" section in the main SKILL.md.
