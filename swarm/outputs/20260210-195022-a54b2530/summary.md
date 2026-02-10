# Batch `do` Command — Implementation Summary

## Agent: 395056ea | Task: 89ccd581

## What was done

Implemented the `do` command that allows agents to batch multiple UI actions in a single CLI call, eliminating LLM round-trips between steps.

## Files created

- **cmd/do.go** — Full `do` command implementation (~480 lines)

## Files modified

- **README.md** — Added `do` command documentation with 5 usage examples
- **SKILL.md** — Added `do` command quick reference and updated agent workflow section

## Implementation details

The `do` command:
- Accepts a YAML list of steps on stdin
- Supports all 9 step types: `click`, `type`, `action`, `set-value`, `scroll`, `wait`, `focus`, `read`, `sleep`
- Initializes the platform provider once and reuses it across all steps (no per-step process overhead)
- `--app` and `--window` flags set defaults; per-step `app`/`window` keys override them
- `--stop-on-error` (default: true) halts on first failure with partial results
- Returns per-step results with target/focused element info
- Collects display elements once at the end (not per-step) to reduce output
- Each step handler calls the same lower-level platform APIs as the standalone commands

## Verification

- `go build ./...` — passes
- `go test ./...` — all tests pass
- `desktop-cli do --help` — shows correct usage and flags
