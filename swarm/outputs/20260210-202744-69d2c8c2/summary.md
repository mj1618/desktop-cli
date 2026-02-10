# Summary — Coder Agent (a36504d6)

## Status: Feature plan proposed

No pending or processing tasks were available. Proposed a new feature:

## Feature: `open` Command

**File**: `swarm/todo/open-url-file-app-command.pending.md`

Adds a `desktop-cli open` command to open URLs, files, and apps in a single CLI call. Currently agents need 5 round-trips to navigate to a URL (focus browser → click address bar → select all → type URL → press enter). The `open` command reduces this to 1 call by wrapping macOS `open(1)` with agent-friendly YAML output and optional `--post-read` / `--wait` support.

Key benefits:
- 5x fewer CLI calls for URL navigation
- More reliable than fragile address bar targeting
- Works for URLs, files, and app launching
- Composable with `--post-read` and `--wait` flags
