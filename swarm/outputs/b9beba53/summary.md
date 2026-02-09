# Task: read-text-filter — Completed

## What was done

Implemented `--text` and `--flat` flags for the `read` command to enable text-based element search and flat list output.

### Files created
- `internal/model/flatten.go` — `FlatElement` struct and `FlattenElements` function for converting element trees to flat lists with path breadcrumbs
- `internal/model/flatten_test.go` — Tests for flattening: basic, nested paths, ID preservation, field preservation, traversal order

### Files modified
- `internal/model/filter.go` — Added `FilterByText` function for case-insensitive text matching against title/value/description with recursive child search
- `internal/model/filter_test.go` — Added 10 tests for FilterByText: empty text, title match, case-insensitive, value match, description match, no match, substring, recursive children, parent match with non-matching children, ID preservation
- `internal/platform/types.go` — Added `Text` and `Flat` fields to `ReadOptions`
- `internal/output/yaml.go` — Added `ReadFlatResult` struct for flat output
- `cmd/read.go` — Added `--text` and `--flat` flags, wired up text filtering and flat output
- `cmd/read_test.go` — Added flag registration tests for `text` and `flat`
- `README.md` — Added `--text` and `--flat` examples to Read UI elements section
- `SKILL.md` — Added `--text` and `--flat` examples and `p` key to YAML output keys table

### Verification
- `go build ./...` — passes
- `go test ./...` — all tests pass
