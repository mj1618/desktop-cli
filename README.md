# Desktop CLI

A command-line tool that lets AI agents read and interact with desktop UI elements. Agents get a structured YAML snapshot of any window's accessibility tree, then issue commands to click, type, scroll, and more — all from the terminal.

## Features

- **Read UI elements** — Get a compact YAML tree of buttons, links, text fields, and other elements in any window
- **Click elements** — Click by element ID or screen coordinates
- **Type text** — Simulate keyboard input and key combinations
- **Focus windows** — Bring windows to the foreground
- **Scroll & drag** — Scroll within windows or drag between points
- **Screenshot** — Capture windows for vision model fallback
- **Clipboard** — Read, write, and clear the system clipboard; grab selected text from any app
- **Token efficient** — Short YAML keys, role abbreviations, and filtering to minimize agent token usage
- **Fast** — Native accessibility APIs via CGo, no OCR or screenshots in the critical path

## Requirements

- **macOS**: Grant Accessibility permission in System Settings > Privacy & Security > Accessibility
- **macOS** (for screenshot): Grant Screen Recording permission in System Settings > Privacy & Security > Screen Recording
- Go 1.22+ (for building from source)

## Installation

### Download Binary (Recommended)

Download the latest binary for your platform from the [releases page](https://github.com/mj1618/desktop-cli/releases/latest).

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_darwin_arm64.tar.gz | tar xz
sudo mv desktop-cli /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_darwin_amd64.tar.gz | tar xz
sudo mv desktop-cli /usr/local/bin/
```

**Linux (x64):**
```bash
curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_linux_amd64.tar.gz | tar xz
sudo mv desktop-cli /usr/local/bin/
```

**Linux (ARM64):**
```bash
curl -L https://github.com/mj1618/desktop-cli/releases/download/latest/desktop-cli_linux_arm64.tar.gz | tar xz
sudo mv desktop-cli /usr/local/bin/
```

### Install with Go

```bash
go install github.com/mj1618/desktop-cli@latest
```

### Build from Source

```bash
git clone https://github.com/mj1618/desktop-cli.git
cd desktop-cli
go build -ldflags "-X github.com/mj1618/desktop-cli/internal/version.Commit=$(git rev-parse --short HEAD) -X github.com/mj1618/desktop-cli/internal/version.BuildDate=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" -o desktop-cli .
codesign --force --sign - ./desktop-cli   # required on macOS Apple Silicon
sudo mv desktop-cli /usr/local/bin/
```

Or use the update script which handles ldflags and build verification automatically:

```bash
./update.sh
```

## Usage

### List windows

```bash
# List all windows with app name, title, PID, bounds, and focused state
desktop-cli list --windows

# List running applications (unique app names with PIDs)
desktop-cli list --apps

# Filter by app name
desktop-cli list --app "Safari"

# Filter by PID
desktop-cli list --pid 1234

# Pretty-print output
desktop-cli list --pretty
```

### Read UI elements

```bash
# Read the full element tree for an app
desktop-cli read --app "Finder"

# Limit traversal depth
desktop-cli read --app "Safari" --depth 3

# Filter to specific roles
desktop-cli read --app "Safari" --depth 4 --roles "btn,lnk,input,txt"

# Use "interactive" meta-role to match all input-like elements (input,other,chk,toggle,radio,list)
# Useful for web apps where Chrome exposes fields as "other" instead of "input"
desktop-cli read --app "Chrome" --roles "interactive" --flat

# Target a specific window by title substring
desktop-cli read --app "Safari" --window "GitHub"

# Target by PID or window ID
desktop-cli read --pid 1234
desktop-cli read --window-id 5678

# Filter to a bounding box region (x,y,width,height)
desktop-cli read --app "Finder" --bbox "0,0,800,600"

# Search for elements by text (case-insensitive substring match on title/value/description)
desktop-cli read --app "Safari" --text "Submit"

# Combine text search with role filter
desktop-cli read --app "Safari" --text "Save" --roles "btn"

# Get results as a flat list (no nesting, includes path breadcrumbs)
desktop-cli read --app "Safari" --roles "btn" --flat

# Find a specific element by text as a flat result (most token-efficient)
desktop-cli read --app "Safari" --text "Submit" --flat

# Get only the currently focused element
desktop-cli read --app "Safari" --focused

# Get the focused element as a flat result with path breadcrumb
desktop-cli read --app "Safari" --focused --flat

# Get the focused element only if it matches a role filter
desktop-cli read --app "Safari" --focused --roles "input"

# Prune anonymous group/other elements that have no title/value/description
# (dramatically reduces output size — 5-8x fewer elements for typical web pages)
desktop-cli read --app "Chrome" --text "Subject" --flat --prune

# Prune works with tree mode too (empty groups are removed, children promoted)
desktop-cli read --app "Chrome" --depth 4 --prune

# Scope to descendants of a specific element (e.g. a container or panel)
desktop-cli read --app "Chrome" --scope-id 156 --depth 3
desktop-cli read --app "Chrome" --scope-id 156 --format agent

# Get direct children of a matched element (e.g. list items in a results container)
desktop-cli read --app "Chrome" --text "Results" --children --format agent
desktop-cli read --app "Chrome" --scope-id 156 --children

# Agent format: compact one-line-per-element output showing only clickable elements
# (dramatically reduces output — typically 20-30x fewer lines than YAML)
desktop-cli read --app "Chrome" --format agent
desktop-cli read --pid 1234 --format agent
```

#### Smart Defaults

When output is piped (typical agent context), smart defaults are applied automatically:

- **Auto format**: `--format agent` is used when stdout is piped (not a terminal)
- **Auto prune**: `--prune` is enabled when web content is detected (elements with "web" role)
- **Auto role expansion**: `--roles "input"` auto-includes "other" for web apps (Chrome exposes some inputs as "other")

Applied defaults are shown in the `smart_defaults` response field. Override any default with explicit flags:

```bash
# Smart defaults applied automatically:
desktop-cli read --app "Chrome"
# → auto-prune (web content), agent format (piped output)

# Override: force YAML output
desktop-cli read --app "Chrome" --format yaml

# Override: disable auto-prune
desktop-cli read --app "Chrome" --prune=false

# Disable all smart defaults:
desktop-cli read --app "Chrome" --raw
```

#### Agent Format (`--format agent`)

The agent format produces ultra-compact output designed for AI agent consumption. It shows interactive (clickable) elements plus display text elements that contain values, one per line. Elements with zero-width or zero-height bounds (off-screen or virtualized) are automatically excluded:

```
# Gmail - Google Chrome (pid: 44037)

[11] btn "Back" (917,239,34,34)
[12] btn "Forward" (953,239,34,34) disabled
[16] input "Address and search bar" (1069,244,1180,24) val="mail.google.com/..."
[71] input "Search mail" (1223,302,569,20)
[114] btn "Compose" (920,352,143,56)
[125] lnk "Inbox 23288 unread" (976,431,38,18)
[311] radio "Primary" (1168,392,251,56) selected
```

Each line: `[id] role "label" (x,y,w,h) [flags]`

- **id** — Use with `click --id`, `action --id`, `type --id`, etc.
- **role** — Element type (btn, lnk, input, chk, radio, etc.)
- **label** — Title or accessibility description
- **bounds** — Screen position and size (x, y, width, height)
- **flags** — `disabled`, `selected`, `focused`, `checked`/`unchecked`, `val="..."`, `display` (read-only text with a value)

### Click an element

```bash
# Click by text (finds element matching text, then clicks it)
desktop-cli click --text "Submit" --app "Safari"

# Click by text with role filter (disambiguate when multiple interactive elements match)
desktop-cli click --text "Save" --roles "btn" --app "Safari"

# Exact match (match full title/value/description, not substring)
desktop-cli click --text "Subject" --exact --app "Chrome"

# Scope text search to descendants of a specific element (e.g. a dialog)
desktop-cli click --text "Subject" --scope-id 42 --app "Chrome"

# Click by element ID (re-reads the element tree to get current coordinates)
desktop-cli click --id 5 --app "Safari"

# Click at absolute screen coordinates
desktop-cli click --x 100 --y 200

# Right-click
desktop-cli click --x 100 --y 200 --button right

# Double-click
desktop-cli click --x 100 --y 200 --double

# Click nearest interactive element to a text label (e.g. checkbox near text)
desktop-cli click --text "Buy milk" --near --app "Notes"

# Specify search direction for --near (left, right, above, below)
desktop-cli click --text "Buy milk" --near --near-direction left --app "Notes"

# Click and include full UI state in response (saves a follow-up read call)
desktop-cli click --text "Submit" --app "Safari" --post-read

# Click with delay before reading state (for actions that trigger animations)
desktop-cli click --text "Submit" --app "Safari" --post-read --post-read-delay 500
```

### Type text or key combos

```bash
# Type text (positional arg or --text flag)
desktop-cli type "hello world"
desktop-cli type --text "hello world"

# Type with delay between keystrokes (ms)
desktop-cli type --text "hello" --delay 50

# Press key combinations
desktop-cli type --key "cmd+c"
desktop-cli type --key "ctrl+shift+t"
desktop-cli type --key "enter"

# Type text then press a key in one call (eliminates a round-trip)
desktop-cli type --text "gmail.com" --key "enter"
desktop-cli type --text "matt@example.com" --key "tab"

# Find an element by text, focus it, then type into it
desktop-cli type --target "Search" --app "Safari" --text "search query"

# Find an element by text + role filter, then type into it
desktop-cli type --target "Address" --roles "input" --app "Safari" --text "https://example.com"

# Target an element, type text, and press a key — all in one call
desktop-cli type --target "Address" --app "Safari" --text "https://example.com" --key "enter"

# Click an element by ID to focus it, then type into it
desktop-cli type --id 4 --app "Safari" --text "search query"

# Type a full expression into Calculator (1 command instead of 11 individual button presses)
desktop-cli type --app "Calculator" --text "347*29+156="
```

**Response format:** The `type` command returns information about the target or focused element, plus display elements when `--app` is specified (e.g. Calculator display), eliminating the need for a follow-up `read` call.

When typing into a targeted element (`--target` or `--id`), the response includes a `target` field with the element's current state (including its updated value):

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

When typing without a target (bare `type --text`), the response includes a `focused` field showing which element received the input:

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

For `--key` actions, the response includes the currently focused element after the key press:

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

When both `--text` and `--key` are provided, text is typed first, then the key is pressed. The action is `type+key`:

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

When `--app` is specified, the response also includes `display` elements (read-only text with values, e.g. Calculator display), capped at 20 elements — no follow-up `read` needed to check the result. Use `--no-display` to skip display collection entirely. When multiple display elements exist, the most prominent one (largest font) is marked `primary: true`:

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

**Focus feedback for navigation keys:** When you press navigation keys (`tab`, `shift+tab`, `enter`, `escape`, arrow keys), the response automatically includes a `focused` field showing which element received focus:

```yaml
ok: true
action: key
key: tab
focused:
    i: 3463
    r: input
    d: Subject
    b: [550, 792, 421, 20]
```

This eliminates the need for a separate `read` call to find where focus moved. If no element has focus after the key press, the `focused` field is omitted. Non-navigation keys (e.g. `cmd+c`) return only `ok`, `action`, and `key` as before.

### Scroll

```bash
# Scroll down 3 lines (default) at current mouse position
desktop-cli scroll --direction down

# Scroll up 10 lines
desktop-cli scroll --direction up --amount 10

# Scroll left or right
desktop-cli scroll --direction left --amount 5
desktop-cli scroll --direction right

# Scroll at specific screen coordinates
desktop-cli scroll --direction down --x 500 --y 400

# Scroll within an element found by text
desktop-cli scroll --direction down --text "Web Content" --app "Safari"

# Scroll within a specific element by ID
desktop-cli scroll --direction down --id 6 --app "Safari"
```

### Drag

```bash
# Drag from one screen coordinate to another
desktop-cli drag --from-x 100 --from-y 200 --to-x 400 --to-y 300

# Drag between elements found by text
desktop-cli drag --from-text "Document.pdf" --to-text "Trash" --app "Finder"

# Drag between elements by ID
desktop-cli drag --from-id 3 --to-id 7 --app "Finder"

# Mix: drag from element to coordinates
desktop-cli drag --from-id 5 --to-x 500 --to-y 600 --app "Finder"
```

### Wait for UI conditions

```bash
# Wait for a "Submit" button to appear
desktop-cli wait --app "Safari" --for-text "Submit" --for-role "btn"

# Wait for a loading indicator to disappear
desktop-cli wait --app "Safari" --for-text "Loading..." --gone

# Wait for any input field to appear with custom timeout
desktop-cli wait --app "Chrome" --for-role "input" --timeout 10

# Wait for element ID 5 to exist
desktop-cli wait --app "Safari" --for-id 5

# Fast polling for time-sensitive waits
desktop-cli wait --app "Safari" --for-text "Done" --interval 200
```

### Batch multiple actions (`do`)

```bash
# Form fill — 8 steps in 1 CLI call instead of 8
desktop-cli do --app "Safari" <<'EOF'
- click: { text: "Full Name" }
- type: { text: "John Doe" }
- type: { key: "tab" }
- type: { text: "john@example.com" }
- click: { text: "Submit" }
- wait: { for-text: "Thank you", timeout: 10 }
EOF

# Calculator — type full expression
desktop-cli do --app "Calculator" <<'EOF'
- action: { text: "C" }
- type: { text: "347*29+156=" }
EOF

# Multi-app workflow
desktop-cli do <<'EOF'
- focus: { app: "Safari" }
- click: { text: "Address", app: "Safari" }
- type: { text: "https://example.com", key: "enter" }
- wait: { for-text: "Example Domain", app: "Safari", timeout: 10 }
- read: { app: "Safari", format: "agent" }
EOF

# With sleep for animations
desktop-cli do --app "System Settings" <<'EOF'
- click: { text: "General" }
- sleep: { ms: 500 }
- click: { text: "About" }
- sleep: { ms: 500 }
- read: { format: "agent" }
EOF

# Continue past errors
desktop-cli do --app "Safari" --stop-on-error=false <<'EOF'
- click: { text: "Maybe Missing" }
- click: { text: "Definitely Here" }
EOF
```

Steps are provided as a YAML list on stdin. Each step is a command name with its flags as a map. Supported step types: `click`, `type`, `action`, `set-value`, `scroll`, `wait`, `focus`, `read`, `sleep`.

The `--app` and `--window` flags set defaults for all steps; per-step `app`/`window` keys override them. By default, execution stops on the first error (`--stop-on-error`). Display elements are collected once at the end.

### Find elements across windows

```bash
# Find an element by text across all windows
desktop-cli find --text "Save As"

# Find with role filter
desktop-cli find --text "Allow" --roles "btn"

# Find across all windows of a specific app
desktop-cli find --text "Save" --app "Safari"

# Limit total results (default: 10)
desktop-cli find --text "OK" --limit 5

# Exact match instead of substring
desktop-cli find --text "Submit" --exact
```

Searches all windows (or all windows of a specific app) for matching elements. Useful when a dialog, notification, or new window appeared and you don't know which app owns it. Results are grouped by window, with focused windows searched first.

### Perform accessibility actions

```bash
# Press a button by text (finds it and presses via accessibility API — most reliable)
desktop-cli action --text "Submit" --app "Safari"

# Press a button by text with role filter
desktop-cli action --text "Save" --roles "btn" --app "Safari"

# Press a button by element ID
desktop-cli action --id 5 --app "Safari"

# Press is the default action, but you can specify others:
desktop-cli action --id 5 --action press --app "Safari"

# Increment a slider/stepper
desktop-cli action --id 12 --action increment --app "System Settings"

# Decrement a value
desktop-cli action --id 12 --action decrement --app "System Settings"

# Show context menu for an element
desktop-cli action --id 8 --action showMenu --app "Finder"

# Pick/select a dropdown item
desktop-cli action --id 15 --action pick --app "Safari"

# Cancel a dialog or operation
desktop-cli action --id 3 --action cancel --app "Safari"
```

**Response format:** The `action` command returns information about the target element, plus any display elements (read-only text with values, e.g. Calculator display) that reflect the result of the action:

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

The `display` field is omitted when no display elements exist in the app. Display elements are capped at 20 to prevent excessive output in apps with many text elements. Use `--no-display` on `click`, `action`, or `type` to skip display collection entirely. This eliminates the need for a follow-up `read` call to check display values after pressing buttons (e.g. in Calculator).

### Post-read UI state (`--post-read`)

The `--post-read` flag is available on `click`, `type`, `action`, and `set-value`. It appends a compact agent-format snapshot of the full UI state to the response, eliminating the need for a follow-up `read` call:

```bash
desktop-cli click --text "Submit" --app "Safari" --post-read
```

```yaml
ok: true
action: click
x: 250
y: 416
button: left
count: 1
state: |
    # Contact Form - Safari (pid: 5678)
    [91] txt "Thank you for your submission" (100,250,300,20) display
    [92] lnk "Return to home" (100,290,150,18)
    [93] btn "Close" (100,330,80,32)
```

The `state` field contains the same output as `read --format agent`. When `--post-read` is used, the `display` field is omitted to avoid duplication (display elements are included in the state). Use `--post-read-delay <ms>` to wait before reading (e.g. for page transitions):

```bash
desktop-cli click --text "Submit" --app "Safari" --post-read --post-read-delay 500
```

### Set element values

```bash
# Set a text field's value by finding it by text
desktop-cli set-value --text "Search" --value "hello world" --app "Safari"

# Set a text field's value directly by ID (instant, no keystroke simulation)
desktop-cli set-value --id 4 --value "hello world" --app "Safari"

# Set a slider to a specific position
desktop-cli set-value --id 12 --value "75" --app "System Settings"

# Clear a text field
desktop-cli set-value --id 4 --value "" --app "Safari"

# Set focus on an element
desktop-cli set-value --id 4 --attribute focused --value "true" --app "Safari"

# Set selection state
desktop-cli set-value --id 8 --attribute selected --value "true" --app "Finder"
```

### Clipboard

```bash
# Read the current clipboard text content
desktop-cli clipboard read

# Write text to the clipboard
desktop-cli clipboard write "Hello, world"
desktop-cli clipboard write --text "Hello, world"

# Clear the clipboard
desktop-cli clipboard clear

# Select all + copy from an app, then read clipboard
# (focuses the app, sends Cmd+A, Cmd+C, then reads clipboard)
desktop-cli clipboard grab --app "Safari"
desktop-cli clipboard grab --app "Google Chrome" --window "Gmail"
```

The `clipboard grab` command is a convenience for extracting text from apps where the accessibility tree may not expose content (e.g. Gmail compose body, contenteditable fields).

### Focus a window

```bash
# Focus by app name
desktop-cli focus --app "Safari"

# Focus by PID
desktop-cli focus --pid 1234

# Focus a specific window by title substring
desktop-cli focus --app "Safari" --window "GitHub"

# Focus by system window ID
desktop-cli focus --window-id 5678
```

### Observe UI changes

```bash
# Watch Safari for any UI changes, emit diffs as JSONL
desktop-cli observe --app "Safari"

# Watch for changes to buttons and links only
desktop-cli observe --app "Safari" --roles "btn,lnk"

# Watch a specific window with custom polling interval (ms)
desktop-cli observe --app "Chrome" --window "Gmail" --interval 500

# Watch for a limited duration (seconds)
desktop-cli observe --app "Safari" --duration 10

# Watch at limited depth
desktop-cli observe --app "Safari" --depth 3

# Ignore noisy layout and focus changes
desktop-cli observe --app "Safari" --ignore-bounds --ignore-focus

# Pipe to jq for real-time filtering
desktop-cli observe --app "Safari" | jq 'select(.type=="added")'
```

Output is always JSONL (one JSON object per line). Events: `snapshot` (initial count), `added`, `removed`, `changed`, `error`, `done`.

### Screenshot

```bash
# Capture the full screen (outputs base64 PNG to stdout)
desktop-cli screenshot

# Capture a specific app's window
desktop-cli screenshot --app "Safari"

# Capture by window title substring
desktop-cli screenshot --window "GitHub"

# Save to a file instead of stdout
desktop-cli screenshot --app "Safari" --output /tmp/safari.png

# Capture at full resolution (default is 0.5 for token efficiency)
desktop-cli screenshot --app "Safari" --scale 1.0

# Capture at quarter resolution
desktop-cli screenshot --app "Safari" --scale 0.25

# Include menu bar in app screenshot (composite of menu bar + window)
desktop-cli screenshot --app "Safari" --include-menubar

# Capture as JPEG with custom quality
desktop-cli screenshot --app "Safari" --format jpg --quality 60

# Capture by window ID or PID
desktop-cli screenshot --window-id 5678
desktop-cli screenshot --pid 1234
```

### Screenshot with Coordinates

```bash
# Capture screenshot with coordinate labels on interactive elements (default)
desktop-cli screenshot-coords --app "Safari" --output /tmp/safari-coords.png

# Capture with all elements labeled (not just interactive)
desktop-cli screenshot-coords --app "Safari" --all-elements --output /tmp/safari-all.png

# Filter to specific roles
desktop-cli screenshot-coords --app "Safari" --roles "btn,lnk" --output /tmp/safari-buttons.png

# Filter by text content
desktop-cli screenshot-coords --app "Safari" --text "Search" --output /tmp/safari-search.png

# Combine options: all elements, specific depth, pruned, full resolution
desktop-cli screenshot-coords --app "Safari" --all-elements --depth 4 --prune --scale 1.0 --output /tmp/full.png

# Output as base64 to stdout (like regular screenshot)
desktop-cli screenshot-coords --app "Safari"

# JPEG output
desktop-cli screenshot-coords --app "Safari" --format jpg --quality 85
```

Each element is drawn with:
- A red bounding box showing its screen location
- A coordinate label at the center: `(x,y)` showing the center point

Default behavior shows only interactive elements (buttons, links, inputs, etc.). Use `--all-elements` to label everything in the accessibility tree.

See `desktop-cli --help` and `desktop-cli <command> --help` for full usage details.

## Development

### Build

```bash
# Quick dev build (includes git commit and build date for traceability)
./update.sh

# Or manually:
go build -ldflags "-X github.com/mj1618/desktop-cli/internal/version.Commit=$(git rev-parse --short HEAD) -X github.com/mj1618/desktop-cli/internal/version.BuildDate=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" -o desktop-cli .
codesign --force --sign - ./desktop-cli   # required on macOS Apple Silicon
```

### Test

```bash
go test -v ./...
```

## License

MIT
