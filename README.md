# Desktop CLI

A command-line tool that lets AI agents read and interact with desktop UI elements. Agents get a structured JSON snapshot of any window's accessibility tree, then issue commands to click, type, scroll, and more — all from the terminal.

## Features

- **Read UI elements** — Get a compact JSON tree of buttons, links, text fields, and other elements in any window
- **Click elements** — Click by element ID or screen coordinates
- **Type text** — Simulate keyboard input and key combinations
- **Focus windows** — Bring windows to the foreground
- **Scroll & drag** — Scroll within windows or drag between points
- **Screenshot** — Capture windows for vision model fallback
- **Token efficient** — Short JSON keys, role abbreviations, and filtering to minimize agent token usage
- **Fast** — Native accessibility APIs via CGo, no OCR or screenshots in the critical path

## Requirements

- **macOS**: Grant Accessibility permission in System Settings > Privacy & Security > Accessibility
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
desktop-cli list --windows
```

### Read UI elements

```bash
desktop-cli read --app "Safari" --depth 4 --roles "btn,lnk,input,txt"
```

### Click an element

```bash
desktop-cli click --id 5 --app "Safari"
desktop-cli click --x 100 --y 200
```

### Type text or key combos

```bash
desktop-cli type --text "hello world"
desktop-cli type --key "cmd+c"
```

### Focus a window

```bash
desktop-cli focus --app "Safari"
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
