#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>
#include <string.h>
#include "window_focus.h"

int ns_activate_app(pid_t pid) {
    NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
    if (!app) {
        return -1;
    }
    BOOL ok = [app activateWithOptions:NSApplicationActivateAllWindows];
    return ok ? 0 : -1;
}

int ax_raise_window(pid_t pid, const char* windowTitle, int windowID) {
    // First activate the app
    if (ns_activate_app(pid) != 0) {
        return -1;
    }

    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (!app) {
        return -1;
    }

    CFTypeRef windowsValue = NULL;
    AXError err = AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, &windowsValue);
    if (err != kAXErrorSuccess || !windowsValue) {
        CFRelease(app);
        return -1;
    }

    if (CFGetTypeID(windowsValue) != CFArrayGetTypeID()) {
        CFRelease(windowsValue);
        CFRelease(app);
        return -1;
    }

    CFArrayRef windows = (CFArrayRef)windowsValue;
    CFIndex count = CFArrayGetCount(windows);
    int found = 0;

    // Get CGWindowID from AXUIElement using private API
    extern AXError _AXUIElementGetWindow(AXUIElementRef, CGWindowID*);

    for (CFIndex i = 0; i < count; i++) {
        AXUIElementRef win = (AXUIElementRef)CFArrayGetValueAtIndex(windows, i);

        // Match by window ID
        if (windowID > 0) {
            CGWindowID wid = 0;
            if (_AXUIElementGetWindow(win, &wid) == kAXErrorSuccess) {
                if ((int)wid != windowID) {
                    continue;
                }
            } else {
                continue;
            }
        }

        // Match by title substring
        if (windowTitle && windowTitle[0] != '\0') {
            CFTypeRef titleValue = NULL;
            AXError titleErr = AXUIElementCopyAttributeValue(win, kAXTitleAttribute, &titleValue);
            if (titleErr != kAXErrorSuccess || !titleValue) {
                continue;
            }
            if (CFGetTypeID(titleValue) != CFStringGetTypeID()) {
                CFRelease(titleValue);
                continue;
            }
            // Convert title to C string for comparison
            CFIndex len = CFStringGetLength((CFStringRef)titleValue);
            CFIndex maxSize = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
            char* title = (char*)malloc(maxSize);
            CFStringGetCString((CFStringRef)titleValue, title, maxSize, kCFStringEncodingUTF8);
            CFRelease(titleValue);

            int match = (strcasestr(title, windowTitle) != NULL);
            free(title);
            if (!match) {
                continue;
            }
        }

        // Raise this window
        AXUIElementPerformAction(win, kAXRaiseAction);
        AXUIElementSetAttributeValue(win, kAXMainAttribute, kCFBooleanTrue);
        found = 1;
        break;
    }

    CFRelease(windows);
    CFRelease(app);
    return found ? 0 : -1;
}

int ns_get_frontmost_app(char** outName, pid_t* outPid) {
    NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
    if (!app) {
        *outName = NULL;
        *outPid = 0;
        return -1;
    }

    *outPid = app.processIdentifier;

    NSString *name = app.localizedName;
    if (name) {
        const char *cstr = [name UTF8String];
        *outName = strdup(cstr);
    } else {
        *outName = strdup("");
    }

    return 0;
}
