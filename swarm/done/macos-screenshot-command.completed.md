# Feature: Implement `screenshot` Command with macOS Screen Capture

## Priority: HIGH (Phase 4 — enables vision model fallback for agents)

## Problem

Agents using `desktop-cli` can read the accessibility tree with `read`, but some UI elements lack good accessibility labels, or agents need to verify visual layout. The `screenshot` command is currently a stub returning "not yet implemented". Without it, agents have no fallback to capture what's actually rendered on screen.

The `screenshot` command enables agents to capture a window (or full screen), scale it down for token efficiency, and send it to a vision model (e.g., Claude) for analysis. This is the primary "plan B" when the accessibility tree doesn't give enough context.

## What to Build

### 1. Screenshotter Interface — `internal/platform/platform.go`

Add a new `Screenshotter` interface to the platform abstraction:

```go
// ScreenshotOptions configures what to capture.
type ScreenshotOptions struct {
    App      string  // Capture frontmost window of this app
    Window   string  // Capture window matching this title substring
    WindowID int     // Capture window by system ID
    PID      int     // Capture frontmost window of this PID
    Format   string  // "png" or "jpg"
    Quality  int     // JPEG quality 1-100 (ignored for PNG)
    Scale    float64 // Scale factor 0.1-1.0 (default 0.5)
}

// Screenshotter captures screenshots.
type Screenshotter interface {
    // CaptureWindow captures a screenshot of a specific window or the full screen.
    // Returns the image bytes in the requested format.
    CaptureWindow(opts ScreenshotOptions) ([]byte, error)
}
```

Add `Screenshotter` to the `Provider` struct:

```go
type Provider struct {
    Reader        Reader
    Inputter      Inputter
    WindowManager WindowManager
    Screenshotter Screenshotter
}
```

### 2. macOS Screenshot Backend — `internal/platform/darwin/screenshotter.go`

Implement the `Screenshotter` interface using macOS `CGWindowListCreateImage` and `CGImage` APIs.

```go
//go:build darwin

package darwin

// DarwinScreenshotter implements platform.Screenshotter for macOS.
type DarwinScreenshotter struct {
    reader *DarwinReader
}

func NewScreenshotter(reader *DarwinReader) *DarwinScreenshotter {
    return &DarwinScreenshotter{reader: reader}
}
```

#### C Implementation — `internal/platform/darwin/screenshot.c` + `screenshot.h`

**`screenshot.h`:**
```c
#ifndef SCREENSHOT_H
#define SCREENSHOT_H

#include <CoreGraphics/CoreGraphics.h>

typedef struct {
    unsigned char* data;
    int length;
    int width;
    int height;
} ScreenshotResult;

// Capture a specific window by its CGWindowID.
// format: 0=PNG, 1=JPEG
// quality: 1-100 (only for JPEG)
// scale: 0.1-1.0
// Returns 0 on success, -1 on failure.
int cg_capture_window(int windowID, int format, int quality, float scale,
                      ScreenshotResult* result);

// Capture the full screen.
int cg_capture_screen(int format, int quality, float scale,
                      ScreenshotResult* result);

// Free screenshot result data.
void cg_free_screenshot(ScreenshotResult* result);

#endif
```

**`screenshot.c` implementation approach:**

1. **`cg_capture_window(windowID, ...)`:**
   - Call `CGWindowListCreateImage(CGRectNull, kCGWindowListOptionIncludingWindow, windowID, kCGWindowImageBoundsIgnoreFraming)` to capture just the specified window
   - This returns a `CGImageRef` of the window contents
   - If `scale < 1.0`, create a scaled-down bitmap context:
     - Get original width/height from `CGImageGetWidth()` / `CGImageGetHeight()`
     - Compute new size: `newW = width * scale`, `newH = height * scale`
     - Create `CGBitmapContextCreate(NULL, newW, newH, 8, 0, colorSpace, kCGImageAlphaPremultipliedLast)`
     - Set interpolation quality: `CGContextSetInterpolationQuality(ctx, kCGInterpolationHigh)`
     - Draw scaled: `CGContextDrawImage(ctx, CGRectMake(0, 0, newW, newH), image)`
     - Get scaled image: `CGBitmapContextCreateImage(ctx)`
   - Convert `CGImageRef` to PNG or JPEG data:
     - Create `CFMutableDataRef` destination
     - Create `CGImageDestinationRef` with `CGImageDestinationCreateWithData(data, format_uti, 1, NULL)`
       - PNG: `kUTTypePNG`
       - JPEG: `kUTTypeJPEG` with quality property
     - `CGImageDestinationAddImage(dest, image, properties)`
     - `CGImageDestinationFinalize(dest)`
   - Copy the `CFDataRef` bytes into the `ScreenshotResult`
   - Clean up all Core Graphics/Foundation objects

2. **`cg_capture_screen()`:**
   - Call `CGWindowListCreateImage(CGRectInfinite, kCGWindowListOptionOnScreenOnly, kCGNullWindowID, kCGWindowImageDefault)`
   - Same scaling and encoding flow as above

3. **Permission check:**
   - Use `CGPreflightScreenCaptureAccess()` (macOS 10.15+) to check screen recording permission
   - If not authorized, return -1 and let Go code provide a helpful error message
   - Note: Screen recording permission is separate from Accessibility permission

**Required frameworks** (add to CGo LDFLAGS):
- `CoreGraphics` (already linked)
- `ImageIO` (for `CGImageDestination`)
- `CoreServices` (for `kUTTypePNG`, `kUTTypeJPEG` — or use `UniformTypeIdentifiers` on macOS 11+)

Note: On macOS 12+, `kUTTypePNG` and `kUTTypeJPEG` from `CoreServices/LaunchServices` may be deprecated in favor of `UTType` from UniformTypeIdentifiers. For maximum compatibility, use the string constants directly: `CFSTR("public.png")` and `CFSTR("public.jpeg")`.

#### Go wrapper in `screenshotter.go`:

```go
func (s *DarwinScreenshotter) CaptureWindow(opts platform.ScreenshotOptions) ([]byte, error) {
    // Check screen recording permission
    // (Use CGPreflightScreenCaptureAccess via CGo)

    // Resolve the target window ID
    windowID := opts.WindowID
    if windowID == 0 {
        // Use the Reader's ListWindows to find the window, similar to how
        // click and type resolve targets
        windowID, err = s.resolveWindowID(opts)
    }

    // Set defaults
    scale := opts.Scale
    if scale <= 0 || scale > 1.0 {
        scale = 0.5
    }
    format := 0 // PNG
    if opts.Format == "jpg" || opts.Format == "jpeg" {
        format = 1
    }
    quality := opts.Quality
    if quality <= 0 || quality > 100 {
        quality = 80
    }

    var result C.ScreenshotResult
    var rc C.int

    if windowID != 0 {
        rc = C.cg_capture_window(C.int(windowID), C.int(format), C.int(quality),
            C.float(scale), &result)
    } else {
        rc = C.cg_capture_screen(C.int(format), C.int(quality),
            C.float(scale), &result)
    }

    if rc != 0 {
        return nil, fmt.Errorf("screenshot capture failed (check Screen Recording permission)")
    }
    defer C.cg_free_screenshot(&result)

    return C.GoBytes(unsafe.Pointer(result.data), C.int(result.length)), nil
}
```

### 3. Wire Screenshot Command — `cmd/screenshot.go`

Replace the stub with real logic:

```go
func runScreenshot(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    if err != nil {
        return err
    }
    if provider.Screenshotter == nil {
        return fmt.Errorf("screenshot not supported on this platform")
    }

    appName, _ := cmd.Flags().GetString("app")
    window, _ := cmd.Flags().GetString("window")
    output, _ := cmd.Flags().GetString("output")
    format, _ := cmd.Flags().GetString("format")
    quality, _ := cmd.Flags().GetInt("quality")
    scale, _ := cmd.Flags().GetFloat64("scale")

    opts := platform.ScreenshotOptions{
        App:     appName,
        Window:  window,
        Format:  format,
        Quality: quality,
        Scale:   scale,
    }

    data, err := provider.Screenshotter.CaptureWindow(opts)
    if err != nil {
        return err
    }

    // Output to file or stdout
    if output != "" {
        return os.WriteFile(output, data, 0644)
    }

    // Default: write to stdout as base64 for easy agent consumption
    encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
    _, err = encoder.Write(data)
    if err != nil {
        return err
    }
    encoder.Close()
    fmt.Println() // newline after base64
    return nil
}
```

### 4. Permission Check — `internal/platform/darwin/permissions.go`

Add a screen recording permission check alongside the existing accessibility check:

```go
// CheckScreenRecordingPermission checks if the process has macOS screen recording permission.
func CheckScreenRecordingPermission() error {
    // CGPreflightScreenCaptureAccess() returns true if permission is granted.
    // CGRequestScreenCaptureAccess() prompts the user (one-time).
    if C.cg_check_screen_recording() == 0 {
        return fmt.Errorf(
            "screen recording permission required\n\n" +
            "Grant permission at: System Settings > Privacy & Security > Screen Recording\n" +
            "Add your terminal app, then restart the terminal and try again.")
    }
    return nil
}
```

The C function:
```c
static int cg_check_screen_recording(void) {
    if (@available(macOS 10.15, *)) {
        return CGPreflightScreenCaptureAccess() ? 1 : 0;
    }
    return 1; // Pre-Catalina: no permission needed
}
```

### 5. Update Provider Registration — `internal/platform/darwin/init.go`

Add the screenshotter to the provider:

```go
func init() {
    platform.NewProviderFunc = func() (*platform.Provider, error) {
        reader := NewReader()
        inputter := NewInputter()
        windowManager := NewWindowManager(reader)
        screenshotter := NewScreenshotter(reader)
        return &platform.Provider{
            Reader:        reader,
            Inputter:      inputter,
            WindowManager: windowManager,
            Screenshotter: screenshotter,
        }, nil
    }
}
```

### 6. Update Documentation

Update README.md and SKILL.md with the new `screenshot` command examples. Add a section showing the screenshot workflow:

```bash
# Capture a specific app's window as PNG (default)
desktop-cli screenshot --app "Safari"

# Capture at full resolution
desktop-cli screenshot --app "Safari" --scale 1.0

# Capture as JPEG with custom quality
desktop-cli screenshot --app "Safari" --format jpg --quality 60

# Save to a file
desktop-cli screenshot --app "Safari" --output /tmp/safari.png

# Capture by window title
desktop-cli screenshot --window "GitHub"

# Capture the full screen
desktop-cli screenshot
```

## Files to Create

- `internal/platform/darwin/screenshot.c` — C implementation of CGImage capture, scaling, and encoding
- `internal/platform/darwin/screenshot.h` — C header for screenshot functions
- `internal/platform/darwin/screenshotter.go` — Go `DarwinScreenshotter` implementing `platform.Screenshotter`

## Files to Modify

- `internal/platform/platform.go` — Add `ScreenshotOptions` struct and `Screenshotter` interface
- `internal/platform/provider.go` — Add `Screenshotter` field to `Provider` struct (already in `platform.go` if combined)
- `internal/platform/darwin/init.go` — Register `Screenshotter` in provider
- `internal/platform/darwin/permissions.go` — Add `CheckScreenRecordingPermission()`
- `cmd/screenshot.go` — Replace stub with real `runScreenshot()` implementation
- `README.md` — Add screenshot command examples and agent workflow
- `SKILL.md` — Add screenshot command to quick reference

## Acceptance Criteria

- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on non-macOS (Screenshotter is nil in provider, command returns helpful error)
- [ ] `go test ./...` passes
- [ ] `desktop-cli screenshot` with no flags captures the full screen and outputs base64 PNG to stdout
- [ ] `desktop-cli screenshot --app "Finder"` captures Finder's frontmost window
- [ ] `desktop-cli screenshot --window "Downloads"` captures a window matching "Downloads"
- [ ] `desktop-cli screenshot --output /tmp/test.png` saves to file instead of stdout
- [ ] `desktop-cli screenshot --format jpg --quality 60` produces JPEG output
- [ ] `desktop-cli screenshot --scale 1.0` captures at full resolution
- [ ] `desktop-cli screenshot --scale 0.25` captures at quarter resolution
- [ ] Default scale is 0.5 (half resolution, for token efficiency)
- [ ] Missing Screen Recording permission produces a clear, actionable error message
- [ ] The output image is valid and viewable (PNG or JPEG)
- [ ] README.md and SKILL.md are updated with screenshot examples

## Implementation Notes

- **Screen Recording permission**: This is separate from Accessibility permission. The tool must check `CGPreflightScreenCaptureAccess()` before attempting capture and give a clear error if denied. The error should tell the user to go to System Settings > Privacy & Security > Screen Recording.
- **`CGWindowListCreateImage`**: This is the core API. It can capture a specific window by ID (`kCGWindowListOptionIncludingWindow`) or the full screen (`kCGWindowListOptionOnScreenOnly`). It does NOT require the window to be frontmost — it can capture any window, even if occluded, as long as Screen Recording permission is granted.
- **Window resolution**: Reuse the `DarwinReader.ListWindows()` method to resolve `--app`, `--window`, and `--pid` to a window ID, exactly like the `click` and `focus` commands do.
- **Scaling**: Scale down BEFORE encoding to reduce file size. Use `CGBitmapContextCreate` + `CGContextDrawImage` to resize. Default scale of 0.5 is a good balance between detail and token cost (a 1440x900 screen becomes 720x450, which is clear enough for vision models).
- **Base64 stdout default**: When no `--output` flag is given, output the image as base64-encoded PNG to stdout. This is the most convenient format for agents to pass to vision model APIs. Include a trailing newline.
- **ImageIO framework**: Required for `CGImageDestinationCreateWithData`. Add `-framework ImageIO` to the CGo LDFLAGS.
- **UTType strings**: Use `CFSTR("public.png")` and `CFSTR("public.jpeg")` directly instead of the deprecated `kUTTypePNG`/`kUTTypeJPEG` constants to avoid deprecation warnings and framework dependency issues.
- **Memory management**: Be careful to release all CGImage, CGContext, CFData, CGImageDestination objects. Use `CFRelease`/`CGImageRelease`/`CGContextRelease` as appropriate.
- **Retina displays**: `CGWindowListCreateImage` returns images at the backing store resolution (2x on Retina). The `--scale` flag applies on top of this, so `--scale 0.5` on a Retina display produces an image at native resolution (not 2x). This is usually what agents want.
- **No `--window-id` and `--pid` flags on the command yet**: The existing stub defines `--window`, `--app`, `--output`, `--format`, `--quality`, and `--scale` flags. You may need to add `--window-id` and `--pid` flags to `cmd/screenshot.go` `init()` for parity with other commands.
