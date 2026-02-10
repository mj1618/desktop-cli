# Gmail Compose Tips

## Quick Compose Script

For fastest email composition, use the helper script:
```bash
./scripts/gmail-compose.sh "recipient@example.com" "Subject" "Body text"
```

This runs in 3-5 seconds and uses the efficient `fill` command (reads UI tree once, fills all fields).

See `scripts/gmail-compose.sh` and `scripts/gmail-compose.yaml` for the implementation.

## Manual Compose Tips

- **Don't rely on stale element IDs in compose**: Gmail's compose interface is highly dynamic. Element IDs shift frequently as the DOM changes. Always click/focus a field explicitly before typing into it, rather than using cached IDs from previous reads.
- **Always click fields to focus before typing**: Use `click: { text: "To", roles: "other" }` to focus the To field, `click: { text: "Subject", roles: "input" }` for Subject, etc. This ensures the next type command targets the correct field.
- **Wait between field transitions**: Add `sleep: { ms: 300 }` after clicking a new field to ensure it's properly focused before typing. Gmail's UI can be slow to update focus.
- **The "To" field has role "other"**: In Chrome/Gmail, the recipient field is exposed as role `other`, not `input`. Use `roles: "other"` when targeting it.
- **Subject and Message Body are role "input"**: These standard fields have role `input` and can be targeted by their description text.
- **Verify email was added to To field**: After typing an email address, verify it was accepted by checking the element's value before proceeding. Autocomplete suggestions can interfere if not properly dismissed.
- **Use fill command for multiple fields**: The `fill` command is 4-8x faster than individual click/type operations because it reads the UI tree once and fills all fields in a single operation.
