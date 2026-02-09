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
desktop-cli read --app "Safari" --text "Submit"                    # search by text (title/value/description)
desktop-cli read --app "Safari" --text "Save" --roles "btn"        # combine text + role filter
desktop-cli read --app "Safari" --roles "btn" --flat               # flat list with path breadcrumbs
desktop-cli read --app "Safari" --text "Submit" --flat             # find element by text, flat output
desktop-cli read --app "Safari" --focused                          # get only the focused element
desktop-cli read --app "Safari" --focused --flat                   # focused element with path breadcrumb
desktop-cli read --app "Safari" --focused --roles "input"          # focused element only if it's an input
desktop-cli read --app "Chrome" --text "Subject" --flat --prune    # prune empty groups (5-8x fewer elements)
desktop-cli read --app "Chrome" --depth 4 --prune                  # prune works in tree mode too
```

Returns compact YAML with short keys: `i` (id), `r` (role), `t` (title), `v` (value), `d` (description), `b` (bounds), `c` (children), `a` (actions), `p` (path, flat mode only).

### Click an element

```bash
desktop-cli click --text "Submit" --app "Safari"                  # click by text (preferred)
desktop-cli click --text "Save" --roles "btn" --app "Safari"      # text + role filter
desktop-cli click --id 5 --app "Safari"                           # click by element ID
desktop-cli click --x 100 --y 200                                 # click at coordinates
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
desktop-cli type --target "Search" --app "Safari" --text "search query"       # find element by text, then type
desktop-cli type --target "Address" --roles "input" --app "Safari" --text "https://example.com"
desktop-cli type --id 4 --app "Safari" --text "search query"                  # find element by ID, then type
```

Responses automatically include target/focused element info — no separate `read` needed after typing:

When typing with `--target` or `--id`, the response includes the `target` element with its updated value:

```yaml
ok: true
action: type
text: hello
target:
    i: 42
    r: input
    t: Subject
    v: hello
    b: [550, 792, 421, 20]
```

When typing without a target, the response includes the `focused` element that received input:

```yaml
ok: true
action: type
text: hello
focused:
    i: 42
    r: input
    t: Subject
    v: hello
    b: [550, 792, 421, 20]
```

Key presses (`--key`) return the currently focused element after the key press:

```yaml
ok: true
action: key
key: tab
focused:
    i: 55
    r: input
    t: Message Body
    b: [550, 830, 421, 200]
```

### Drag

```bash
desktop-cli drag --from-x 100 --from-y 200 --to-x 400 --to-y 300
desktop-cli drag --from-text "Document.pdf" --to-text "Trash" --app "Finder"
desktop-cli drag --from-id 3 --to-id 7 --app "Finder"
```

### Perform accessibility actions

```bash
desktop-cli action --text "Submit" --app "Safari"                              # press by text (preferred)
desktop-cli action --text "Save" --roles "btn" --app "Safari"                  # text + role filter
desktop-cli action --id 5 --app "Safari"                                       # press by element ID
desktop-cli action --id 5 --action press --app "Safari"
desktop-cli action --id 12 --action increment --app "System Settings"
desktop-cli action --id 12 --action decrement --app "System Settings"
desktop-cli action --id 8 --action showMenu --app "Finder"
```

Responses include the target element info — no separate `read` needed:

```yaml
ok: true
action: action
id: 89
name: press
target:
    i: 89
    r: btn
    t: Submit
    b: [200, 400, 100, 32]
```

### Set element values

```bash
desktop-cli set-value --text "Search" --value "hello world" --app "Safari"     # find by text, set value
desktop-cli set-value --id 4 --value "hello world" --app "Safari"              # find by ID, set value
desktop-cli set-value --id 12 --value "75" --app "System Settings"
desktop-cli set-value --id 4 --value "" --app "Safari"
desktop-cli set-value --id 4 --attribute focused --value "true" --app "Safari"
```

### Observe UI changes

```bash
desktop-cli observe --app "Safari"                                     # stream UI diffs as JSONL
desktop-cli observe --app "Safari" --roles "btn,lnk" --interval 500    # watch specific roles, fast poll
desktop-cli observe --app "Safari" --duration 10                       # observe for 10 seconds then stop
desktop-cli observe --app "Safari" --ignore-bounds --ignore-focus      # reduce noise
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
desktop-cli scroll --direction down --text "Web Content" --app "Safari"       # scroll within element by text
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
2. **If you know the element text** — act directly without reading:
   - `action --text "Submit" --app <name>` to press buttons (preferred — works on occluded elements)
   - `click --text "Submit" --app <name>` to click elements by text
   - `set-value --text "Search" --value "..." --app <name>` to set text fields by label
   - `type --target "Search" --app <name> --text "..."` to type into fields by label
   - Add `--roles "btn"` to disambiguate when multiple elements match the text
   - **No follow-up `read` needed** — `type` and `action` responses include target/focused element info
3. **If you need to explore the UI** — read first, then act by ID:
   - `read --app <name> --depth 3 --roles "btn,lnk,input,txt"` to get the element tree as YAML
   - `read --app <name> --text "Submit" --flat` to find a specific element efficiently
   - `read --app <name> --focused` to check which element has focus
   - Use the element `i` (id) field with `action --id <id>`, `click --id <id>`, `set-value --id <id>`, or `type --id <id>`
4. `wait --app <name> --for-text "..." --timeout 10` to wait for a known condition, OR
   `observe --app <name> --duration 10` to stream UI diffs (token-efficient for open-ended monitoring)
5. Repeat act/wait loop as needed
6. If the accessibility tree lacks detail, use `screenshot --app <name>` as a vision model fallback

## Known Limitations

### Contenteditable / Rich-Text Body Fields (Chrome)

Chrome's `contenteditable` divs (e.g. Gmail compose body) may not expose typed text through `AXValue`. The reader uses `AXStringForRange` as a fallback, which works for many text elements. If the body text still doesn't appear in the `v` (value) field, use one of these workarounds:

- **Screenshot + vision model** — `screenshot --app "Google Chrome"` and inspect visually
- **Clipboard verification** — Select all (`Cmd+A`), copy (`Cmd+C`), then read clipboard to verify content
- **Trust the type command** — For simple cases, trust that `type --text "..."` succeeded without verification

Standard `<input>` and `<textarea>` fields (e.g. Subject, To) are always readable.

## App-Specific Skills

See the `skills/` folder for step-by-step guides for specific apps:

- [Gmail](skills/gmail.md) — compose and send emails in Chrome

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
| `p` | Path breadcrumb (flat mode only, e.g. `window > toolbar > btn`) |
