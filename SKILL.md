# desktop-cli

A CLI tool for AI agents to read and interact with desktop UI elements.

## Installation

```bash
go install github.com/mj1618/desktop-cli@latest
```

Or download a binary from the [releases page](https://github.com/mj1618/desktop-cli/releases/latest).

**macOS**: Grant Accessibility permission in System Settings > Privacy & Security > Accessibility.

## Quick Reference

### List windows

```bash
desktop-cli list --windows
```

### Read UI elements from a window

```bash
desktop-cli read --app "Safari" --depth 4 --roles "btn,lnk,input,txt"
```

Returns compact JSON with short keys: `i` (id), `r` (role), `t` (title), `v` (value), `d` (description), `b` (bounds), `c` (children), `a` (actions).

### Click an element

```bash
desktop-cli click --id 5 --app "Safari"
desktop-cli click --x 100 --y 200
```

### Type text

```bash
desktop-cli type --text "hello world"
desktop-cli type --key "cmd+c"
```

### Focus a window

```bash
desktop-cli focus --app "Safari" --window "GitHub"
```

### Scroll

```bash
desktop-cli scroll --direction down --amount 5 --app "Safari"
```

## Agent Workflow

1. `list --windows` to find the target window
2. `read --app <name> --depth 3 --roles "btn,lnk,input,txt"` to get the element tree as JSON
3. Use the element `i` (id) field to `click --id <id>` or `type --id <id> --text "..."`
4. Repeat read/act loop as needed

## JSON Output Keys

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
