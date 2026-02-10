# Hover Command

## Priority: MEDIUM

## Problem

Many desktop and web UIs reveal additional controls, tooltips, or menus only when the user hovers over an element. Common examples:

- **Table/list row actions**: Gmail, Jira, GitHub, and many web apps show action buttons (archive, delete, edit) only when hovering over a row
- **Tooltips**: Hovering over icons or truncated text reveals full labels or help text
- **Dropdown/flyout menus**: Navigation menus that expand on hover (not click)
- **Preview panels**: Hovering over links or thumbnails can trigger preview popups

Currently, agents have no way to trigger hover-dependent UI. The `click` command moves the mouse and clicks, but there's no way to move the mouse without clicking. Agents working with hover-dependent UIs must either guess at hidden controls or fall back to keyboard navigation, which doesn't always work.

## Solution

Add a `hover` command that moves the mouse cursor to an element (or coordinates) without clicking, then optionally waits for the UI to update and returns the new state.

### CLI Interface

```bash
# Hover over an element found by text
desktop-cli hover --text "Important Email" --app "Chrome"

# Hover by element ID
desktop-cli hover --id 42 --app "Chrome"

# Hover at absolute screen coordinates
desktop-cli hover --x 500 --y 300

# Hover and wait for new elements to appear (e.g. row actions)
desktop-cli hover --text "Important Email" --app "Chrome" --post-read --post-read-delay 300

# Hover with role filter
desktop-cli hover --text "Settings" --roles "menuitem" --app "Safari"

# Hover to trigger tooltip, then read the result
desktop-cli hover --id 15 --app "Xcode" --post-read --post-read-delay 500
```

### Response Format

```yaml
ok: true
action: hover
x: 450
y: 312
target:
    i: 42
    r: row
    t: "Important Email"
    b: [100, 300, 800, 24]
```

With `--post-read`:

```yaml
ok: true
action: hover
x: 450
y: 312
target:
    i: 42
    r: row
    t: "Important Email"
    b: [100, 300, 800, 24]
state: |
    # Gmail - Google Chrome (pid: 44037)
    [42] row "Important Email" (100,300,800,24)
    [201] btn "Archive" (750,302,20,20)
    [202] btn "Delete" (775,302,20,20)
    [203] btn "Mark as read" (800,302,20,20)
```

### Implementation

1. **`cmd/hover.go`** — New command file following the same pattern as `click.go`:
   - Reuse `addTextTargetingFlags`, `addPostReadFlags`, `resolveTargetElement` helpers
   - Support `--id`, `--text`, `--x/--y`, `--app`, `--window`, `--roles`, `--scope-id`, `--exact`
   - Support `--post-read` and `--post-read-delay` for capturing state after hover
   - Move the mouse to the element center (or coordinates) without clicking

2. **Platform interface** — Add `MouseMove(x, y int) error` to the platform provider:
   - On macOS: Use `CGEventCreateMouseEvent` with `kCGEventMouseMoved` type
   - This is simpler than click — just a single mouse move event, no button press/release

3. **`do` command integration** — Add `hover` as a valid step type in the batch `do` command

4. **Agent format in responses** — Follow the same display/state pattern as `click`

### Key Design Decisions

- **No click**: The whole point is mouse movement without clicking. Agents already have `click` for that.
- **`--post-read` support**: Critical for hover use cases since the point is usually to reveal new UI elements. The `--post-read-delay` flag lets the agent wait for animations/transitions before reading the new state.
- **Reuse targeting infrastructure**: Same `--text`, `--id`, `--x/--y` flags and element resolution as `click`, `action`, etc.
- **No hold/duration**: Simple move-to-point. If a UI needs sustained hover, the agent can just not move the mouse away. No need for a duration flag.

### Dependencies

None. Uses existing infrastructure (element targeting, post-read, platform mouse APIs).

### Testing

- Unit test: verify flag parsing and command wiring
- Manual test: hover over a Gmail row in Chrome, verify action buttons appear in `--post-read` output
- Manual test: hover over a toolbar icon in Xcode, verify tooltip appears

### Documentation

Update README.md and SKILL.md with hover command examples and agent workflow guidance (e.g., "hover over table rows to reveal action buttons").
