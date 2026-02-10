# Task: `open` Command — Open URLs, Files, and Apps

## Status: Completed

## What was done

Implemented the `open` command as specified in `swarm/todo/open-url-file-app-command.pending.md`.

### Files created
- **`cmd/open.go`** — New `open` command implementation

### Files modified
- **`cmd/do.go`** — Added `open` as a supported batch step type with `executeOpen()` function
- **`README.md`** — Added `open` command documentation section and feature bullet, updated `do` step list
- **`SKILL.md`** — Added `open` command quick reference, updated agent workflow (step 1), updated `do` step list

### Features implemented
- `desktop-cli open "https://example.com"` — positional arg auto-detects URL vs file path
- `--url` / `--file` flags for explicit mode
- `--app` to specify which application to use, or to launch an app by itself
- `--wait` + `--timeout` to wait for the application window to appear
- `--post-read` + `--post-read-delay` to read UI state after opening
- Batch support: `open` step in `do` command YAML (with `url`, `file`, `app` params)

### Validation
- `go build ./...` — passes
- `go test ./...` — all 112 tests pass
- `desktop-cli open --help` — shows correct usage and all flags
