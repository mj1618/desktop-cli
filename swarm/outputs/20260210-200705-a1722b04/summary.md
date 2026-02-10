# Task: Cross-App Element Search (`find` command)

## Agent: 1a542d5e | Task: 0fd38bfc | Iteration: 4 of 5

## What was done

Implemented the `find` command that searches for UI elements across all windows (or all windows of a specific app). This eliminates 2-3 round-trips when agents need to locate elements after cross-app interactions (dialogs, notifications, permission prompts).

## Files created

- `cmd/find.go` — New `find` command with cobra CLI, flags (--text, --roles, --app, --limit, --exact), cross-window search logic, and focused-first window ordering
- `cmd/find_test.go` — Unit tests for command registration, flag defaults, and the `sortWindowsFocusedFirst` helper

## Files modified

- `README.md` — Added "Find elements across windows" section with usage examples
- `SKILL.md` — Added find command reference and updated agent workflow step 6

## Key design decisions

1. **Focused windows first**: Windows are sorted with focused windows at the front, so the most likely target is searched first
2. **Result limit**: Default limit of 10 elements to prevent token waste when searching across many windows
3. **Reuses existing helpers**: Uses `collectLeafMatches` from `cmd/helpers.go` for text matching, and `model.ExpandRoles` for role expansion
4. **Grouped output**: Results are grouped by window (app, window title, PID, window ID) so the agent knows which window context to use for follow-up commands
5. **Graceful error handling**: Windows that fail to read are silently skipped (e.g. permission denied)

## Verification

- `go build ./...` — passes
- `go test -count=1 ./cmd/` — passes (8 new tests for find command)
- `go test ./...` — all packages pass
