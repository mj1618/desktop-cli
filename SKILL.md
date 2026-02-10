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

Smart defaults are applied automatically when output is piped (agent context):
- **Format**: auto-selects `agent` format (compact, one element per line)
- **Prune**: auto-enables `--prune` for web apps (removes empty divs, 5-8x less output)
- **Roles**: auto-expands `--roles "input"` to include "other" for web apps

Use `--raw` to disable all smart defaults. Use `--format yaml` to force YAML output.

```bash
desktop-cli read --app "Chrome"                                    # smart defaults: agent format, auto-prune for web
desktop-cli read --app "Finder"                                    # smart defaults: agent format (no prune, not web)
desktop-cli read --app "Safari" --depth 4 --roles "btn,lnk,input,txt"
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
desktop-cli read --app "Chrome" --text "Subject" --flat            # prune is auto-applied for web apps
desktop-cli read --app "Chrome" --scope-id 156 --depth 3           # read only descendants of element 156
desktop-cli read --app "Chrome" --scope-id 156                     # agent format scoped to a container
desktop-cli read --app "Chrome" --text "Results" --children        # direct children of matched element (e.g. list items)
desktop-cli read --app "Chrome" --scope-id 156 --children          # direct children only (no grandchildren)
desktop-cli read --app "Chrome" --raw --format yaml                # disable smart defaults, force YAML
```

Returns compact YAML with short keys: `i` (id), `r` (role), `t` (title), `v` (value), `d` (description), `b` (bounds), `c` (children), `a` (actions), `p` (path, flat mode only).

**Agent format** (`--format agent`): Ultra-compact output for AI agents — shows interactive elements plus display text (read-only text with values, e.g. Calculator display), one per line: `[id] role "label" (x,y,w,h) [flags]`. Display text elements are marked with `display` flag. Elements with zero-width or zero-height bounds (off-screen/virtualized) are automatically excluded. Typically 20-30x fewer lines than YAML. Use element IDs with `click --id`, `action --id`, etc.

### Click an element

```bash
desktop-cli click --text "Submit" --app "Safari"                  # click by text (preferred)
desktop-cli click --text "Save" --roles "btn" --app "Safari"      # text + role filter
desktop-cli click --id 5 --app "Safari"                           # click by element ID
desktop-cli click --x 100 --y 200                                 # click at coordinates
desktop-cli click --x 100 --y 200 --button right
desktop-cli click --x 100 --y 200 --double
desktop-cli click --text "Buy milk" --near --app "Notes"              # click nearest interactive element to text
desktop-cli click --text "Buy milk" --near --near-direction left --app "Notes"  # search left only (for checkboxes)
desktop-cli click --text "Submit" --app "Safari" --post-read          # click and include full UI state in response
desktop-cli click --text "Submit" --app "Safari" --post-read --post-read-delay 500  # wait 500ms before reading state
```

### Type text or key combos

```bash
desktop-cli type "hello world"
desktop-cli type --text "hello world"
desktop-cli type --text "hello" --delay 50
desktop-cli type --key "cmd+c"
desktop-cli type --key "ctrl+shift+t"
desktop-cli type --key "enter"
desktop-cli type --text "gmail.com" --key "enter"                              # type text then press key in one call
desktop-cli type --text "matt@example.com" --key "tab"                         # type text then press tab
desktop-cli type --target "Search" --app "Safari" --text "search query"        # find element by text, then type
desktop-cli type --target "Address" --roles "input" --app "Safari" --text "https://example.com"
desktop-cli type --target "Address" --app "Safari" --text "https://example.com" --key "enter"  # type into target then press key
desktop-cli type --id 4 --app "Safari" --text "search query"                   # find element by ID, then type
desktop-cli type --app "Calculator" --text "347*29+156="                       # type full expression into Calculator (1 command instead of 11)
```

Responses automatically include target/focused element info and display elements (when `--app` is specified) — no separate `read` needed after typing:

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

When both `--text` and `--key` are provided, text is typed first, then the key is pressed:

```yaml
ok: true
action: type+key
text: gmail.com
key: enter
focused:
    i: 55
    r: input
    t: Address
    b: [550, 792, 421, 20]
```

When `--app` is specified, the response also includes `display` elements (read-only text with values, e.g. Calculator display), capped at 20 elements to avoid excessive output. Use `--no-display` to skip display collection entirely. When multiple display elements exist, the most prominent one (largest font) is marked `primary: true`:

```yaml
ok: true
action: type
text: "347*29+156="
display:
    - i: 9
      r: txt
      v: "347×29+156"
      b: [624, 731, 124, 26]
    - i: 11
      r: txt
      v: "10219"
      primary: true
      b: [664, 761, 83, 36]
```

### Batch multiple actions (`do`)

```bash
# Form fill — all steps in 1 CLI call instead of 8
desktop-cli do --app "Safari" <<'EOF'
- click: { text: "Full Name" }
- type: { text: "John Doe" }
- type: { key: "tab" }
- type: { text: "john@example.com" }
- click: { text: "Submit" }
- wait: { for-text: "Thank you", timeout: 10 }
EOF

# Multi-app workflow
desktop-cli do <<'EOF'
- focus: { app: "Safari" }
- click: { text: "Address", app: "Safari" }
- type: { text: "https://example.com", key: "enter" }
- wait: { for-text: "Example Domain", app: "Safari", timeout: 10 }
EOF

# With sleep for animations
desktop-cli do --app "System Settings" <<'EOF'
- click: { text: "General" }
- sleep: { ms: 500 }
- click: { text: "About" }
EOF
```

Steps: `click`, `type`, `action`, `set-value`, `scroll`, `wait`, `focus`, `read`, `sleep`. Each step is `{ command: { params } }`. `--app`/`--window` set defaults; per-step `app`/`window` override. `--stop-on-error` (default: true) stops on first failure.

### Find elements across windows

```bash
desktop-cli find --text "Save As"                       # search all windows
desktop-cli find --text "Allow" --roles "btn"            # with role filter
desktop-cli find --text "Save" --app "Safari"            # limit to one app
desktop-cli find --text "OK" --limit 5                   # cap results
desktop-cli find --text "Submit" --exact                 # exact match
```

Searches all windows for matching elements, grouped by window. Focused windows are searched first. Use when a dialog or notification appeared and you don't know which app owns it.

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

Responses include the target element info and any display elements (read-only text with values, e.g. Calculator display) — no separate `read` needed:

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
display:
    - i: 9
      r: txt
      v: "42"
      b: [686, 761, 62, 36]
```

The `display` field is omitted when no display elements exist. The `click` command also includes display elements in its response when `--app` or `--window` is specified. Display elements are capped at 20 to prevent excessive output in apps with many text elements. Use `--no-display` on `click`, `action`, or `type` to skip display collection entirely.

### Set element values

```bash
desktop-cli set-value --text "Search" --value "hello world" --app "Safari"     # find by text, set value
desktop-cli set-value --id 4 --value "hello world" --app "Safari"              # find by ID, set value
desktop-cli set-value --id 12 --value "75" --app "System Settings"
desktop-cli set-value --id 4 --value "" --app "Safari"
desktop-cli set-value --id 4 --attribute focused --value "true" --app "Safari"
```

### Clipboard

```bash
desktop-cli clipboard read                                             # read current clipboard text
desktop-cli clipboard write "Hello, world"                             # write text to clipboard
desktop-cli clipboard write --text "Hello, world"                      # write text (flag form)
desktop-cli clipboard clear                                            # clear the clipboard
desktop-cli clipboard grab --app "Safari"                              # select-all + copy + read from app
desktop-cli clipboard grab --app "Chrome" --window "Gmail"             # grab from specific window
```

Use `clipboard grab` when the accessibility tree doesn't expose content (e.g. contenteditable fields, Gmail compose body). Use `clipboard read` after manual `Cmd+C` to verify copied content.

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
desktop-cli screenshot --app "Safari" --include-menubar         # include menu bar in app screenshot
desktop-cli screenshot --format jpg --quality 60                # JPEG output
desktop-cli screenshot --window-id 5678                         # by window ID
desktop-cli screenshot --pid 1234                               # by PID
```

### Screenshot with Coordinates

```bash
desktop-cli screenshot-coords --app "Safari" --output /tmp/coords.png             # annotate interactive elements
desktop-cli screenshot-coords --app "Safari" --all-elements --output /tmp/all.png # annotate all elements
desktop-cli screenshot-coords --app "Safari" --roles "btn,lnk" --output /tmp/buttons.png  # specific roles
desktop-cli screenshot-coords --app "Safari" --text "Search" --output /tmp/search.png     # filter by text
```

Annotates screenshot with red bounding boxes and (x,y) coordinate labels. Helps visualize where UI elements are located.

## Agent Workflow

1. `list --windows` to find the target window
2. **If you know the element text** — act directly without reading:
   - `action --text "Submit" --app <name>` to press buttons (preferred — works on occluded elements)
   - `click --text "Submit" --app <name>` to click elements by text
   - `set-value --text "Search" --value "..." --app <name>` to set text fields by label
   - `type --target "Search" --app <name> --text "..."` to type into fields by label
   - Interactive elements (btn, lnk, input, etc.) are auto-preferred over static text when multiple elements match — e.g. `click --text "3" --app "Calculator"` clicks the button, not the display text
   - Add `--roles "btn"` to disambiguate further when multiple interactive elements match
   - Add `--exact` for exact match instead of substring (e.g. match "Subject" but not "Test Subject")
   - Add `--scope-id <id>` to limit text search to descendants of a specific element (e.g. a dialog)
   - For Calculator: `type --app "Calculator" --text "347*29+156="` types the full expression in 1 command (instead of 11 individual button presses)
   - **No follow-up `read` needed** — `type`, `action`, and `click` responses include target/focused element info and display elements (e.g. Calculator display value)
   - Add `--post-read` to include full UI state (agent format) in the response — eliminates a follow-up `read` call to see what changed after the action
   - Add `--post-read-delay 500` to wait before reading state (for page transitions or animations)
3. **If you need to explore the UI** — read first, then act by ID:
   - `read --app <name>` to get a compact list of all clickable elements (agent format auto-applied when piped, web apps auto-pruned)
   - `read --app <name> --format yaml --depth 3 --roles "btn,lnk,input,txt"` to get the element tree as YAML
   - `read --app <name> --text "Submit" --flat` to find a specific element efficiently
   - `read --app <name> --focused` to check which element has focus
   - Use the element `i` (id) field with `action --id <id>`, `click --id <id>`, `set-value --id <id>`, or `type --id <id>`
4. `wait --app <name> --for-text "..." --timeout 10` to wait for a known condition, OR
   `observe --app <name> --duration 10` to stream UI diffs (token-efficient for open-ended monitoring)
5. **For multi-step sequences** — use `do` to batch actions in one call:
   - `do --app <name> <<'EOF'` + YAML list of steps eliminates LLM round-trips between actions
   - Each step is `{ command: { params } }` — supports all action types
   - Ideal for form fills, menu navigation, and Calculator expressions
6. **If a dialog/notification appeared** and you don't know which app owns it:
   - `find --text "Allow" --roles "btn"` to search all windows for the element
   - Results include the app name, window title, PID, and element details so you can act immediately
7. Repeat act/wait loop as needed
8. If the accessibility tree lacks detail:
   - Use `clipboard grab --app <name>` to select-all + copy + read text from any app (works for contenteditable, rich text fields)
   - Use `clipboard read` after `type --key "cmd+c"` to verify copied content
   - Use `click --text "label" --near --app <name>` to click the nearest interactive element to a text label (e.g. checkboxes in Apple Notes)
   - Use `screenshot --app <name>` for raw screenshot as vision model fallback
   - Use `screenshot --app <name> --include-menubar` to capture the app's menu bar along with the window (useful when menu items aren't in the accessibility tree)
   - Use `screenshot-coords --app <name>` to see coordinates and identify element positions visually

## Known Limitations

### Web Input Fields May Have Role "other" (Chrome)

In Chrome, some input-like fields (e.g. Gmail's "To" recipients field) are exposed with role `other` instead of `input` in the accessibility tree. Use `--roles "interactive"` instead of `--roles "input"` to match these fields. The `interactive` meta-role expands to `input,other,chk,toggle,radio,list` — covering form elements that Chrome may misreport and native toggle switches in System Settings.

```bash
# Instead of this (may miss web app fields):
desktop-cli wait --app "Google Chrome" --for-text "To" --for-role "input"

# Use this:
desktop-cli wait --app "Google Chrome" --for-text "To" --for-role "interactive"

# Also works with --roles:
desktop-cli read --app "Google Chrome" --roles "interactive" --flat
```

### Apple Notes Checklist Checkboxes

Apple Notes checklist checkboxes are not exposed in the macOS accessibility tree. The checklist text items appear merged together in static text elements, and the checkboxes themselves are invisible to accessibility queries. Use `--near` with `click --text` to click the nearest interactive element to a text label. When no interactive element is found in the tree, `--near` automatically clicks to the left of the text (where checkboxes typically are). Use `--near-direction` to control the search direction:

```bash
# Use --near to click the checkbox nearest to the text label (prefers left by default)
desktop-cli click --text "Buy milk" --near --app "Notes"

# Explicit direction: search left for checkbox
desktop-cli click --text "Buy milk" --near --near-direction left --app "Notes"

# Or use screenshot + coordinate-based clicking as fallback
desktop-cli screenshot-coords --app "Notes" --output /tmp/notes.png
desktop-cli click --x <x> --y <y>
```

### Contenteditable / Rich-Text Body Fields (Chrome)

Chrome's `contenteditable` divs (e.g. Gmail compose body) may not expose typed text through `AXValue`. The reader uses `AXStringForRange` as a fallback, which works for many text elements. If the body text still doesn't appear in the `v` (value) field, use one of these workarounds:

- **Clipboard grab** — `clipboard grab --app "Google Chrome"` selects all, copies, and reads clipboard in one command
- **Clipboard verification** — `clipboard read` after a manual `type --key "cmd+c"` to verify copied content
- **Screenshot + vision model** — `screenshot --app "Google Chrome"` and inspect visually
- **Trust the type command** — For simple cases, trust that `type --text "..."` succeeded without verification

Standard `<input>` and `<textarea>` fields (e.g. Subject, To) are always readable.

## YAML Output Keys

| Key | Meaning |
|-----|---------|
| `i` | Element ID (integer, stable within one read) |
| `r` | Role: `btn`, `txt`, `lnk`, `img`, `input`, `chk`, `toggle`, `radio`, `menu`, `menuitem`, `tab`, `list`, `row`, `cell`, `group`, `scroll`, `toolbar`, `web`, `window`, `other` |
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
