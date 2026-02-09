# Drag Command Implementation — Complete

## Task
Implemented the `drag` command with macOS CGEvent drag simulation.

## Changes Made

### `internal/platform/darwin/inputter.go`
- Added `#include <unistd.h>` for `usleep`
- Added `cg_drag` C function to the CGo block: moves mouse to start, mouse-down, interpolates 20 drag steps with `kCGEventLeftMouseDragged` events over 100ms, then mouse-up
- Replaced stub `Drag()` method with real implementation calling `cg_drag`

### `cmd/drag.go`
- Replaced `notImplemented("drag")` stub with full `runDrag` implementation
- Added `DragResult` struct for YAML output
- Supports `--from-x/--from-y` and `--to-x/--to-y` coordinate flags
- Supports `--from-id/--to-id` element ID resolution (reads element tree, computes center of bounding box)
- Supports `--app` and `--window` scoping for element ID resolution
- Validates that at least some coordinates are specified

### `README.md`
- Added "Drag" section with usage examples (coordinates, element IDs, mixed)

### `SKILL.md`
- Added drag command to the quick reference

## Validation
- `go vet ./cmd/...` passes
- `go test ./cmd/... ./internal/model/... ./internal/output/...` passes
- `go build ./...` fails only due to pre-existing screenshot module errors (separate task `8b7ccda9`, not related to drag)
