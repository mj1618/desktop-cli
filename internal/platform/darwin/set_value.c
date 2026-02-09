#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
#include "set_value.h"

// Helper: copy CFString to C string (caller frees).
static char* setval_cfstring_to_cstring(CFStringRef str) {
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
static char* setval_get_string_attr(AXUIElementRef elem, CFStringRef attr) {
    CFTypeRef value = NULL;
    AXError err = AXUIElementCopyAttributeValue(elem, attr, &value);
    if (err != kAXErrorSuccess || !value) {
        return strdup("");
    }
    if (CFGetTypeID(value) == CFStringGetTypeID()) {
        char* result = setval_cfstring_to_cstring((CFStringRef)value);
        CFRelease(value);
        return result;
    }
    CFRelease(value);
    return strdup("");
}

// Match a CGWindowID from an AXUIElement window.
static int setval_get_window_id(AXUIElementRef windowElem) {
    CGWindowID windowID = 0;
    extern AXError _AXUIElementGetWindow(AXUIElementRef, CGWindowID*);
    if (_AXUIElementGetWindow(windowElem, &windowID) == kAXErrorSuccess) {
        return (int)windowID;
    }
    return 0;
}

// Recursively traverse the tree in the same order as ax_traverse in accessibility.c,
// looking for the element at targetIndex. Returns the AXUIElementRef (retained) or NULL.
static AXUIElementRef setval_find_element_by_index(AXUIElementRef elem, int targetIndex,
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
                AXUIElementRef found = setval_find_element_by_index(child, targetIndex,
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

// Create the appropriate CFTypeRef value based on the current attribute type.
// Queries the element's current attribute value to determine the type.
static CFTypeRef create_typed_value(AXUIElementRef elem, CFStringRef attrName, const char* value) {
    // Check what type the current value is
    CFTypeRef currentValue = NULL;
    AXError err = AXUIElementCopyAttributeValue(elem, attrName, &currentValue);

    if (err == kAXErrorSuccess && currentValue) {
        CFTypeID typeID = CFGetTypeID(currentValue);
        CFRelease(currentValue);

        if (typeID == CFStringGetTypeID()) {
            return CFStringCreateWithCString(kCFAllocatorDefault, value, kCFStringEncodingUTF8);
        }
        if (typeID == CFNumberGetTypeID()) {
            double num = atof(value);
            return CFNumberCreate(kCFAllocatorDefault, kCFNumberDoubleType, &num);
        }
        if (typeID == CFBooleanGetTypeID()) {
            if (strcasecmp(value, "true") == 0 || strcmp(value, "1") == 0) {
                return kCFBooleanTrue;
            }
            return kCFBooleanFalse;
        }
    }

    // Check if this is a boolean attribute (AXSelected, AXFocused, etc.)
    // by looking at the attribute name
    char attrCStr[256];
    if (CFStringGetCString(attrName, attrCStr, sizeof(attrCStr), kCFStringEncodingUTF8)) {
        if (strcmp(attrCStr, "AXSelected") == 0 || strcmp(attrCStr, "AXFocused") == 0 ||
            strcmp(attrCStr, "AXEnabled") == 0) {
            if (strcasecmp(value, "true") == 0 || strcmp(value, "1") == 0) {
                return kCFBooleanTrue;
            }
            return kCFBooleanFalse;
        }
    }

    // Default: try as string
    return CFStringCreateWithCString(kCFAllocatorDefault, value, kCFStringEncodingUTF8);
}

int ax_set_value(pid_t pid, const char* windowTitle, int windowID,
                 int maxDepth, int elementIndex, const char* attributeName,
                 const char* value) {
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
            int wid = setval_get_window_id(win);
            if (wid != windowID) {
                continue;
            }
        }

        // Filter by window title if specified
        if (windowTitle && windowTitle[0] != '\0') {
            char* title = setval_get_string_attr(win, kAXTitleAttribute);
            int match = (title && strcasestr(title, windowTitle) != NULL);
            free(title);
            if (!match) {
                continue;
            }
        }

        // Search this window's element tree
        foundElement = setval_find_element_by_index(win, elementIndex, 1, maxDepth, &nextID);
    }

    CFRelease(windows);
    CFRelease(app);

    if (!foundElement) return -1;

    // Create the attribute name CFString
    CFStringRef attrName = CFStringCreateWithCString(kCFAllocatorDefault, attributeName, kCFStringEncodingUTF8);

    // Create the typed value
    CFTypeRef cfValue = create_typed_value(foundElement, attrName, value);
    if (!cfValue) {
        CFRelease(attrName);
        CFRelease(foundElement);
        return -1;
    }

    // Set the value
    AXError result = AXUIElementSetAttributeValue(foundElement, attrName, cfValue);

    // Clean up — don't release CFBoolean singletons
    if (cfValue != kCFBooleanTrue && cfValue != kCFBooleanFalse) {
        CFRelease(cfValue);
    }
    CFRelease(attrName);
    CFRelease(foundElement);

    return (result == kAXErrorSuccess) ? 0 : -1;
}
