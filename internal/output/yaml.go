package output

import (
	"fmt"
	"os"

	"github.com/mj1618/desktop-cli/internal/model"
	"gopkg.in/yaml.v3"
)

// ReadResult is the top-level YAML output of the `read` command.
type ReadResult struct {
	App      string          `yaml:"app,omitempty"`
	PID      int             `yaml:"pid,omitempty"`
	Window   string          `yaml:"window,omitempty"`
	TS       int64           `yaml:"ts"`
	Elements []model.Element `yaml:"elements"`
}

// PrintYAML serializes v to stdout as YAML.
func PrintYAML(v interface{}) error {
	enc := yaml.NewEncoder(os.Stdout)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("yaml encode: %w", err)
	}
	return enc.Close()
}
