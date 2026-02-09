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

// Check screen recording permission. Returns 1 if granted, 0 if denied.
int cg_check_screen_recording(void);

#endif
