# Desktop CLI

A command-line tool that lets AI agents read and interact with desktop UI elements. Agents get a structured YAML snapshot of any window's accessibility tree, then issue commands to click, type, scroll, and more — all from the terminal.

## Features

- **Read UI elements** — Get a compact YAML tree of buttons, links, text fields, and other elements in any window
- **Click elements** — Click by element ID or screen coordinates
- **Type text** — Simulate keyboard input and key combinations
- **Focus windows** — Bring windows to the foreground
- **Scroll & drag** — Scroll within windows or drag between points
- **Screenshot** — Capture windows for vision model fallback
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
go build -o desktop-cli .
sudo mv desktop-cli /usr/local/bin/
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

# Target a specific window by title substring
desktop-cli read --app "Safari" --window "GitHub"

# Target by PID or window ID
desktop-cli read --pid 1234
desktop-cli read --window-id 5678

# Filter to a bounding box region (x,y,width,height)
desktop-cli read --app "Finder" --bbox "0,0,800,600"
```

### Click an element

```bash
# Click by element ID (re-reads the element tree to get current coordinates)
desktop-cli click --id 5 --app "Safari"

# Click at absolute screen coordinates
desktop-cli click --x 100 --y 200

# Right-click
desktop-cli click --x 100 --y 200 --button right

# Double-click
desktop-cli click --x 100 --y 200 --double
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

# Click an element to focus it, then type into it
desktop-cli type --id 4 --app "Safari" --text "search query"
```

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

# Scroll within a specific element by ID
desktop-cli scroll --direction down --id 6 --app "Safari"
```

### Drag

```bash
# Drag from one screen coordinate to another
desktop-cli drag --from-x 100 --from-y 200 --to-x 400 --to-y 300

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

### Perform accessibility actions

```bash
# Press a button directly via accessibility API (more reliable than click)
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

# Capture as JPEG with custom quality
desktop-cli screenshot --app "Safari" --format jpg --quality 60

# Capture by window ID or PID
desktop-cli screenshot --window-id 5678
desktop-cli screenshot --pid 1234
```

See `desktop-cli --help` and `desktop-cli <command> --help` for full usage details.

## Development

### Build

```bash
go build -v ./...
```

### Test

```bash
go test -v ./...
```

## License

MIT
