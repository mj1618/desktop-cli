# Feature: `open` Command — Open URLs, Files, and Apps

## Problem

Agents frequently need to open URLs in a browser, launch applications, or open files. Currently this requires a multi-step workflow:

1. `focus --app "Safari"` or `focus --app "Google Chrome"`
2. `click --text "Address" --app "Safari"` (or similar to focus the address bar)
3. `type --key "cmd+a"` to select existing URL
4. `type --text "https://example.com" --key "enter"`
5. `wait --app "Safari" --for-text "..." --timeout 10`

This is 5 CLI calls with LLM round-trips between each. It's fragile (the address bar text varies by browser), slow, and wastes tokens.

## Solution

Add a `desktop-cli open` command that wraps macOS `open(1)` with agent-friendly YAML output:

```bash
# Open a URL in the default browser
desktop-cli open "https://example.com"
desktop-cli open --url "https://example.com"

# Open a URL in a specific browser
desktop-cli open --url "https://example.com" --app "Google Chrome"

# Open a file with its default app
desktop-cli open "/path/to/document.pdf"
desktop-cli open --file "/path/to/document.pdf"

# Open a file with a specific app
desktop-cli open --file "/path/to/image.png" --app "Preview"

# Launch an application
desktop-cli open --app "Calculator"
desktop-cli open --app "System Settings"

# Open and wait for the window to appear (combines open + wait)
desktop-cli open --url "https://example.com" --wait --timeout 10

# Open and read the resulting UI state
desktop-cli open --url "https://example.com" --post-read --post-read-delay 2000
```

## Output Format

```yaml
ok: true
action: open
url: "https://example.com"
app: "Safari"
pid: 12345
```

With `--post-read`:
```yaml
ok: true
action: open
url: "https://example.com"
app: "Safari"
pid: 12345
state: |
    # Example Domain - Safari (pid: 12345)
    [1] input "Address and search bar" (100,52,800,22) val="https://example.com"
    [15] txt "Example Domain" (300,200,200,30) display
    [20] lnk "More information..." (300,300,180,18)
```

## Implementation Plan

1. **New file**: `cmd/open.go`
   - Register `openCmd` cobra command under `rootCmd`
   - Positional arg treated as URL if starts with `http://` or `https://`, else as file path
   - `--url`, `--file` flags for explicit mode
   - `--app` to specify which app to use
   - `--wait` + `--timeout` to wait for the app window to appear
   - `--post-read` + `--post-read-delay` to read UI state after opening

2. **Implementation**:
   - Use `exec.Command("open", args...)` on macOS — this is the standard way to open URLs/files/apps
   - For `--app`, use `open -a "AppName"`
   - For URLs, use `open "https://..."` or `open -a "AppName" "https://..."`
   - For files, use `open "/path/to/file"` or `open -a "AppName" "/path/to/file"`
   - After `open`, optionally wait for the window to appear using existing `wait` logic
   - Optionally read post-action state using existing `readPostActionState` helper

3. **Platform consideration**:
   - macOS: use `open` command
   - Linux: use `xdg-open` command
   - Wrap in platform provider interface for cross-platform support

4. **Update docs**: Add to README.md, SKILL.md, and agent workflow section

## Value

- **5x fewer CLI calls** for URL navigation (1 call instead of 5)
- **More reliable** — no fragile address bar targeting that varies by browser
- **Faster** — single `exec.Command` vs. multiple accessibility tree reads + mouse movements
- **Works for any URL/file/app** — not just browsers
- **Composable** — `--post-read` gives agents the page state without a follow-up `read` call

## Complexity

Low — relies on OS-provided `open`/`xdg-open` commands, reuses existing helpers for `--post-read` and `--wait`.
