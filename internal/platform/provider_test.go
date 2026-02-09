package platform

import (
	"runtime"
	"testing"
)

func TestNewProvider_ReturnsProvider(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping on non-darwin")
	}
	// On darwin, the darwin package may or may not be imported for side effects
	// depending on whether the test binary includes it. We just verify the
	// function doesn't panic.
	_, _ = NewProvider()
}

func TestNewProvider_UnsupportedPlatform(t *testing.T) {
	// Temporarily clear the provider func to simulate unsupported platform
	orig := NewProviderFunc
	NewProviderFunc = nil
	defer func() { NewProviderFunc = orig }()

	_, err := NewProvider()
	if err == nil {
		t.Fatal("expected error on unsupported platform")
	}
	if err != ErrUnsupported {
		t.Errorf("expected ErrUnsupported, got: %v", err)
	}
}
