//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework Foundation
#include <ApplicationServices/ApplicationServices.h>

static int is_trusted() {
    return AXIsProcessTrusted();
}
*/
import "C"
import "fmt"

// CheckAccessibilityPermission checks if the process has macOS accessibility permission.
// Returns an error with instructions if permission is not granted.
func CheckAccessibilityPermission() error {
	if C.is_trusted() == 0 {
		return fmt.Errorf(
			"accessibility permission required\n\n" +
				"Grant permission at: System Settings > Privacy & Security > Accessibility\n" +
				"Add your terminal app (e.g. Terminal.app, iTerm2, or the IDE running this command).\n" +
				"Then restart the terminal and try again.")
	}
	return nil
}

// IsAccessibilityTrusted returns true if the process has accessibility permission.
func IsAccessibilityTrusted() bool {
	return C.is_trusted() != 0
}
