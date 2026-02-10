package platform

import (
	"fmt"
	"runtime"
)

// Provider bundles all platform backends for the current OS.
type Provider struct {
	Reader           Reader
	Inputter         Inputter
	WindowManager    WindowManager
	Screenshotter    Screenshotter
	ActionPerformer  ActionPerformer
	ValueSetter      ValueSetter
	ClipboardManager ClipboardManager
}

// ErrUnsupported is returned on unsupported platforms.
var ErrUnsupported = fmt.Errorf("desktop-cli is not supported on %s/%s; supported: darwin/amd64, darwin/arm64", runtime.GOOS, runtime.GOARCH)

// NewProviderFunc is set by platform-specific packages via init().
// See internal/platform/darwin/init.go for the macOS registration.
var NewProviderFunc func() (*Provider, error)

// RequestPermissionsFunc is set by platform-specific packages via init().
// It triggers OS permission prompts (e.g. screen recording) at startup.
var RequestPermissionsFunc func()

// NewProvider returns a Provider for the current OS.
func NewProvider() (*Provider, error) {
	if NewProviderFunc == nil {
		return nil, ErrUnsupported
	}
	return NewProviderFunc()
}
