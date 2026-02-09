#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
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

// Helper: try to get text content via AXStringForRange parameterized attribute.
// This is a fallback for elements (like contenteditable divs) where AXValue is empty
// but the element has text content accessible via the text navigation attributes.
// Returns a malloc'd string on success, NULL if not available.
static char* ax_get_text_content(AXUIElementRef elem) {
    // Check if the element reports AXNumberOfCharacters
    CFTypeRef charCountValue = NULL;
    AXError err = AXUIElementCopyAttributeValue(elem, CFSTR("AXNumberOfCharacters"), &charCountValue);
    if (err != kAXErrorSuccess || !charCountValue) {
        return NULL;
    }

    CFIndex charCount = 0;
    if (CFGetTypeID(charCountValue) == CFNumberGetTypeID()) {
        CFNumberGetValue((CFNumberRef)charCountValue, kCFNumberCFIndexType, &charCount);
    }
    CFRelease(charCountValue);

    if (charCount <= 0) {
        return NULL;
    }

    // Cap to a reasonable length to avoid huge allocations
    if (charCount > 10000) {
        charCount = 10000;
    }

    // Create a CFRange and wrap it in an AXValueRef
    CFRange range = CFRangeMake(0, charCount);
    AXValueRef rangeValue = AXValueCreate(kAXValueCFRangeType, &range);
    if (!rangeValue) {
        return NULL;
    }

    // Use the parameterized attribute to get the string for that range
    CFTypeRef textValue = NULL;
    err = AXUIElementCopyParameterizedAttributeValue(elem,
        CFSTR("AXStringForRange"), rangeValue, &textValue);
    CFRelease(rangeValue);

    if (err != kAXErrorSuccess || !textValue) {
        return NULL;
    }

    if (CFGetTypeID(textValue) == CFStringGetTypeID()) {
        char* result = ax_cfstring_to_cstring((CFStringRef)textValue);
        CFRelease(textValue);
        return result;
    }

    CFRelease(textValue);
    return NULL;
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

    // Fallback: if AXValue is empty, try AXStringForRange parameterized attribute.
    // This captures text from contenteditable divs and rich-text editors (e.g. Gmail
    // compose body in Chrome) that don't expose text through kAXValueAttribute.
    if (info->value[0] == '\0') {
        char* textContent = ax_get_text_content(elem);
        if (textContent) {
            free(info->value);
            info->value = textContent;
        }
    }

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

// Activate enhanced UI mode for the application.
// This is required for Chrome/Chromium browsers which lazily activate their
// accessibility tree. Setting AXEnhancedUserInterface signals that an assistive
// technology is present, causing Chrome to expose web page content in the tree.
// Only sleeps on the first activation (when the attribute was not already set).
static void ax_activate_enhanced_ui(AXUIElementRef app) {
    CFTypeRef currentValue = NULL;
    AXError err = AXUIElementCopyAttributeValue(app,
        CFSTR("AXEnhancedUserInterface"), &currentValue);
    Boolean alreadyEnabled = false;
    if (err == kAXErrorSuccess && currentValue) {
        if (CFGetTypeID(currentValue) == CFBooleanGetTypeID()) {
            alreadyEnabled = CFBooleanGetValue((CFBooleanRef)currentValue);
        }
        CFRelease(currentValue);
    }
    if (!alreadyEnabled) {
        AXUIElementSetAttributeValue(app,
            CFSTR("AXEnhancedUserInterface"), kCFBooleanTrue);
        // Give the app time to build its accessibility tree
        usleep(200000); // 200ms
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

    // Activate enhanced UI to ensure Chrome exposes web content
    ax_activate_enhanced_ui(app);

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

int ax_list_window_titles(pid_t pid, AXWindowTitle** outTitles, int* outCount) {
    *outTitles = NULL;
    *outCount = 0;

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
    CFIndex windowCount = CFArrayGetCount(windows);
    if (windowCount == 0) {
        CFRelease(windows);
        CFRelease(app);
        return 0;
    }

    AXWindowTitle* titles = (AXWindowTitle*)calloc(windowCount, sizeof(AXWindowTitle));
    int validCount = 0;

    for (CFIndex i = 0; i < windowCount; i++) {
        AXUIElementRef win = (AXUIElementRef)CFArrayGetValueAtIndex(windows, i);

        int wid = ax_get_window_id(win);
        if (wid == 0) continue;

        char* title = ax_get_string_attr(win, kAXTitleAttribute);
        titles[validCount].windowID = wid;
        titles[validCount].title = title;
        validCount++;
    }

    CFRelease(windows);
    CFRelease(app);

    *outTitles = titles;
    *outCount = validCount;
    return 0;
}

void ax_free_window_titles(AXWindowTitle* titles, int count) {
    if (!titles) return;
    for (int i = 0; i < count; i++) {
        free(titles[i].title);
    }
    free(titles);
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
