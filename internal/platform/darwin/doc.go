//go:build darwin

// Package darwin provides macOS platform support using CoreGraphics and Accessibility APIs.
// All functionality requires CGo (Objective-C frameworks).
// When CGo is disabled, the package compiles as a no-op stub.
package darwin
