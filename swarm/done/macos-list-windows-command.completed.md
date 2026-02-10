# Feature: Implement `list --windows` Command with macOS Darwin Backend

## Priority: CRITICAL (Phase 1, Task 2 — first functional command)

## Problem

All 8 CLI commands are stubs returning "not yet implemented". The project has well-defined platform interfaces (`Reader`, `Inputter`, `WindowManager`) and model types (`Element`, `Window`), but zero platform implementations exist. No command actually does anything.

The `list --windows` command is the simplest possible end-to-end slice: it queries macOS for running windows and outputs structured JSON. Implementing it proves out the full pipeline from CLI → platform backend → JSON output, and gives agents their first usable command.

## What to Build

### 1. Platform Provider — `internal/platform/provider.go`

A factory that creates platform-specific backends based on the current OS. Commands use this to get `Reader`, `Inputter`, and `WindowManager` instances.

```go
package platform

// Provider bundles all platform backends for the current OS.
type Provider struct {
    Reader        Reader
    Inputter      Inputter
    WindowManager WindowManager
}

// NewProvider returns a Provider for the current OS.
// Returns an error if the current platform is unsupported.
func NewProvider() (*Provider, error) { ... }
```

On macOS, `NewProvider()` returns darwin-backed implementations. On unsupported platforms, it returns a clear error like `"desktop-cli is not supported on <os>/<arch>; supported: darwin/amd64, darwin/arm64"`.

Use build tags (`//go:build darwin`) so only the darwin implementation compiles on macOS.

### 2. Accessibility Permission Check — `internal/platform/darwin/permissions.go`

Before any accessibility API call, check that the process has permission. macOS requires explicit opt-in at System Settings > Privacy & Security > Accessibility.

```go
package darwin

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework ApplicationServices -framework Foundation
// #include <ApplicationServices/ApplicationServices.h>
// static int is_trusted() { return AXIsProcessTrusted(); }
import "C"

// CheckAccessibilityPermission checks if the process has accessibility permission.
// If not, returns an error with clear instructions for the user.
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
```

### 3. macOS Window Listing — `internal/platform/darwin/reader.go`

Implement the `Reader` interface's `ListWindows()` method using macOS `CGWindowListCopyWindowInfo` API.

```go
package darwin

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework ApplicationServices -framework CoreGraphics -framework Foundation
/*
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

// C helper struct for window info
typedef struct {
    int pid;
    int windowID;
    char* appName;
    char* title;
    float x, y, width, height;
    int onScreen;
    int layer;
} CGWindowInfo;

// Enumerate visible windows, return flat array
int cg_list_windows(CGWindowInfo** outWindows, int* outCount);
void cg_free_windows(CGWindowInfo* windows, int count);
*/
import "C"

type DarwinReader struct{}

func NewReader() *DarwinReader {
    return &DarwinReader{}
}

// ListWindows returns all windows visible on screen using CGWindowListCopyWindowInfo.
// Filters by app name and PID per ListOptions.
func (r *DarwinReader) ListWindows(opts platform.ListOptions) ([]model.Window, error) {
    if err := CheckAccessibilityPermission(); err != nil {
        return nil, err
    }
    // Call CGo to enumerate windows
    // Convert C structs to model.Window
    // Apply filters (opts.App, opts.PID)
    // Return sorted by focused first, then app name
    ...
}

// ReadElements is a stub for now — to be implemented when `read` command is built.
func (r *DarwinReader) ReadElements(opts platform.ReadOptions) ([]model.Element, error) {
    return nil, fmt.Errorf("read not yet implemented")
}
```

#### C Implementation — `internal/platform/darwin/window_list.c`

```c
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
    int pid;
    int windowID;
    char* appName;
    char* title;
    float x, y, width, height;
    int onScreen;
    int layer;
} CGWindowInfo;

// Helper: copy CFString to C string (caller frees)
static char* cfstring_to_cstring(CFStringRef str) {
    if (!str) return strdup("");
    CFIndex len = CFStringGetLength(str);
    CFIndex maxSize = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
    char* buf = (char*)malloc(maxSize);
    if (!CFStringGetCString(str, buf, maxSize, kCFStringEncodingUTF8)) {
        buf[0] = '\0';
    }
    return buf;
}

int cg_list_windows(CGWindowInfo** outWindows, int* outCount) {
    CFArrayRef windowList = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID
    );
    if (!windowList) {
        *outWindows = NULL;
        *outCount = 0;
        return -1;
    }

    CFIndex count = CFArrayGetCount(windowList);
    CGWindowInfo* windows = (CGWindowInfo*)calloc(count, sizeof(CGWindowInfo));
    int validCount = 0;

    for (CFIndex i = 0; i < count; i++) {
        CFDictionaryRef dict = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);

        // Get PID
        CFNumberRef pidRef = CFDictionaryGetValue(dict, kCGWindowOwnerPID);
        int pid = 0;
        if (pidRef) CFNumberGetValue(pidRef, kCFNumberIntType, &pid);

        // Get window ID
        CFNumberRef widRef = CFDictionaryGetValue(dict, kCGWindowNumber);
        int wid = 0;
        if (widRef) CFNumberGetValue(widRef, kCFNumberIntType, &wid);

        // Get app name
        CFStringRef appName = CFDictionaryGetValue(dict, kCGWindowOwnerName);

        // Get title
        CFStringRef title = CFDictionaryGetValue(dict, kCGWindowName);

        // Get bounds
        CGRect bounds;
        CFDictionaryRef boundsDict = CFDictionaryGetValue(dict, kCGWindowBounds);
        if (boundsDict) {
            CGRectMakeWithDictionaryRepresentation(boundsDict, &bounds);
        }

        // Get layer (skip layer != 0 — those are system UI, menubar, etc.)
        CFNumberRef layerRef = CFDictionaryGetValue(dict, kCGWindowLayer);
        int layer = 0;
        if (layerRef) CFNumberGetValue(layerRef, kCFNumberIntType, &layer);

        // Get on-screen status
        CFBooleanRef onScreenRef = CFDictionaryGetValue(dict, kCGWindowIsOnscreen);
        int onScreen = onScreenRef ? CFBooleanGetValue(onScreenRef) : 0;

        windows[validCount].pid = pid;
        windows[validCount].windowID = wid;
        windows[validCount].appName = cfstring_to_cstring(appName);
        windows[validCount].title = cfstring_to_cstring(title);
        windows[validCount].x = bounds.origin.x;
        windows[validCount].y = bounds.origin.y;
        windows[validCount].width = bounds.size.width;
        windows[validCount].height = bounds.size.height;
        windows[validCount].onScreen = onScreen;
        windows[validCount].layer = layer;
        validCount++;
    }

    CFRelease(windowList);
    *outWindows = windows;
    *outCount = validCount;
    return 0;
}

void cg_free_windows(CGWindowInfo* windows, int count) {
    for (int i = 0; i < count; i++) {
        free(windows[i].appName);
        free(windows[i].title);
    }
    free(windows);
}
```

### 4. Wire `list` Command — `cmd/list.go`

Replace the `notImplemented("list")` stub with real logic:

```go
func runList(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }

    apps, _ := cmd.Flags().GetBool("apps")
    pid, _ := cmd.Flags().GetInt("pid")
    appName, _ := cmd.Flags().GetString("app")

    opts := platform.ListOptions{
        Apps: apps,
        PID:  pid,
        App:  appName,
    }

    windows, err := provider.Reader.ListWindows(opts)
    if err != nil {
        return err
    }

    pretty, _ := cmd.Flags().GetBool("pretty")
    return output.PrintJSON(windows, pretty)
}
```

Note: Add `--pretty` flag to the `list` command (currently missing but useful for debugging).

### 5. Build Tag Structure

Use build tags to keep darwin-specific code from compiling on other platforms:

- `internal/platform/darwin/*.go` — all files get `//go:build darwin`
- `internal/platform/provider_darwin.go` — `//go:build darwin` — returns darwin backends
- `internal/platform/provider_other.go` — `//go:build !darwin` — returns "unsupported platform" error

### 6. Unit Tests

- `internal/platform/darwin/permissions_test.go` — Test that `CheckAccessibilityPermission` doesn't panic (actual permission check depends on environment)
- `cmd/list_test.go` — Test flag parsing logic (mock Reader via interface)
- Integration test: `go build && ./desktop-cli list --windows` on a macOS machine should output real window JSON

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (with unsupported-platform stub)
- [ ] `go test ./...` passes on macOS
- [ ] `desktop-cli list` outputs JSON array of windows on macOS
- [ ] `desktop-cli list --windows` outputs JSON array of windows
- [ ] `desktop-cli list --app "Finder"` filters output to Finder windows only
- [ ] `desktop-cli list --pid <pid>` filters output by PID
- [ ] `desktop-cli list --apps` lists running applications (unique app names with PIDs)
- [ ] Output matches the JSON schema from PLAN.md: `[{"app":"...","pid":...,"title":"...","id":...,"bounds":[x,y,w,h],"focused":...},...]`
- [ ] Accessibility permission error is clear and actionable
- [ ] Running on Linux/Windows gives a clear "unsupported platform" error
- [ ] Other commands still return "not yet implemented"
- [ ] README.md updated if relevant
- [ ] SKILL.md updated if relevant

## Files to Create

- `internal/platform/provider_darwin.go` — macOS provider factory (build tag: darwin)
- `internal/platform/provider_other.go` — Unsupported platform stub (build tag: !darwin)
- `internal/platform/darwin/permissions.go` — Accessibility permission check via CGo
- `internal/platform/darwin/reader.go` — DarwinReader implementing Reader interface (ListWindows + ReadElements stub)
- `internal/platform/darwin/window_list.c` — C implementation for CGWindowListCopyWindowInfo
- `internal/platform/darwin/window_list.h` — C header for window list functions

## Files to Modify

- `cmd/list.go` — Replace stub with real implementation wired to platform provider
- `cmd/root.go` — Optionally add `--pretty` as a persistent flag (or add to `list` only)

## Notes

- `CGWindowListCopyWindowInfo` does NOT require accessibility permission for basic window info (app name, title, bounds, PID). It requires screen recording permission only for window contents/screenshots. So `list` may work even without accessibility permission — but we should still check and warn because other commands will need it.
- Actually, consider making the permission check a warning for `list` rather than a hard error, since `list` works without it. Save the hard error for `read`, `click`, `type` etc.
- The `--apps` flag should aggregate windows by app and return unique app entries: `[{"app":"Safari","pid":1234},{"app":"Terminal","pid":5678}]`
- Filter windows with `layer == 0` to exclude system UI elements, menubar overlays, etc. Only show real application windows.
- The "focused" field in window output should be determined by checking if the window's app is the frontmost app and the window is the key window.
- This feature establishes the CGo build pattern that all subsequent darwin backend work will follow.

---

## Completion Notes (Agent 63b20a16 / Task 77516bcc)

### What was implemented:

1. **Platform Provider** (`internal/platform/provider.go`) — Registration-based provider factory that avoids import cycles. Platform-specific packages register themselves via `init()`.

2. **Darwin Accessibility Permission Check** (`internal/platform/darwin/permissions.go`) — CGo-based `AXIsProcessTrusted()` check with clear user instructions on how to grant permission. Note: Not called from `list` since `CGWindowListCopyWindowInfo` works without accessibility permission. Available for future commands.

3. **Darwin Window Listing** (`internal/platform/darwin/reader.go` + `window_list.c` + `window_list.h`) — Full CGo implementation using `CGWindowListCopyWindowInfo`. Filters to layer 0 (real app windows), determines focused state via frontmost PID, applies --app and --pid filters, and sorts focused-first then alphabetically by app name.

4. **Darwin Init Registration** (`internal/platform/darwin/init.go`) — Registers the darwin provider via `init()`. Imported via build-tagged `platform_darwin.go` in the root package.

5. **List Command** (`cmd/list.go`) — Fully wired to platform provider with all flags: `--windows`, `--apps`, `--app`, `--pid`, `--pretty`. The `--apps` mode aggregates to unique `{"app","pid"}` entries.

6. **Build Tags** — `//go:build darwin` on all darwin files. `provider_other.go` replaced by the registration pattern (nil `NewProviderFunc` returns `ErrUnsupported`).

7. **Tests** — `cmd/list_test.go` (flag verification), `internal/platform/provider_test.go` (unsupported platform simulation).

8. **Docs** — Updated README.md and SKILL.md with list command examples.

### Build/Test Status:
- `go build ./...` passes
- `go test ./...` passes (all packages)
- `go vet ./...` passes
- Runtime testing requires a real macOS desktop (sandbox kills the process due to CGo framework loading)
