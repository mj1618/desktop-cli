package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mj1618/desktop-cli/internal/model"
	"gopkg.in/yaml.v3"
)

// Format represents the output format.
type Format string

const (
	FormatYAML Format = "yaml"
	FormatJSON Format = "json"
)

// OutputFormat is the current output format, set by the root command's --format flag.
var OutputFormat Format = FormatYAML

// PrettyOutput enables pretty-printing for JSON output.
var PrettyOutput bool

// ReadResult is the top-level output of the `read` command.
type ReadResult struct {
	App      string          `yaml:"app,omitempty"    json:"app,omitempty"`
	PID      int             `yaml:"pid,omitempty"    json:"pid,omitempty"`
	Window   string          `yaml:"window,omitempty" json:"window,omitempty"`
	TS       int64           `yaml:"ts"               json:"ts"`
	Elements []model.Element `yaml:"elements"         json:"elements"`
}

// ReadFlatResult is the top-level output when --flat is used.
type ReadFlatResult struct {
	App      string              `yaml:"app,omitempty"    json:"app,omitempty"`
	PID      int                 `yaml:"pid,omitempty"    json:"pid,omitempty"`
	Window   string              `yaml:"window,omitempty" json:"window,omitempty"`
	TS       int64               `yaml:"ts"               json:"ts"`
	Elements []model.FlatElement `yaml:"elements"         json:"elements"`
}

// Print serializes v to stdout in the current output format.
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

// PrintJSON serializes v to stdout as compact single-line JSON.
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// PrintPrettyJSON serializes v to stdout as indented JSON.
func PrintPrettyJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
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
