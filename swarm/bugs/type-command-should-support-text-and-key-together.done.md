# type command should support --text and --key together

## Summary

The `type` command currently requires separate invocations for typing text and pressing a key. In practice, typing text followed by a key press (Enter, Tab, etc.) is extremely common. Supporting both `--text` and `--key` in a single invocation would eliminate a full CLI round-trip each time.

## Examples from real usage

### Typing a URL and pressing Enter (currently 2 calls)

```bash
desktop-cli type --text "gmail.com"
desktop-cli type --key "enter"
```

Should be possible as:

```bash
desktop-cli type --text "gmail.com" --key "enter"
```

### Typing an email address and pressing Tab to confirm autocomplete (currently 2 calls)

```bash
desktop-cli type --text "matt@supplywise.app"
desktop-cli type --key "tab"
```

Should be possible as:

```bash
desktop-cli type --text "matt@supplywise.app" --key "tab"
```

## Behavior

When both `--text` and `--key` are provided, the command should:

1. Type the text characters first
2. Then press the key combo
3. Return the focused element info as usual

## Impact

In a typical Gmail compose-and-send workflow (15 CLI calls), this would eliminate 2 round-trips. In more complex workflows with many form fields, the savings multiply. Each saved round-trip eliminates ~700ms of CLI overhead plus agent reasoning time.
