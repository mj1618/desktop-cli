//go:build darwin

package darwin

import "github.com/mj1618/desktop-cli/internal/platform"

func init() {
	platform.NewProviderFunc = func() (*platform.Provider, error) {
		reader := NewReader()
		inputter := NewInputter()
		windowManager := NewWindowManager(reader)
		screenshotter := NewScreenshotter(reader)
		actionPerformer := NewActionPerformer(reader)
		return &platform.Provider{
			Reader:          reader,
			Inputter:        inputter,
			WindowManager:   windowManager,
			Screenshotter:   screenshotter,
			ActionPerformer: actionPerformer,
		}, nil
	}
}
