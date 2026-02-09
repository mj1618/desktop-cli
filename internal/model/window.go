package model

// Window represents an application window.
type Window struct {
	App     string `yaml:"app"               json:"app"`
	PID     int    `yaml:"pid"               json:"pid"`
	Title   string `yaml:"title"             json:"title"`
	ID      int    `yaml:"id"                json:"id"`
	Bounds  [4]int `yaml:"bounds"            json:"bounds"`
	Focused bool   `yaml:"focused,omitempty" json:"focused,omitempty"`
}
