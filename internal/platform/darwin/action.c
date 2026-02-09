#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
#include "action.h"

// Helper: copy CFString to C string (caller frees).
static char* action_cfstring_to_cstring(CFStringRef str) {
    if (!str) return strdup("");
    CFIndex len = CFStringGetLength(str);
    CFIndex maxSize = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
    char* buf = (char*)malloc(maxSize);
    if (!CFStringGetCString(str, buf, maxSize, kCFStringEncodingUTF8)) {
        buf[0] = '\0';
    }
    return buf;
}

// Helper: get a string attribute from an AX element.
static char* action_get_string_attr(AXUIElementRef elem, CFStringRef attr) {
    CFTypeRef value = NULL;
    AXError err = AXUIElementCopyAttributeValue(elem, attr, &value);
    if (err != kAXErrorSuccess || !value) {
        return strdup("");
    }
    if (CFGetTypeID(value) == CFStringGetTypeID()) {
        char* result = action_cfstring_to_cstring((CFStringRef)value);
        CFRelease(value);
        return result;
    }
    CFRelease(value);
    return strdup("");
}

// Match a CGWindowID from an AXUIElement window.
static int action_get_window_id(AXUIElementRef windowElem) {
    CGWindowID windowID = 0;
    extern AXError _AXUIElementGetWindow(AXUIElementRef, CGWindowID*);
    if (_AXUIElementGetWindow(windowElem, &windowID) == kAXErrorSuccess) {
        return (int)windowID;
    }
    return 0;
}

// Recursively traverse the tree in the same order as ax_traverse in accessibility.c,
// looking for the element at targetIndex. Returns the AXUIElementRef (retained) or NULL.
static AXUIElementRef find_element_by_index(AXUIElementRef elem, int targetIndex,
                                             int currentDepth, int maxDepth, int* nextID) {
    if (maxDepth > 0 && currentDepth > maxDepth) {
        return NULL;
    }

    int myID = (*nextID)++;

    if (myID == targetIndex) {
        CFRetain(elem);
        return elem;
    }

    // Recurse into children (same condition as ax_traverse)
    if (maxDepth == 0 || currentDepth < maxDepth) {
        CFTypeRef children = NULL;
        AXError err = AXUIElementCopyAttributeValue(elem, kAXChildrenAttribute, &children);
        if (err == kAXErrorSuccess && children && CFGetTypeID(children) == CFArrayGetTypeID()) {
            CFArrayRef childArray = (CFArrayRef)children;
            CFIndex childCount = CFArrayGetCount(childArray);
            for (CFIndex i = 0; i < childCount; i++) {
                AXUIElementRef child = (AXUIElementRef)CFArrayGetValueAtIndex(childArray, i);
                AXUIElementRef found = find_element_by_index(child, targetIndex,
                                                              currentDepth + 1, maxDepth, nextID);
                if (found) {
                    CFRelease(children);
                    return found;
                }
            }
        }
        if (children) CFRelease(children);
    }

    return NULL;
}

int ax_perform_action(pid_t pid, const char* windowTitle, int windowID,
                      int maxDepth, int elementIndex, const char* actionName) {
    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (!app) return -1;

    // Get windows
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
    CFIndex windowCount = CFArrayGetCount(windows);

    int nextID = 1;
    AXUIElementRef foundElement = NULL;

    // Iterate windows in the same order as ax_read_elements
    for (CFIndex i = 0; i < windowCount && !foundElement; i++) {
        AXUIElementRef win = (AXUIElementRef)CFArrayGetValueAtIndex(windows, i);

        // Filter by window ID if specified
        if (windowID > 0) {
            int wid = action_get_window_id(win);
            if (wid != windowID) {
                continue;
            }
        }

        // Filter by window title if specified
        if (windowTitle && windowTitle[0] != '\0') {
            char* title = action_get_string_attr(win, kAXTitleAttribute);
            int match = (title && strcasestr(title, windowTitle) != NULL);
            free(title);
            if (!match) {
                continue;
            }
        }

        // Search this window's element tree
        foundElement = find_element_by_index(win, elementIndex, 1, maxDepth, &nextID);
    }

    CFRelease(windows);
    CFRelease(app);

    if (!foundElement) return -1;

    // Perform the action
    CFStringRef action = CFStringCreateWithCString(kCFAllocatorDefault, actionName, kCFStringEncodingUTF8);
    AXError result = AXUIElementPerformAction(foundElement, action);

    CFRelease(action);
    CFRelease(foundElement);

    return (result == kAXErrorSuccess) ? 0 : -1;
}
