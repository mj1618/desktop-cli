# desktop-cli

A CLI tool for AI agents to read and interact with desktop UI elements.

## Installation

```bash
go install github.com/mj1618/desktop-cli@latest
```

Or download a binary from the [releases page](https://github.com/mj1618/desktop-cli/releases/latest).

**macOS**: Grant Accessibility permission in System Settings > Privacy & Security > Accessibility.
**macOS** (for screenshot): Grant Screen Recording permission in System Settings > Privacy & Security > Screen Recording.

## Quick Reference

### List windows

```bash
desktop-cli list --windows
desktop-cli list --apps
desktop-cli list --app "Safari"
desktop-cli list --pid 1234
desktop-cli list --pretty
```

### Read UI elements from a window

```bash
desktop-cli read --app "Safari" --depth 4 --roles "btn,lnk,input,txt"
desktop-cli read --app "Finder"
desktop-cli read --app "Safari" --window "GitHub"
desktop-cli read --pid 1234
desktop-cli read --window-id 5678
desktop-cli read --app "Finder" --bbox "0,0,800,600"
```

Returns compact YAML with short keys: `i` (id), `r` (role), `t` (title), `v` (value), `d` (description), `b` (bounds), `c` (children), `a` (actions).

### Click an element

```bash
desktop-cli click --id 5 --app "Safari"
desktop-cli click --x 100 --y 200
desktop-cli click --x 100 --y 200 --button right
desktop-cli click --x 100 --y 200 --double
```

### Type text or key combos

```bash
desktop-cli type "hello world"
desktop-cli type --text "hello world"
desktop-cli type --text "hello" --delay 50
desktop-cli type --key "cmd+c"
desktop-cli type --key "ctrl+shift+t"
desktop-cli type --key "enter"
desktop-cli type --id 4 --app "Safari" --text "search query"
```

### Drag

```bash
desktop-cli drag --from-x 100 --from-y 200 --to-x 400 --to-y 300
desktop-cli drag --from-id 3 --to-id 7 --app "Finder"
```

### Perform accessibility actions

```bash
desktop-cli action --id 5 --app "Safari"
desktop-cli action --id 5 --action press --app "Safari"
desktop-cli action --id 12 --action increment --app "System Settings"
desktop-cli action --id 12 --action decrement --app "System Settings"
desktop-cli action --id 8 --action showMenu --app "Finder"
```

### Wait for UI conditions

```bash
desktop-cli wait --app "Safari" --for-text "Submit" --for-role "btn"
desktop-cli wait --app "Safari" --for-text "Loading..." --gone
desktop-cli wait --app "Chrome" --for-role "input" --timeout 10
desktop-cli wait --app "Safari" --for-id 5
```

### Focus a window

```bash
desktop-cli focus --app "Safari"
desktop-cli focus --app "Safari" --window "GitHub"
desktop-cli focus --pid 1234
desktop-cli focus --window-id 5678
```

### Scroll

```bash
desktop-cli scroll --direction down
desktop-cli scroll --direction up --amount 10
desktop-cli scroll --direction left --amount 5
desktop-cli scroll --direction down --x 500 --y 400
desktop-cli scroll --direction down --id 6 --app "Safari"
```

### Screenshot

```bash
desktop-cli screenshot                                          # full screen, base64 PNG to stdout
desktop-cli screenshot --app "Safari"                           # specific app's window
desktop-cli screenshot --window "GitHub"                        # by window title
desktop-cli screenshot --app "Safari" --output /tmp/safari.png  # save to file
desktop-cli screenshot --app "Safari" --scale 1.0               # full resolution
desktop-cli screenshot --app "Safari" --scale 0.25              # quarter resolution
desktop-cli screenshot --format jpg --quality 60                # JPEG output
desktop-cli screenshot --window-id 5678                         # by window ID
desktop-cli screenshot --pid 1234                               # by PID
```

## Agent Workflow

1. `list --windows` to find the target window
2. `read --app <name> --depth 3 --roles "btn,lnk,input,txt"` to get the element tree as YAML
3. Use the element `i` (id) field with:
   - `action --id <id> --app <name>` to press buttons, toggle checkboxes, etc. (preferred — works on occluded elements)
   - `click --id <id> --app <name>` to click at coordinates (fallback when action isn't available)
   - `type --id <id> --app <name> --text "..."` to type into fields
4. `wait --app <name> --for-text "..." --timeout 10` to wait for UI to update
5. Repeat read/act/wait loop as needed
6. If the accessibility tree lacks detail, use `screenshot --app <name>` as a vision model fallback

## YAML Output Keys

| Key | Meaning |
|-----|---------|
| `i` | Element ID (integer, stable within one read) |
| `r` | Role: `btn`, `txt`, `lnk`, `img`, `input`, `chk`, `radio`, `menu`, `menuitem`, `tab`, `list`, `row`, `cell`, `group`, `scroll`, `toolbar`, `web`, `window`, `other` |
| `t` | Title / label text |
| `v` | Current value |
| `d` | Accessibility description / alt-text |
| `b` | Bounds as `[x, y, width, height]` |
| `f` | Focused (boolean, omitted when false) |
| `e` | Enabled (boolean, omitted when true) |
| `s` | Selected (boolean, omitted when false) |
| `c` | Children (array of elements) |
| `a` | Available actions |
