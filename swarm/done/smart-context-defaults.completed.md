# Feature: Smart Context Detection — Optimal Flags Applied Automatically

## Priority: MEDIUM (reduces flag verbosity and eliminates common agent mistakes)

## Problem

Agents must learn and specify the right combination of flags for each situation:

```bash
# Web page: must add --prune or output is 5-8x too large
desktop-cli read --app "Chrome" --format agent --prune

# Complex app: must add --depth or traversal takes too long
desktop-cli read --app "Chrome" --depth 4 --roles "btn,lnk,input,txt"

# Simple app: can use defaults
desktop-cli read --app "Calculator" --format agent
```

Agents frequently forget `--prune` on web apps (wasting tokens on empty divs), use wrong depth limits, or omit `--format agent` (getting verbose YAML). Each suboptimal flag choice either wastes tokens (too much output) or misses elements (too aggressive filtering).

**Common mistakes observed in swarm tests:**
1. Forgetting `--prune` on Chrome/Safari → 5-8x more output than necessary
2. Using `--depth 3` on shallow UIs (Calculator) → misses nothing, but on deep UIs (web pages) → misses critical elements
3. Omitting `--format agent` → getting nested YAML that wastes 20-30x more tokens
4. Not using `--roles "interactive"` for Chrome → missing web inputs exposed as role "other"

## What to Build

### 1. App-Aware Default Profiles

Detect the target application and apply smart defaults:

```go
type AppProfile struct {
    Prune       bool
    DefaultDepth int      // 0 = unlimited
    DefaultRoles []string // empty = all roles
    FormatHint   string   // suggested format
    Notes        string   // hints for the agent
}

var appProfiles = map[string]AppProfile{
    "Google Chrome": {
        Prune: true,               // web pages have many empty divs
        DefaultRoles: []string{},  // empty = all (but prune handles noise)
    },
    "Safari": {
        Prune: true,
    },
    "Firefox": {
        Prune: true,
    },
    "Arc": {
        Prune: true,
    },
    // Desktop apps: no special treatment needed
}
```

When `--prune` is not explicitly set and the app is a known web browser, auto-enable pruning.

### 2. Auto-Detect Web Content

Instead of hardcoding browser names, detect web content by checking for the "web" role in the element tree:

```go
func hasWebContent(elements []model.Element) bool {
    for _, el := range elements {
        if el.Role == "web" {
            return true
        }
        if hasWebContent(el.Children) {
            return true
        }
    }
    return false
}
```

If web content is detected, auto-enable `--prune` (unless `--no-prune` is explicitly passed).

### 3. Piped Output Detection

When stdout is a pipe (agent context), suggest/apply agent-friendly defaults:

```go
func isOutputPiped() bool {
    stat, _ := os.Stdout.Stat()
    return (stat.Mode() & os.ModeCharDevice) == 0
}
```

When piped:
- Default format: `agent` (not `yaml`) — 20-30x fewer tokens
- Default: include only interactive + display elements

When interactive (terminal):
- Default format: `yaml` (human-readable)
- Default: all elements

### 4. Adaptive Depth

Instead of a fixed `--depth`, detect tree complexity and adaptively limit depth:

```go
func adaptiveDepth(elements []model.Element) int {
    totalCount := countElements(elements)
    if totalCount > 2000 {
        return 5 // very large tree: limit to prevent overwhelming output
    }
    if totalCount > 500 {
        return 8
    }
    return 0 // unlimited for small trees
}
```

Apply adaptive depth only when `--depth` is not explicitly set.

### 5. Response Header with Applied Defaults

Show which smart defaults were applied so the agent understands the output:

```yaml
ok: true
action: read
app: Google Chrome
smart_defaults: "auto-pruned (web content detected), agent format (piped output)"
elements:
    ...
```

This transparency helps agents learn the optimal flags and override when needed.

### 6. `--raw` Flag to Disable All Smart Defaults

```bash
# Smart defaults applied:
desktop-cli read --app "Chrome"

# No smart defaults — explicit control:
desktop-cli read --app "Chrome" --raw
```

### 7. Usage Examples

```bash
# Before: agent must know all the right flags
desktop-cli read --app "Chrome" --format agent --prune --roles "btn,lnk,input,txt"

# After: tool detects web content and applies optimal defaults
desktop-cli read --app "Chrome"
# Auto-applies: --prune (web content), --format agent (piped output)
# Response header: smart_defaults: "auto-pruned, agent format"

# Override any auto-applied default:
desktop-cli read --app "Chrome" --no-prune          # disable auto-prune
desktop-cli read --app "Chrome" --format yaml        # force YAML
desktop-cli read --app "Chrome" --depth 0            # force unlimited depth
desktop-cli read --app "Chrome" --raw                # disable everything
```

### 8. Auto-Interactive Roles for Web Content

When web content is detected, auto-expand `--roles "input"` to include `"other"` (Chrome exposes some web inputs as role "other"):

```go
// If targeting a web app and user specified --roles "input",
// auto-expand to include "other" (Chrome web input quirk)
if hasWebContent(elements) && containsRole(roles, "input") && !containsRole(roles, "other") {
    roles = append(roles, "other")
    // Note in response: 'auto-expanded roles: added "other" (Chrome web input compatibility)'
}
```

## Files to Create

- `internal/model/profiles.go` — App profiles and smart default logic

## Files to Modify

- `cmd/read.go` — Integrate smart defaults into read command
- `cmd/helpers.go` — Add web content detection and piped output detection
- `README.md` — Document smart defaults behavior
- `SKILL.md` — Update read examples (can now be simpler)

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] Web apps auto-enable `--prune` when not explicitly set
- [ ] Piped output auto-enables `--format agent` when not explicitly set
- [ ] `--raw` disables all smart defaults
- [ ] `--no-prune` explicitly disables auto-pruning
- [ ] `--format yaml` explicitly overrides piped output format
- [ ] Response includes `smart_defaults` field showing what was applied
- [ ] Web content detection works (checks for "web" role in tree)
- [ ] Adaptive depth applies for very large trees (>2000 elements)
- [ ] `--roles "input"` auto-expands to include "other" for web content
- [ ] No behavior change when running in terminal (non-piped) mode
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Backwards compatibility**: Smart defaults only activate when flags are NOT explicitly set. Any explicit flag takes precedence. This means existing scripts/agents continue to work identically.
- **Detection order**: Read elements first (needed anyway), then detect web content, then apply smart defaults, then filter/format. The detection adds negligible overhead (~1ms tree scan).
- **Piped output detection**: `os.Stdout.Stat()` checks if stdout is a terminal or pipe. When an agent calls the CLI, stdout is always a pipe. When a human runs it in a terminal, stdout is a TTY. This is a reliable heuristic.
- **`--no-prune` vs `--prune`**: Need a way to explicitly disable auto-pruning. Use `--no-prune` (cobra supports `--no-` prefix for bool flags). Or use `--prune=false`.
- **Adaptive depth risk**: Limiting depth might hide elements the agent needs. Only apply for very large trees (>2000 elements) and always mention it in the response header. The agent can override with `--depth 0`.
- **Profile updates**: As new apps are tested, profiles can be added. But the web content detection heuristic is more general and handles unknown browsers too.
- **Depends on**: No hard dependencies. Can be implemented independently.
