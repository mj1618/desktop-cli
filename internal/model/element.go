package model

// Element represents a UI element in the accessibility tree.
type Element struct {
	ID          int       `yaml:"i"`                    // Sequential integer ID
	Role        string    `yaml:"r"`                    // Abbreviated role code
	Title       string    `yaml:"t,omitempty"`          // Visible label / title
	Value       string    `yaml:"v,omitempty"`          // Current value
	Description string    `yaml:"d,omitempty"`          // Accessibility description
	Bounds      [4]int    `yaml:"b"`                    // [x, y, width, height]
	Focused     bool      `yaml:"f,omitempty"`          // Has keyboard focus
	Enabled     *bool     `yaml:"e,omitempty"`          // nil or true = enabled (omit); false = disabled (include)
	Selected    bool      `yaml:"s,omitempty"`          // Is selected
	Children    []Element `yaml:"c,omitempty"`          // Child elements
	Actions     []string  `yaml:"a,omitempty"`          // Available actions
}
