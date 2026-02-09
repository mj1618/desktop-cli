package model

// Element represents a UI element in the accessibility tree.
type Element struct {
	ID          int       `yaml:"i"           json:"i"`           // Sequential integer ID
	Role        string    `yaml:"r"           json:"r"`           // Abbreviated role code
	Title       string    `yaml:"t,omitempty" json:"t,omitempty"` // Visible label / title
	Value       string    `yaml:"v,omitempty" json:"v,omitempty"` // Current value
	Description string    `yaml:"d,omitempty" json:"d,omitempty"` // Accessibility description
	Bounds      [4]int    `yaml:"b"           json:"b"`           // [x, y, width, height]
	Focused     bool      `yaml:"f,omitempty" json:"f,omitempty"` // Has keyboard focus
	Enabled     *bool     `yaml:"e,omitempty" json:"e,omitempty"` // nil or true = enabled (omit); false = disabled (include)
	Selected    bool      `yaml:"s,omitempty" json:"s,omitempty"` // Is selected
	Children    []Element `yaml:"c,omitempty" json:"c,omitempty"` // Child elements
	Actions     []string  `yaml:"a,omitempty" json:"a,omitempty"` // Available actions
}
