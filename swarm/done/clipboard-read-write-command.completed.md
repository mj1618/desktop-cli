# Clipboard Read/Write Command

## Problem

Agents frequently need to interact with the system clipboard but currently have no way to do so through desktop-cli. Common scenarios include:

1. **Verifying typed content** — After typing into a contenteditable/rich-text field (e.g. Gmail compose body), the accessibility tree may not expose the typed text. The agent must select-all, copy, then read clipboard to verify.
2. **Extracting text from inaccessible elements** — Some apps don't expose their content via the accessibility tree. The clipboard is the universal fallback for reading text.
3. **Pasting content** — Agents sometimes need to paste pre-formed content (e.g. formatted text, URLs, file paths) rather than typing character by character, which is faster and preserves formatting.
4. **Data transfer between apps** — Copy data from one app and paste into another (e.g. copying a URL from Safari's address bar to paste into a form).

Currently agents must use raw key combos (`type --key "cmd+c"` / `type --key "cmd+v"`) and have **no way to read the clipboard contents** or **set clipboard contents** through desktop-cli. This is a significant gap.

## Proposed Implementation

### Commands

```bash
# Read the current clipboard text content
desktop-cli clipboard read

# Write text to the clipboard
desktop-cli clipboard write "Hello, world"
desktop-cli clipboard write --text "Hello, world"

# Clear the clipboard
desktop-cli clipboard clear

# Convenience: select-all + copy + read clipboard for a given app
# (focuses the app, sends Cmd+A, Cmd+C, then reads clipboard)
desktop-cli clipboard grab --app "Safari"
```

### Response Format

```yaml
# clipboard read
ok: true
action: clipboard-read
text: "The clipboard contents here"

# clipboard write
ok: true
action: clipboard-write

# clipboard grab
ok: true
action: clipboard-grab
text: "All selected text from the app"
```

### Implementation Details

1. **New file**: `cmd/clipboard.go` — Cobra subcommand with `read`, `write`, `clear`, `grab` subcommands
2. **Platform interface**: Add `ClipboardManager` interface to `internal/platform/platform.go`:
   ```go
   type ClipboardManager interface {
       GetText() (string, error)
       SetText(text string) error
       Clear() error
   }
   ```
3. **macOS implementation**: `internal/platform/darwin/clipboard.go` — Use `NSPasteboard` via CGo or shell out to `pbcopy`/`pbpaste` (simpler, reliable)
4. **Provider**: Add `ClipboardManager` to the `Provider` struct

### Why This Is High Value

- **Universal fallback** for reading content from any app, even when the accessibility tree is incomplete
- **Eliminates the biggest verification gap** — agents can now confirm what they typed actually appeared
- **Single command `clipboard grab`** replaces a 3-step sequence (focus + select-all + copy + ...no way to read)
- **Enables new workflows** like programmatic paste of complex content
- **Minimal implementation effort** — `pbcopy`/`pbpaste` on macOS makes the platform layer trivial

### Dependencies

None — this is a standalone feature that doesn't depend on any other pending work.

### Testing

- Unit test clipboard read/write round-trip
- Test `grab` command focuses correct app and returns text
- Test empty clipboard returns empty string (not error)
- Test writing and reading back preserves Unicode and whitespace
