# Bug Fix: Inconsistent --app/--window requirements across commands using --id

## Priority: MEDIUM (inconsistent UX, may cause confusion for agents)

## Problem

Commands that accept `--id` for element lookup have inconsistent requirements for `--app`/`--window` scoping:

| Command | `--id` scope requirement |
|---------|-------------------------|
| `click` | No requirement — reads elements from all apps (potentially slow/ambiguous) |
| `scroll` | Requires `--app` OR `--window` |
| `type` | Requires `--app` only (doesn't accept `--window` alone) |
| `drag` | No requirement for `--from-id`/`--to-id` |

This inconsistency means:
- `desktop-cli click --id 5` works (reads all apps)
- `desktop-cli scroll --id 5` fails with "requires --app or --window"
- `desktop-cli type --id 5` fails with "requires --app"
- `desktop-cli drag --from-id 5 --to-id 10` works (reads all apps)

## Expected Behavior

All commands should have the same scoping requirement when using `--id`. The recommended pattern (used by `scroll`) is: **require `--app` or `--window`** when using element ID lookup. This is the safest approach because:

1. Reading all apps is slow and returns potentially thousands of elements
2. Element IDs are only stable within a single app's element tree
3. Without scoping, the same ID number could match different elements in different apps

## Files to Modify

- `cmd/click.go` — Add `--app` or `--window` requirement when `--id > 0`
- `cmd/typecmd.go` — Change to accept `--app` OR `--window` (currently only accepts `--app`)
- `cmd/drag.go` — Add `--app` or `--window` requirement when `--from-id > 0` or `--to-id > 0`

## Dependencies

- None

## Acceptance Criteria

- [ ] `click --id 5` without `--app`/`--window` returns error asking to specify one
- [ ] `type --id 5 --window "GitHub"` works (currently requires `--app`)
- [ ] `drag --from-id 5 --to-id 10` without `--app`/`--window` returns error
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
