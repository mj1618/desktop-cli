# Screenshot Command Implementation Complete

## What was done

Implemented the `screenshot` command for macOS, enabling AI agents to capture window or full-screen screenshots as a vision model fallback.

## Files Created

- `internal/platform/darwin/screenshot.h` — C header for screenshot functions
- `internal/platform/darwin/screenshot.c` — C implementation using CGWindowListCreateImage (loaded via dlsym to bypass macOS 15 SDK unavailability), with scaling and PNG/JPEG encoding via ImageIO
- `internal/platform/darwin/screenshotter.go` — Go `DarwinScreenshotter` implementing `platform.Screenshotter`, with window resolution, permission checking, and CGo wrapper

## Files Modified

- `internal/platform/types.go` — Added `ScreenshotOptions` struct
- `internal/platform/platform.go` — Added `Screenshotter` interface
- `internal/platform/provider.go` — Added `Screenshotter` field to `Provider` struct
- `internal/platform/darwin/init.go` — Registered `Screenshotter` in provider
- `cmd/screenshot.go` — Replaced stub with full `runScreenshot()` implementation with base64 stdout and file output
- `README.md` — Added screenshot section and screen recording permission requirement
- `SKILL.md` — Added screenshot quick reference and updated agent workflow

## Key Technical Decisions

- Used `dlsym` to dynamically load `CGWindowListCreateImage` at runtime because the macOS 15 SDK marks it as `API_UNAVAILABLE` (hard error, not suppressible via pragmas). The function still works at runtime.
- Added `--window-id` and `--pid` flags to the screenshot command for parity with other commands.
- Default output is base64-encoded PNG to stdout for easy agent consumption.
- Default scale is 0.5 (half resolution) for token efficiency.

## Verification

- `go build ./...` succeeds
- `go test ./...` passes
- Full-screen capture produces valid PNG (4288x1440 at 0.5 scale on Retina)
- App-specific capture works (tested with Google Chrome)
- JPEG output with quality setting works
- Scale 1.0 and 0.25 both produce correct dimensions
- Base64 stdout output is valid
