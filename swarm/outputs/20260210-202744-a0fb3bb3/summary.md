# Task Summary: Indexed Screenshot Read

## Task
Implemented `read --format screenshot` — a new output format that combines an annotated screenshot (with `[id]` labels) and a structured element list in a single response.

## What Was Done

### Files Modified
- **`cmd/read.go`** — Added `runReadScreenshot()` function that:
  - Captures a screenshot of the target window
  - Annotates it with `[id]` labels using `AnnotateScreenshotWithMode` with `LabelIDs` mode
  - Filters to interactive elements by default (`--all-elements` for everything)
  - Excludes zero-bound elements from annotations
  - Generates agent-format element list
  - Outputs as YAML with `image` (base64 or file path) and `elements` fields
  - Supports `--screenshot-output` to save image to file
  - Default scale 0.25 for token efficiency, JPEG by default

- **`internal/output/output.go`** — Added `FormatScreenshot` case to `Print()` function (falls back to YAML for non-read commands)

- **`README.md`** — Added "Screenshot Format" section documenting usage and output format

- **`SKILL.md`** — Added screenshot format documentation and updated agent workflow to recommend it for visual understanding

### Flags Used (already defined, now functional)
- `--format screenshot` — Triggers combined visual+structured output
- `--scale` (default 0.25) — Screenshot scale factor
- `--screenshot-output` — Save image to file instead of inline base64
- `--image-format` (default "jpg") — Image format
- `--quality` (default 80) — JPEG quality
- `--all-elements` — Label all elements (default: interactive only)

### Pre-existing Infrastructure Leveraged
- `AnnotateScreenshotWithMode()` with `LabelIDs` mode (already in `screenshot_coords_draw.go`)
- `ScreenshotReadResult` struct (already in `output.go`)
- `FormatScreenshot` constant (already in `output.go`)
- `flattenElementsForAnnotation()` (already in `screenshot_coords.go`)

## Verification
- `go build ./...` — passes
- `go test ./...` — all tests pass
