#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
#include "window_list.h"

// Helper: copy CFString to C string (caller frees).
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
        CGRect bounds = CGRectZero;
        CFDictionaryRef boundsDict = CFDictionaryGetValue(dict, kCGWindowBounds);
        if (boundsDict) {
            CGRectMakeWithDictionaryRepresentation(boundsDict, &bounds);
        }

        // Get layer
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

int cg_get_frontmost_pid(void) {
    // Use CGWindowListCopyWindowInfo to find the frontmost window's PID.
    // The first window at layer 0 is typically the frontmost.
    CFArrayRef windowList = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID
    );
    if (!windowList) return -1;

    int frontPid = -1;
    CFIndex count = CFArrayGetCount(windowList);
    for (CFIndex i = 0; i < count; i++) {
        CFDictionaryRef dict = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);

        CFNumberRef layerRef = CFDictionaryGetValue(dict, kCGWindowLayer);
        int layer = -1;
        if (layerRef) CFNumberGetValue(layerRef, kCFNumberIntType, &layer);

        if (layer == 0) {
            CFNumberRef pidRef = CFDictionaryGetValue(dict, kCGWindowOwnerPID);
            if (pidRef) CFNumberGetValue(pidRef, kCFNumberIntType, &frontPid);
            break;
        }
    }

    CFRelease(windowList);
    return frontPid;
}
