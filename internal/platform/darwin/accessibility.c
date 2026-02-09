#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
#include "accessibility.h"

// Helper: copy CFString to C string (caller frees).
static char* ax_cfstring_to_cstring(CFStringRef str) {
    if (!str) return strdup("");
    CFIndex len = CFStringGetLength(str);
    CFIndex maxSize = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
    char* buf = (char*)malloc(maxSize);
    if (!CFStringGetCString(str, buf, maxSize, kCFStringEncodingUTF8)) {
        buf[0] = '\0';
    }
    return buf;
}

// Helper: get a string attribute from an AX element. Returns strdup("") if not available.
static char* ax_get_string_attr(AXUIElementRef elem, CFStringRef attr) {
    CFTypeRef value = NULL;
    AXError err = AXUIElementCopyAttributeValue(elem, attr, &value);
    if (err != kAXErrorSuccess || !value) {
        return strdup("");
    }
    if (CFGetTypeID(value) == CFStringGetTypeID()) {
        char* result = ax_cfstring_to_cstring((CFStringRef)value);
        CFRelease(value);
        return result;
    }
    // For non-string values (e.g., AXValue might be a number/bool), try description
    if (CFGetTypeID(value) == CFBooleanGetTypeID()) {
        int bval = CFBooleanGetValue((CFBooleanRef)value);
        CFRelease(value);
        return strdup(bval ? "true" : "false");
    }
    if (CFGetTypeID(value) == CFNumberGetTypeID()) {
        double dval = 0;
        CFNumberGetValue((CFNumberRef)value, kCFNumberDoubleType, &dval);
        CFRelease(value);
        char buf[64];
        snprintf(buf, sizeof(buf), "%g", dval);
        return strdup(buf);
    }
    CFRelease(value);
    return strdup("");
}

// Helper: get a boolean attribute (returns 0 or 1, defaults to defaultVal on error).
static int ax_get_bool_attr(AXUIElementRef elem, CFStringRef attr, int defaultVal) {
    CFTypeRef value = NULL;
    AXError err = AXUIElementCopyAttributeValue(elem, attr, &value);
    if (err != kAXErrorSuccess || !value) {
        return defaultVal;
    }
    int result = defaultVal;
    if (CFGetTypeID(value) == CFBooleanGetTypeID()) {
        result = CFBooleanGetValue((CFBooleanRef)value) ? 1 : 0;
    } else if (CFGetTypeID(value) == CFNumberGetTypeID()) {
        CFNumberGetValue((CFNumberRef)value, kCFNumberIntType, &result);
    }
    CFRelease(value);
    return result;
}

// Helper: get position and size of an element.
static void ax_get_bounds(AXUIElementRef elem, float* x, float* y, float* w, float* h) {
    *x = 0; *y = 0; *w = 0; *h = 0;

    CFTypeRef posValue = NULL;
    if (AXUIElementCopyAttributeValue(elem, kAXPositionAttribute, &posValue) == kAXErrorSuccess && posValue) {
        CGPoint point;
        if (AXValueGetValue((AXValueRef)posValue, kAXValueCGPointType, &point)) {
            *x = (float)point.x;
            *y = (float)point.y;
        }
        CFRelease(posValue);
    }

    CFTypeRef sizeValue = NULL;
    if (AXUIElementCopyAttributeValue(elem, kAXSizeAttribute, &sizeValue) == kAXErrorSuccess && sizeValue) {
        CGSize size;
        if (AXValueGetValue((AXValueRef)sizeValue, kAXValueCGSizeType, &size)) {
            *w = (float)size.width;
            *h = (float)size.height;
        }
        CFRelease(sizeValue);
    }
}

// Dynamic array for collecting elements during traversal.
typedef struct {
    AXElementInfo* items;
    int count;
    int capacity;
} ElementArray;

static void elem_array_init(ElementArray* arr) {
    arr->count = 0;
    arr->capacity = 256;
    arr->items = (AXElementInfo*)calloc(arr->capacity, sizeof(AXElementInfo));
}

static void elem_array_grow(ElementArray* arr) {
    arr->capacity *= 2;
    arr->items = (AXElementInfo*)realloc(arr->items, arr->capacity * sizeof(AXElementInfo));
}

static int elem_array_add(ElementArray* arr) {
    if (arr->count >= arr->capacity) {
        elem_array_grow(arr);
    }
    int idx = arr->count;
    arr->count++;
    memset(&arr->items[idx], 0, sizeof(AXElementInfo));
    return idx;
}

// Get action names for an element.
static void ax_get_actions(AXUIElementRef elem, char*** outActions, int* outCount) {
    *outActions = NULL;
    *outCount = 0;

    CFArrayRef actionNames = NULL;
    AXError err = AXUIElementCopyActionNames(elem, &actionNames);
    if (err != kAXErrorSuccess || !actionNames) {
        return;
    }

    CFIndex count = CFArrayGetCount(actionNames);
    if (count == 0) {
        CFRelease(actionNames);
        return;
    }

    char** actions = (char**)malloc(count * sizeof(char*));
    int validCount = 0;

    for (CFIndex i = 0; i < count; i++) {
        CFStringRef name = (CFStringRef)CFArrayGetValueAtIndex(actionNames, i);
        char* cname = ax_cfstring_to_cstring(name);
        if (cname && cname[0] != '\0') {
            actions[validCount++] = cname;
        } else {
            free(cname);
        }
    }

    CFRelease(actionNames);
    *outActions = actions;
    *outCount = validCount;
}

// Match a CGWindowID from an AXUIElement window.
// Returns the CGWindowID or 0 if not available.
static int ax_get_window_id(AXUIElementRef windowElem) {
    CGWindowID windowID = 0;
    // _AXUIElementGetWindow is a private but widely-used API for getting the CGWindowID.
    extern AXError _AXUIElementGetWindow(AXUIElementRef, CGWindowID*);
    if (_AXUIElementGetWindow(windowElem, &windowID) == kAXErrorSuccess) {
        return (int)windowID;
    }
    return 0;
}

// Recursively traverse the accessibility tree.
static void ax_traverse(AXUIElementRef elem, int parentID, int currentDepth,
                        int maxDepth, int* nextID, ElementArray* arr) {
    if (maxDepth > 0 && currentDepth > maxDepth) {
        return;
    }

    int idx = elem_array_add(arr);
    int myID = (*nextID)++;
    AXElementInfo* info = &arr->items[idx];

    info->id = myID;
    info->parentID = parentID;
    info->role = ax_get_string_attr(elem, kAXRoleAttribute);
    info->title = ax_get_string_attr(elem, kAXTitleAttribute);
    info->value = ax_get_string_attr(elem, kAXValueAttribute);
    info->description = ax_get_string_attr(elem, kAXDescriptionAttribute);
    info->enabled = ax_get_bool_attr(elem, kAXEnabledAttribute, 1);
    info->focused = ax_get_bool_attr(elem, kAXFocusedAttribute, 0);
    info->selected = ax_get_bool_attr(elem, kAXSelectedAttribute, 0);
    ax_get_bounds(elem, &info->x, &info->y, &info->width, &info->height);
    ax_get_actions(elem, &info->actions, &info->actionCount);

    // Get children and recurse
    if (maxDepth == 0 || currentDepth < maxDepth) {
        CFTypeRef children = NULL;
        AXError err = AXUIElementCopyAttributeValue(elem, kAXChildrenAttribute, &children);
        if (err == kAXErrorSuccess && children && CFGetTypeID(children) == CFArrayGetTypeID()) {
            CFArrayRef childArray = (CFArrayRef)children;
            CFIndex childCount = CFArrayGetCount(childArray);
            for (CFIndex i = 0; i < childCount; i++) {
                AXUIElementRef child = (AXUIElementRef)CFArrayGetValueAtIndex(childArray, i);
                ax_traverse(child, myID, currentDepth + 1, maxDepth, nextID, arr);
            }
        }
        if (children) CFRelease(children);
    }
}

int ax_read_elements(pid_t pid, const char* windowTitle, int windowID,
                     int maxDepth, AXElementInfo** outElements, int* outCount) {
    *outElements = NULL;
    *outCount = 0;

    AXUIElementRef app = AXUIElementCreateApplication(pid);
    if (!app) {
        return -1;
    }

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

    ElementArray arr;
    elem_array_init(&arr);
    int nextID = 1;

    for (CFIndex i = 0; i < windowCount; i++) {
        AXUIElementRef win = (AXUIElementRef)CFArrayGetValueAtIndex(windows, i);

        // Filter by window ID if specified
        if (windowID > 0) {
            int wid = ax_get_window_id(win);
            if (wid != windowID) {
                continue;
            }
        }

        // Filter by window title if specified
        if (windowTitle && windowTitle[0] != '\0') {
            char* title = ax_get_string_attr(win, kAXTitleAttribute);
            int match = (title && strcasestr(title, windowTitle) != NULL);
            free(title);
            if (!match) {
                continue;
            }
        }

        // Traverse this window's element tree
        ax_traverse(win, -1, 1, maxDepth, &nextID, &arr);
    }

    CFRelease(windows);
    CFRelease(app);

    *outElements = arr.items;
    *outCount = arr.count;
    return 0;
}

void ax_free_elements(AXElementInfo* elements, int count) {
    if (!elements) return;
    for (int i = 0; i < count; i++) {
        free(elements[i].role);
        free(elements[i].title);
        free(elements[i].value);
        free(elements[i].description);
        if (elements[i].actions) {
            for (int j = 0; j < elements[i].actionCount; j++) {
                free(elements[i].actions[j]);
            }
            free(elements[i].actions);
        }
    }
    free(elements);
}
