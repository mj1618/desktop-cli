//go:build darwin && cgo

package darwin

import "github.com/mj1618/desktop-cli/internal/platform"

func init() {
	platform.RequestPermissionsFunc = func() {
		RequestScreenRecordingPermission()
	}
	platform.NewProviderFunc = func() (*platform.Provider, error) {
		reader := NewReader()
		inputter := NewInputter()
		windowManager := NewWindowManager(reader)
		screenshotter := NewScreenshotter(reader)
		actionPerformer := NewActionPerformer(reader)
		valueSetter := NewValueSetter(reader)
		clipboard := NewClipboard()
		return &platform.Provider{
			Reader:           reader,
			Inputter:         inputter,
			WindowManager:    windowManager,
			Screenshotter:    screenshotter,
			ActionPerformer:  actionPerformer,
			ValueSetter:      valueSetter,
			ClipboardManager: clipboard,
		}, nil
	}
}
