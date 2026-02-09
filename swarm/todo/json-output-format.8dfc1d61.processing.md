# Feature: Add JSON Output Format with `--format` Flag

## Priority: HIGH (Agent interoperability — many agent frameworks expect JSON)

## Problem

The `desktop-cli` tool currently outputs **only YAML**. The original PLAN.md designed the entire tool around JSON output (all examples use JSON, the schema section is titled "JSON Output Schema"), but the implementation switched to YAML-only. This creates several problems:

1. **Agent framework compatibility** — Most LLM tool-use frameworks (OpenAI, Claude, etc.) parse JSON natively. YAML requires an extra parsing step or custom handling.
2. **Token efficiency regression** — JSON is more compact than YAML for structured data. The PLAN.md's token efficiency analysis was based on JSON. YAML adds whitespace and newlines that inflate token counts.
3. **No `json` struct tags** — The `model.Element` struct only has `yaml` tags, so even `encoding/json` would produce verbose full-name keys instead of the compact single-letter keys designed in PLAN.md.
4. **Programmatic consumption** — JSON is the lingua franca of CLI tools consumed by scripts and agents. `jq` is ubiquitous; YAML tooling is less common.

## What to Build

### 1. Add `json` Struct Tags to All Output Types

**`internal/model/element.go`** — Add `json` tags matching the compact `yaml` tags:

```go
type Element struct {
    ID          int       `yaml:"i"         json:"i"`
    Role        string    `yaml:"r"         json:"r"`
    Title       string    `yaml:"t,omitempty" json:"t,omitempty"`
    Value       string    `yaml:"v,omitempty" json:"v,omitempty"`
    Description string    `yaml:"d,omitempty" json:"d,omitempty"`
    Bounds      [4]int    `yaml:"b"         json:"b"`
    Focused     bool      `yaml:"f,omitempty" json:"f,omitempty"`
    Enabled     *bool     `yaml:"e,omitempty" json:"e,omitempty"`
    Selected    bool      `yaml:"s,omitempty" json:"s,omitempty"`
    Children    []Element `yaml:"c,omitempty" json:"c,omitempty"`
    Actions     []string  `yaml:"a,omitempty" json:"a,omitempty"`
}
```

**`internal/model/window.go`** — Add `json` tags:

```go
type Window struct {
    App     string `yaml:"app"     json:"app"`
    PID     int    `yaml:"pid"     json:"pid"`
    Title   string `yaml:"title"   json:"title"`
    ID      int    `yaml:"id"      json:"id"`
    Bounds  [4]int `yaml:"bounds"  json:"bounds"`
    Focused bool   `yaml:"focused" json:"focused"`
}
```

**`internal/output/yaml.go`** (rename to `output.go`) — Add `json` tags to `ReadResult`:

```go
type ReadResult struct {
    App      string          `yaml:"app,omitempty"    json:"app,omitempty"`
    PID      int             `yaml:"pid,omitempty"    json:"pid,omitempty"`
    Window   string          `yaml:"window,omitempty" json:"window,omitempty"`
    TS       int64           `yaml:"ts"               json:"ts"`
    Elements []model.Element `yaml:"elements"         json:"elements"`
}
```

**`cmd/*.go`** — Add `json` tags to all result structs (ClickResult, DragResult, ScrollResult, TypeResult, FocusResult, WaitResult, ScreenshotResult, and the appEntry struct in list.go):

Example for ClickResult:
```go
type ClickResult struct {
    OK     bool   `yaml:"ok"     json:"ok"`
    Action string `yaml:"action" json:"action"`
    X      int    `yaml:"x"      json:"x"`
    Y      int    `yaml:"y"      json:"y"`
    Button string `yaml:"button" json:"button"`
    Count  int    `yaml:"count"  json:"count"`
}
```

### 2. Add Print Functions — `internal/output/output.go`

Rename `yaml.go` to `output.go` and add a `PrintJSON` function alongside the existing `PrintYAML`. Also add a unified `Print` function that dispatches based on format:

```go
package output

import (
    "encoding/json"
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

// Format represents the output format.
type Format string

const (
    FormatYAML Format = "yaml"
    FormatJSON Format = "json"
)

// Default format (can be set by root command's --format flag)
var OutputFormat Format = FormatYAML

// Print serializes v to stdout in the current output format.
func Print(v interface{}) error {
    switch OutputFormat {
    case FormatJSON:
        return PrintJSON(v)
    case FormatYAML:
        return PrintYAML(v)
    default:
        return fmt.Errorf("unsupported output format: %s", OutputFormat)
    }
}

// PrintJSON serializes v to stdout as compact single-line JSON.
func PrintJSON(v interface{}) error {
    enc := json.NewEncoder(os.Stdout)
    enc.SetEscapeHTML(false)
    return enc.Encode(v)
}

// PrintYAML serializes v to stdout as YAML.
func PrintYAML(v interface{}) error {
    enc := yaml.NewEncoder(os.Stdout)
    if err := enc.Encode(v); err != nil {
        return fmt.Errorf("yaml encode: %w", err)
    }
    return enc.Close()
}
```

Key details:
- `PrintJSON` outputs compact single-line JSON (no pretty-printing by default) — maximally token-efficient
- `SetEscapeHTML(false)` prevents unnecessary escaping of `<`, `>`, `&` in element text
- `json.Encoder.Encode` appends a newline, matching shell conventions
- The `Print` function dispatches based on the global `OutputFormat` variable

### 3. Add `--format` Global Flag — `cmd/root.go`

Add a persistent (global) `--format` flag to the root command:

```go
func init() {
    rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version.Version, version.Commit, version.BuildDate)
    rootCmd.PersistentFlags().String("format", "yaml", "Output format: yaml, json")
}
```

Add a `PersistentPreRunE` to the root command that sets the output format before any subcommand runs:

```go
var rootCmd = &cobra.Command{
    Use:   "desktop-cli",
    Short: "Read and interact with desktop UI elements",
    Long:  "A CLI tool that lets AI agents read and interact with desktop UI elements via accessibility APIs.",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        format, _ := cmd.Flags().GetString("format")
        switch format {
        case "yaml":
            output.OutputFormat = output.FormatYAML
        case "json":
            output.OutputFormat = output.FormatJSON
        default:
            return fmt.Errorf("unsupported format: %s (use yaml or json)", format)
        }
        return nil
    },
}
```

### 4. Replace All `output.PrintYAML` Calls with `output.Print`

In every command file, replace `output.PrintYAML(...)` with `output.Print(...)`. This is a mechanical find-and-replace across all command files:

Files to update:
- `cmd/read.go` — `output.PrintYAML(result)` → `output.Print(result)`
- `cmd/list.go` — two calls to `output.PrintYAML(...)` → `output.Print(...)`
- `cmd/click.go` — `output.PrintYAML(ClickResult{...})` → `output.Print(ClickResult{...})`
- `cmd/typecmd.go` — `output.PrintYAML(...)` → `output.Print(...)`
- `cmd/focus.go` — `output.PrintYAML(...)` → `output.Print(...)`
- `cmd/scroll.go` — `output.PrintYAML(...)` → `output.Print(...)`
- `cmd/drag.go` — `output.PrintYAML(...)` → `output.Print(...)`
- `cmd/screenshot.go` — `output.PrintYAML(...)` → `output.Print(...)`
- `cmd/wait.go` — two calls to `output.PrintYAML(...)` → `output.Print(...)`

Do NOT remove `PrintYAML` — keep it as a public function for direct use and backward compatibility.

### 5. Handle `--pretty` Flag for JSON

The `read` and `list` commands have a `--pretty` flag that is currently a no-op for YAML. For JSON output, `--pretty` should produce indented JSON:

Add a `PrintPrettyJSON` function:
```go
func PrintPrettyJSON(v interface{}) error {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    enc.SetEscapeHTML(false)
    return enc.Encode(v)
}
```

And modify `Print` to accept a pretty option, or add a global `PrettyOutput` bool similar to `OutputFormat`:

```go
var PrettyOutput bool

func Print(v interface{}) error {
    switch OutputFormat {
    case FormatJSON:
        if PrettyOutput {
            return PrintPrettyJSON(v)
        }
        return PrintJSON(v)
    case FormatYAML:
        return PrintYAML(v)
    default:
        return fmt.Errorf("unsupported output format: %s", OutputFormat)
    }
}
```

Set `PrettyOutput` in the root `PersistentPreRunE` by checking if the `--pretty` flag was set. Since `--pretty` is defined on individual commands (not globally), check if it exists on the current command:

```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    // ... format handling ...
    if prettyFlag := cmd.Flags().Lookup("pretty"); prettyFlag != nil {
        if pretty, err := cmd.Flags().GetBool("pretty"); err == nil && pretty {
            output.PrettyOutput = true
        }
    }
    return nil
},
```

### 6. Update Tests

**`internal/output/yaml_test.go`** (rename to `output_test.go`) — Add JSON-specific tests:

```go
func TestPrintJSON(t *testing.T) {
    result := ReadResult{
        App:    "Safari",
        PID:    1234,
        Window: "GitHub",
        TS:     1707500000,
        Elements: []model.Element{
            {ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}},
        },
    }

    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    err := PrintJSON(result)
    w.Close()
    os.Stdout = old

    if err != nil {
        t.Fatal(err)
    }

    var buf bytes.Buffer
    buf.ReadFrom(r)
    out := buf.String()

    // JSON output should be single-line (compact)
    lines := strings.Split(strings.TrimSpace(out), "\n")
    if len(lines) != 1 {
        t.Errorf("compact JSON should be one line, got %d lines", len(lines))
    }

    // Verify it's valid JSON with correct short keys
    var decoded map[string]interface{}
    if err := json.Unmarshal([]byte(out), &decoded); err != nil {
        t.Fatalf("output is not valid JSON: %v", err)
    }
    if decoded["app"] != "Safari" {
        t.Errorf("app: got %v, want Safari", decoded["app"])
    }
}

func TestPrintJSON_CompactKeys(t *testing.T) {
    // Verify that JSON uses the same short keys as YAML
    elem := model.Element{ID: 1, Role: "btn", Title: "OK", Bounds: [4]int{10, 20, 100, 30}}
    data, err := json.Marshal(elem)
    if err != nil {
        t.Fatal(err)
    }
    out := string(data)
    // Should use short "i", "r", "t", "b" keys
    if !strings.Contains(out, `"i":1`) {
        t.Errorf("expected short key 'i' for ID, got: %s", out)
    }
    if !strings.Contains(out, `"r":"btn"`) {
        t.Errorf("expected short key 'r' for Role, got: %s", out)
    }
    if !strings.Contains(out, `"t":"OK"`) {
        t.Errorf("expected short key 't' for Title, got: %s", out)
    }
}

func TestPrint_FormatDispatch(t *testing.T) {
    // Test that Print respects OutputFormat
    data := map[string]string{"hello": "world"}

    // Test JSON
    OutputFormat = FormatJSON
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w
    Print(data)
    w.Close()
    os.Stdout = old

    var buf bytes.Buffer
    buf.ReadFrom(r)
    out := buf.String()
    if !strings.Contains(out, "{") {
        t.Errorf("JSON format should produce JSON, got: %s", out)
    }

    // Reset
    OutputFormat = FormatYAML
}
```

Also update `internal/model/element_test.go` to test JSON marshaling produces compact keys.

### 7. Update Documentation

**README.md** — Add `--format` to the usage section and mention JSON output:

In the intro features section, change:
- "Token efficient — Short YAML keys..." → "Token efficient — Short keys, YAML or JSON output..."

Add a "Output Formats" subsection:
```markdown
### Output format

```bash
# Default: YAML output (human-readable)
desktop-cli read --app "Safari"

# JSON output (compact, single-line — ideal for agent consumption)
desktop-cli read --app "Safari" --format json

# Pretty-printed JSON
desktop-cli read --app "Safari" --format json --pretty

# All commands support --format
desktop-cli list --windows --format json
desktop-cli click --id 5 --app "Safari" --format json
```
```

**SKILL.md** — Add `--format json` to the quick reference and note that JSON is available for all commands. Update the "YAML Output Keys" section title to "Output Keys" (they apply to both formats).

## Files to Create

None — all changes are to existing files.

## Files to Modify

- `internal/model/element.go` — Add `json` struct tags
- `internal/model/window.go` — Add `json` struct tags
- `internal/output/yaml.go` → rename to `output.go` — Add `PrintJSON`, `Print`, format dispatch
- `internal/output/yaml_test.go` → rename to `output_test.go` — Add JSON tests
- `cmd/root.go` — Add `--format` persistent flag and `PersistentPreRunE`
- `cmd/read.go` — Replace `PrintYAML` → `Print`
- `cmd/list.go` — Replace `PrintYAML` → `Print`, add `json` tags to `appEntry`
- `cmd/click.go` — Replace `PrintYAML` → `Print`, add `json` tags to `ClickResult`
- `cmd/typecmd.go` — Replace `PrintYAML` → `Print`, add `json` tags to result struct
- `cmd/focus.go` — Replace `PrintYAML` → `Print`, add `json` tags to result struct
- `cmd/scroll.go` — Replace `PrintYAML` → `Print`, add `json` tags to result struct
- `cmd/drag.go` — Replace `PrintYAML` → `Print`, add `json` tags to result struct
- `cmd/screenshot.go` — Replace `PrintYAML` → `Print`, add `json` tags to result struct
- `cmd/wait.go` — Replace `PrintYAML` → `Print`, add `json` tags to `WaitResult`
- `README.md` — Document `--format` flag and JSON output
- `SKILL.md` — Update quick reference with `--format json` option

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `desktop-cli read --app "Finder" --format json` outputs compact single-line JSON with short keys (`i`, `r`, `t`, `b`, etc.)
- [ ] `desktop-cli read --app "Finder" --format json --pretty` outputs indented JSON
- [ ] `desktop-cli read --app "Finder"` (no --format) still outputs YAML (backward compatible)
- [ ] `desktop-cli list --windows --format json` outputs JSON array of windows
- [ ] `desktop-cli click --x 100 --y 200 --format json` outputs JSON result `{"ok":true,"action":"click",...}`
- [ ] `desktop-cli --format invalid` returns error: "unsupported format: invalid (use yaml or json)"
- [ ] All commands respect `--format` flag: read, list, click, type, focus, scroll, drag, screenshot, wait
- [ ] JSON keys match YAML keys exactly (`i`, `r`, `t`, `v`, `d`, `b`, `f`, `e`, `s`, `c`, `a` for elements)
- [ ] `omitempty` behavior is consistent between JSON and YAML (empty fields omitted in both)
- [ ] README.md documents `--format` flag with examples
- [ ] SKILL.md mentions `--format json` option

## Implementation Notes

- **File renames**: `yaml.go` → `output.go` and `yaml_test.go` → `output_test.go`. Use `git mv` for clean history.
- **No new dependencies**: `encoding/json` is in the Go standard library.
- **Backward compatible**: Default format remains YAML. All existing agent workflows continue to work unchanged.
- **Token savings**: JSON `{"i":1,"r":"btn","t":"OK","b":[10,20,100,30]}` is ~45 chars vs YAML equivalent at ~60+ chars. For a typical 50-element tree, this saves ~20% in tokens.
- **`SetEscapeHTML(false)`**: Important to prevent `<`, `>`, `&` in element text from being escaped to `\u003c`, `\u003e`, `\u0026` — this would inflate token count unnecessarily.
- **Global format via package variable**: The `output.OutputFormat` variable is set once in `PersistentPreRunE` and read by `Print()`. This avoids threading a format parameter through every command. This is safe because the CLI is single-threaded.
- **Pretty printing**: For YAML, `--pretty` is already a no-op (YAML is always human-readable). For JSON, `--pretty` enables `enc.SetIndent("", "  ")`.
