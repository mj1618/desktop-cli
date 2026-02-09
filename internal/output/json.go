package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mj1618/desktop-cli/internal/model"
)

// ReadResult is the top-level JSON output of the `read` command.
type ReadResult struct {
	App      string          `json:"app,omitempty"`
	PID      int             `json:"pid,omitempty"`
	Window   string          `json:"window,omitempty"`
	TS       int64           `json:"ts"`
	Elements []model.Element `json:"elements"`
}

// PrintJSON serializes v to stdout as JSON.
// If pretty is true, uses indentation; otherwise single-line.
func PrintJSON(v interface{}, pretty bool) error {
	enc := json.NewEncoder(os.Stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	return nil
}
