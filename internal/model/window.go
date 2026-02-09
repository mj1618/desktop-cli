package model

// Window represents an application window.
type Window struct {
	App     string `yaml:"app"`
	PID     int    `yaml:"pid"`
	Title   string `yaml:"title"`
	ID      int    `yaml:"id"`
	Bounds  [4]int `yaml:"bounds"`
	Focused bool   `yaml:"focused,omitempty"`
}
