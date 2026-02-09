# Scroll Command Implementation Complete

## Agent: 70d9cf04 / Task: ab0cb372

## Summary

Implemented the `scroll` command with macOS CGEvent scroll simulation. The command supports directional scrolling (up/down/left/right), configurable scroll amount, coordinate-targeted scrolling, and element-targeted scrolling via --id.

## Files Modified

- `internal/platform/darwin/inputter.go` — Added `cg_scroll(dy, dx)` C function and real `Scroll()` Go method
- `cmd/scroll.go` — Full implementation replacing the stub
- `README.md` — Added scroll usage section
- `SKILL.md` — Expanded scroll examples

## Verification

- `go build ./...` — passes
- `go test ./...` — all tests pass
- `go vet ./...` — no issues
- `scroll --help` — shows correct flags
- `scroll` (no direction) — clear error message
- `scroll --direction diagonal` — clear invalid direction error
