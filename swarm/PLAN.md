# Desktop CLI — Implementation Plan

A GoLang CLI that lets AI agents read and interact with desktop UI elements. Agents get a structured JSON snapshot of any window's UI tree, then issue commands to click elements and type text — all from the terminal.

## Problem Statement

Agents need to automate desktop tasks but lack a fast, structured way to:
1. **Read** what's on screen — specifically, what elements exist in a given window/tab, their labels, roles, positions, and states
2. **Act** on those elements — click buttons, type into fields, press key combos
3. **Do both efficiently** — minimize latency and token count so agents can operate in tight loops

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                    desktop-cli                       │
│                                                      │
│  ┌──────────┐   ┌──────────┐   ┌──────────────────┐ │
│  │  Cobra   │   │  Screen  │   │     Input        │ │
│  │  CLI     │──▶│  Reader  │   │   Simulator      │ │
│  │  Layer   │   │          │   │                   │ │
│  └──────────┘   └────┬─────┘   └────────┬─────────┘ │
│                      │                   │           │
│            ┌─────────▼───────────────────▼─────────┐ │
│            │        Platform Abstraction            │ │
│            │  (interface: Reader, Inputter)         │ │
│            └─────────┬───────────────────┬─────────┘ │
│                      │                   │           │
│  ┌───────────────────▼──┐  ┌─────────────▼────────┐ │
│  │   macOS Backend      │  │   Linux Backend      │ │
│  │   (CGo + AX API)     │  │   (AT-SPI / X11)    │ │
│  └──────────────────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

## Core Design Principles

1. **Token efficiency** — JSON output uses short keys, omits empty fields, and supports filtering so agents only see what they need
2. **Element IDs** — Every element gets a short integer ID stable within a single read; agents reference these IDs for click/type actions
3. **Speed** — Native accessibility APIs via CGo; no screenshots or OCR in the critical path
4. **Platform abstraction** — Interfaces for reading and input so we can add Linux/Windows backends later
5. **Stateless** — Each CLI invocation is independent; no daemon, no socket, no persistent state

## CLI Commands

All commands are subcommands of `desktop-cli`. Output defaults to stdout as JSON (for agent consumption) or human-readable (for debugging).

### `desktop-cli read`

Read the UI element tree and output as JSON.

```
desktop-cli read [flags]

Flags:
  --app <name>          Filter to a specific application by name (e.g. "Safari")
  --window <title>      Filter to a specific window by title substring
  --window-id <id>      Filter to a specific window by system window ID
  --pid <pid>           Filter to a specific process by PID
  --depth <n>           Max depth to traverse (default: unlimited)
  --roles <roles>       Comma-separated roles to include (e.g. "button,textfield,link")
  --visible-only        Only include visible/on-screen elements (default: true)
  --bbox <x,y,w,h>      Only include elements within bounding box
  --compact             Ultra-compact output: flatten tree, minimal keys (default: false)
  --pretty              Pretty-print JSON (default: false, single line)
```

### `desktop-cli list`

List available windows and applications.

```
desktop-cli list [flags]

Flags:
  --apps                List running applications
  --windows             List all windows (default)
  --pid <pid>           Filter windows by PID
  --app <name>          Filter windows by app name
```

### `desktop-cli click`

Click on an element or at coordinates.

```
desktop-cli click [flags]

Flags:
  --id <id>             Click element by ID (requires preceding `read` to establish context)
  --x <x> --y <y>      Click at absolute screen coordinates
  --button <btn>        Mouse button: left (default), right, middle
  --double              Double-click
  --app <name>          Scope to application (used with --id)
  --window <title>      Scope to window (used with --id)
```

**How `--id` works**: When the agent calls `read`, each element gets an integer ID. The agent then calls `click --id 42 --app Safari`. The CLI re-reads the element tree for that app, finds element 42 by matching the same traversal order, computes its center coordinates, and clicks. This is stateless — no daemon needed.

### `desktop-cli type`

Type text or press key combinations.

```
desktop-cli type [flags] [text]

Flags:
  --text <text>         Text to type (alternative to positional arg)
  --key <combo>         Key combination (e.g. "cmd+c", "ctrl+shift+t", "enter", "tab")
  --delay <ms>          Delay between keystrokes in ms (default: 0)
  --id <id>             Focus element by ID first, then type
  --app <name>          Scope to application (used with --id)
  --window <title>      Scope to window (used with --id)
```

### `desktop-cli focus`

Bring a window or application to the foreground.

```
desktop-cli focus [flags]

Flags:
  --app <name>          Focus application by name
  --window <title>      Focus window by title substring
  --window-id <id>      Focus window by system ID
  --pid <pid>           Focus application by PID
```

### `desktop-cli screenshot`

Capture a screenshot (useful for vision model fallback).

```
desktop-cli screenshot [flags]

Flags:
  --window <title>      Capture specific window
  --app <name>          Capture specific app's frontmost window
  --output <path>       Output file path (default: stdout as base64 or /tmp/screenshot.png)
  --format <fmt>        png (default), jpg
  --quality <n>         JPEG quality 1-100 (default: 80)
  --scale <n>           Scale factor 0.1-1.0 (default: 0.5, for token efficiency)
```

### `desktop-cli scroll`

Scroll within a window or element.

```
desktop-cli scroll [flags]

Flags:
  --direction <dir>     up, down, left, right
  --amount <n>          Number of "clicks" to scroll (default: 3)
  --x <x> --y <y>      Scroll at specific coordinates
  --id <id>             Scroll within element by ID
  --app <name>          Scope to application
  --window <title>      Scope to window
```

### `desktop-cli drag`

Drag from one point to another.

```
desktop-cli drag [flags]

Flags:
  --from-x <x>         Start X coordinate
  --from-y <y>         Start Y coordinate
  --to-x <x>           End X coordinate
  --to-y <y>           End Y coordinate
  --from-id <id>       Start element (center)
  --to-id <id>         End element (center)
  --app <name>          Scope to application
  --window <title>      Scope to window
```

## JSON Output Schema

### Token-Efficient Design

The JSON schema uses short single-letter keys and omits empty/default fields to minimize tokens.

#### Element Object

| Key | Full Name    | Type     | Description                              | Omit when   |
|-----|-------------|----------|------------------------------------------|-------------|
| `i` | id          | int      | Element ID (stable within one read call) | never       |
| `r` | role        | string   | Element role: `btn`, `txt`, `lnk`, `img`, `input`, `chk`, `radio`, `menu`, `menuitem`, `tab`, `list`, `row`, `cell`, `group`, `scroll`, `toolbar`, `static`, `web`, `other` | never |
| `t` | title       | string   | Visible label / title text               | empty       |
| `v` | value       | string   | Current value (text field content, checkbox state, etc.) | empty |
| `d` | description | string   | Accessibility description / alt-text     | empty       |
| `b` | bounds      | [4]int   | `[x, y, width, height]` in screen coords | never      |
| `f` | focused     | bool     | Whether this element has keyboard focus  | false       |
| `e` | enabled     | bool     | Whether this element is interactive      | true (omit when enabled) |
| `s` | selected    | bool     | Whether this element is selected         | false       |
| `c` | children    | []Element| Child elements                           | empty       |
| `a` | actions     | []string | Available actions: `press`, `cancel`, `pick`, `increment`, `decrement` | empty |

#### Abbreviated Roles

Full accessibility roles are mapped to short tokens:

| Short | macOS AXRole           |
|-------|------------------------|
| `btn` | AXButton               |
| `txt` | AXStaticText           |
| `lnk` | AXLink                 |
| `img` | AXImage                |
| `input`| AXTextField / AXTextArea |
| `chk` | AXCheckBox             |
| `radio`| AXRadioButton          |
| `menu`| AXMenu / AXMenuBar     |
| `menuitem` | AXMenuItem        |
| `tab` | AXTabGroup             |
| `list`| AXList / AXTable       |
| `row` | AXRow                  |
| `cell`| AXCell                 |
| `group`| AXGroup / AXSplitGroup|
| `scroll`| AXScrollArea          |
| `toolbar`| AXToolbar            |
| `static`| AXStaticText (non-interactive) |
| `web` | AXWebArea              |
| `window`| AXWindow              |
| `other`| everything else        |

#### Example Output: `desktop-cli read --app "Safari" --depth 3 --compact`

```json
{
  "app": "Safari",
  "pid": 1234,
  "window": "GitHub - desktop-cli",
  "ts": 1707500000,
  "elements": [
    {"i":1,"r":"toolbar","t":"Navigation","b":[0,0,1440,52],"c":[
      {"i":2,"r":"btn","t":"Back","b":[10,10,32,32],"a":["press"]},
      {"i":3,"r":"btn","t":"Forward","b":[46,10,32,32],"e":false,"a":["press"]},
      {"i":4,"r":"input","t":"Address","v":"https://github.com/mj1618/desktop-cli","b":[200,10,800,32],"f":true},
      {"i":5,"r":"btn","t":"Reload","b":[1010,10,32,32],"a":["press"]}
    ]},
    {"i":6,"r":"web","t":"Web Content","b":[0,52,1440,848],"c":[
      {"i":7,"r":"lnk","t":"Code","b":[120,80,40,20],"a":["press"]},
      {"i":8,"r":"lnk","t":"Issues","b":[170,80,45,20],"a":["press"]},
      {"i":9,"r":"btn","t":"Star","b":[1200,80,60,28],"a":["press"]},
      {"i":10,"r":"txt","t":"A command-line tool for desktop automation.","b":[120,140,600,20]}
    ]}
  ]
}
```

#### Example Output: `desktop-cli list --windows`

```json
[
  {"app":"Safari","pid":1234,"title":"GitHub - desktop-cli","id":42,"bounds":[0,0,1440,900],"focused":true},
  {"app":"Terminal","pid":5678,"title":"~/code/desktop-cli","id":43,"bounds":[100,100,800,600],"focused":false},
  {"app":"Finder","pid":9012,"title":"Downloads","id":44,"bounds":[200,200,700,500],"focused":false}
]
```

## Module Structure

```
desktop-cli/
├── main.go                          # Entry point
├── cmd/
│   ├── root.go                      # Root cobra command, global flags
│   ├── read.go                      # `read` subcommand
│   ├── list.go                      # `list` subcommand
│   ├── click.go                     # `click` subcommand
│   ├── type.go                      # `type` subcommand (named typecmd.go to avoid keyword)
│   ├── focus.go                     # `focus` subcommand
│   ├── scroll.go                    # `scroll` subcommand
│   ├── drag.go                      # `drag` subcommand
│   └── screenshot.go               # `screenshot` subcommand
├── internal/
│   ├── platform/
│   │   ├── platform.go              # Interfaces: Reader, Inputter, WindowManager
│   │   ├── types.go                 # Shared types: Element, Window, Bounds, etc.
│   │   ├── darwin/
│   │   │   ├── reader.go            # macOS AX API screen reader
│   │   │   ├── reader_cgo.go        # CGo bridge for AXUIElement
│   │   │   ├── inputter.go          # macOS CGEvent mouse/keyboard simulation
│   │   │   ├── inputter_cgo.go      # CGo bridge for CGEvent
│   │   │   ├── window.go            # macOS window management
│   │   │   ├── accessibility.h      # C header for AX API helpers
│   │   │   └── accessibility.c      # C implementation for AX API helpers
│   │   └── linux/                   # Future: AT-SPI + xdotool/ydotool
│   │       ├── reader.go
│   │       ├── inputter.go
│   │       └── window.go
│   ├── model/
│   │   ├── element.go               # Element struct with JSON tags
│   │   ├── window.go                # Window struct
│   │   └── filter.go                # Filtering/pruning logic
│   ├── output/
│   │   └── json.go                  # JSON serialization with compact mode
│   └── version/
│       └── version.go               # (existing) build version info
├── go.mod
├── go.sum
├── README.md
├── SKILL.md
├── AGENTS.md
└── CLAUDE.md
```

## Platform Interfaces

```go
// internal/platform/platform.go

// Reader reads the UI element tree from the OS accessibility layer.
type Reader interface {
    // ReadElements returns the element tree for the specified target.
    // Options control filtering (app, window, depth, roles, bbox, visible-only).
    ReadElements(opts ReadOptions) ([]model.Element, error)

    // ListWindows returns all windows, optionally filtered.
    ListWindows(opts ListOptions) ([]model.Window, error)
}

// Inputter simulates mouse and keyboard input.
type Inputter interface {
    Click(x, y int, button MouseButton, count int) error
    MoveMouse(x, y int) error
    Scroll(x, y int, dx, dy int) error
    Drag(fromX, fromY, toX, toY int) error
    TypeText(text string, delayMs int) error
    KeyCombo(keys []string) error
}

// WindowManager manages window focus and positioning.
type WindowManager interface {
    FocusWindow(opts FocusOptions) error
    GetFrontmostApp() (string, int, error)
}
```

## macOS Implementation Details

### Accessibility API (Screen Reading)

The core of the tool. Uses CGo to call macOS Accessibility Framework:

1. **Get running apps** — `NSWorkspace.runningApplications` or `CGWindowListCopyWindowInfo`
2. **Create AX app reference** — `AXUIElementCreateApplication(pid)`
3. **Get windows** — `AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, &windows)`
4. **Traverse element tree recursively**:
   - For each element, read: `kAXRoleAttribute`, `kAXTitleAttribute`, `kAXValueAttribute`, `kAXDescriptionAttribute`, `kAXPositionAttribute`, `kAXSizeAttribute`, `kAXEnabledAttribute`, `kAXFocusedAttribute`, `kAXSelectedAttribute`, `kAXChildrenAttribute`
   - Map AX roles to short role codes
   - Assign sequential integer IDs during traversal
   - Apply filters (depth, roles, bbox, visibility) during traversal to avoid unnecessary work
5. **Return structured element tree**

### CGo Bridge Pattern

```c
// internal/platform/darwin/accessibility.h
#include <ApplicationServices/ApplicationServices.h>

typedef struct {
    int id;
    char* role;
    char* title;
    char* value;
    char* description;
    float x, y, width, height;
    int enabled, focused, selected;
    int childCount;
    int* childIDs;
    int actionCount;
    char** actions;
} AXElementInfo;

// Get all elements for an app window, returns flat array
int ax_read_elements(pid_t pid, int windowIndex, int maxDepth,
                     AXElementInfo** outElements, int* outCount);

// Free element array
void ax_free_elements(AXElementInfo* elements, int count);
```

The C layer does the heavy lifting of AX traversal and returns a flat array. Go code reassembles the tree, applies filters, and serializes to JSON. This minimizes CGo crossing overhead.

### Input Simulation

Uses `CGEventCreateMouseEvent` and `CGEventCreateKeyboardEvent` via CGo:

- **Mouse click**: Create mouse down + mouse up events at coordinates, post to system
- **Key typing**: Convert characters to key codes, create and post keyboard events
- **Key combos**: Set modifier flags (cmd, shift, ctrl, opt) on keyboard events
- **Scroll**: `CGEventCreateScrollWheelEvent`

### Permissions

macOS requires explicit accessibility permissions. The CLI must:
1. Check permission on startup: `AXIsProcessTrusted()`
2. If denied, print a clear message telling the user how to grant permission (System Settings > Privacy & Security > Accessibility)
3. For screen recording (screenshots), check `CGPreflightScreenCaptureAccess()` and request with `CGRequestScreenCaptureAccess()`

## Performance Strategy

### Speed Targets

- `list --windows`: < 50ms
- `read --app "Safari" --depth 3`: < 200ms for typical pages
- `click --id 42`: < 100ms (re-read + click)
- `type --text "hello"`: < 50ms + keystroke delay

### Optimizations

1. **Filter during traversal** — Don't read children of elements below max depth or outside bbox
2. **Parallel app queries** — When no app filter, query multiple apps concurrently
3. **Minimal CGo crossings** — Do bulk work in C, return flat arrays, reassemble in Go
4. **Visible-only default** — Skip off-screen elements (check bounds vs screen)
5. **Role filtering early** — Skip elements and their subtrees when using `--roles` filter
6. **Lazy attribute reads** — Only read `value`, `description`, `actions` for elements that pass initial filters

### Token Efficiency

The compact JSON format is designed to minimize tokens:

| Approach | Tokens for 50-element tree (est.) |
|----------|-----------------------------------|
| Verbose (full key names, all fields) | ~2000 |
| Compact (short keys, omit defaults)  | ~800  |
| Compact + role filter                | ~300  |

Key savings:
- Single-letter keys save ~40% vs full names
- Omitting empty/default fields saves ~30%
- `--roles` filtering saves 50-80% by excluding irrelevant elements
- `--depth` limiting prevents runaway token counts on complex UIs
- Bounds as `[x,y,w,h]` array instead of `{"x":..,"y":..,"width":..,"height":..}` saves 60% on bounds

## Implementation Phases

### Phase 1: Foundation (MVP)

**Goal**: Read UI elements from a macOS window and output JSON.

Tasks:
1. Set up Cobra CLI with root command and `--version` flag
2. Implement `list --windows` command (list all windows with app name, title, PID)
3. Implement `read --app <name>` with basic element tree traversal via CGo + AX API
4. Define element JSON schema with compact short keys
5. Implement `--depth`, `--visible-only` flags
6. Implement `--pretty` and `--compact` output modes
7. Add accessibility permission check with helpful error message
8. Write unit tests for JSON serialization and filtering logic
9. Write integration test that reads a known window

### Phase 2: Input Actions

**Goal**: Click, type, and interact with elements.

Tasks:
1. Implement `click --x <x> --y <y>` (absolute coordinate click)
2. Implement `click --id <id> --app <name>` (element ID click via re-read)
3. Implement `type --text <text>` (keystroke simulation)
4. Implement `type --key <combo>` (key combination)
5. Implement `focus --app <name>` / `focus --window <title>`
6. Implement `scroll` command
7. Add `--double` and `--button` flags to click
8. Write integration tests for input actions

### Phase 3: Filtering & Optimization

**Goal**: Make it fast and token-efficient for real agent workflows.

Tasks:
1. Implement `--roles` filter (only include specific element types)
2. Implement `--bbox` filter (spatial filtering)
3. Implement `--window` and `--window-id` filters for read
4. Optimize CGo bridge: batch element reads in C, minimize crossings
5. Add `--pid` filter support throughout
6. Performance benchmarking and optimization
7. Implement element caching for `click --id` (hash-based matching to avoid drift)

### Phase 4: Advanced Features

**Goal**: Full-featured tool ready for complex agent workflows.

Tasks:
1. Implement `screenshot` command with window targeting and scaling
2. Implement `drag` command
3. Add `--delay` flag to `type` for realistic typing speed
4. Implement `--id` targeting for `type` and `scroll` (focus element first)
5. Add `--format csv` output option for ultra-compact tabular output
6. Add `--watch` mode to `read` (poll and output changes as JSONL)
7. Add element action execution via AX API (`AXUIElementPerformAction`)

### Phase 5: Cross-Platform & Hardening

**Goal**: Linux support and production hardening.

Tasks:
1. Implement Linux backend using AT-SPI for screen reading
2. Implement Linux input via xdotool/ydotool or uinput
3. Implement Linux window management via wmctrl / xdotool
4. Add retry logic for transient accessibility failures
5. Add timeout flags for all commands
6. Comprehensive error messages for all failure modes
7. CI testing on macOS and Linux

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/pflag` | Flag parsing (included with Cobra) |
| CGo + ApplicationServices.framework | macOS Accessibility API |
| CGo + CoreGraphics.framework | macOS input simulation + screenshots |
| (future) AT-SPI D-Bus bindings | Linux accessibility |
| (future) xdotool / ydotool | Linux input simulation |

No heavy external Go dependencies. The tool should be a single static binary with CGo for platform APIs only.

## Security & Permissions

1. **macOS Accessibility Permission** — Required. The CLI must be added to System Settings > Privacy & Security > Accessibility. On first run, detect missing permission and print instructions.
2. **macOS Screen Recording Permission** — Required only for `screenshot` command.
3. **No network access** — The CLI is entirely local. No telemetry, no phone-home.
4. **No persistent state** — No files written, no config needed, no daemon.

## Agent Workflow Example

A typical agent loop using the CLI:

```bash
# 1. List windows to find the target
$ desktop-cli list --windows
[{"app":"Chrome","pid":1234,"title":"Gmail - Inbox","id":42,...},...]

# 2. Read the UI tree for that window
$ desktop-cli read --app "Chrome" --window "Gmail" --depth 4 --roles "btn,lnk,input,txt"
{"app":"Chrome","window":"Gmail - Inbox","elements":[
  {"i":1,"r":"input","t":"Search mail","b":[200,60,500,36],"f":true},
  {"i":2,"r":"btn","t":"Compose","b":[20,60,100,36],"a":["press"]},
  {"i":3,"r":"lnk","t":"Inbox (3)","b":[20,120,80,20]},
  ...
]}

# 3. Click the Compose button
$ desktop-cli click --id 2 --app "Chrome" --window "Gmail"

# 4. Wait a moment, then read again to see the compose window
$ sleep 1 && desktop-cli read --app "Chrome" --depth 3 --roles "input,btn,txt"

# 5. Type into the To field
$ desktop-cli type --id 5 --app "Chrome" --text "colleague@example.com"

# 6. Press Tab to move to subject
$ desktop-cli type --key "tab"

# 7. Type the subject
$ desktop-cli type --text "Meeting tomorrow"
```

## Open Questions

1. **Element ID stability** — IDs are assigned by traversal order, which means they can shift if the UI changes between `read` and `click`. Mitigation: re-read on click and match by role+title+position hash. Should we implement this from the start or add it in Phase 3?

2. **Web content depth** — Browser web content can have extremely deep accessibility trees. Should we default to a depth limit (e.g., 5) or let the agent choose?

3. **Rate limiting** — Should the CLI have built-in rate limiting to prevent accidentally flooding the system with events? Or leave that to the calling agent?

4. **Daemon mode** — A persistent daemon could maintain element state and make `click --id` faster and more reliable. But it adds complexity. Worth it in a later phase?

5. **Tab support** — Browser tabs appear as children in the accessibility tree. Should we add explicit `--tab` filtering, or is `--window` sufficient since each tab has a distinct title?
