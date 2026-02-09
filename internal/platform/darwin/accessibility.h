#ifndef ACCESSIBILITY_H
#define ACCESSIBILITY_H

#include <ApplicationServices/ApplicationServices.h>

typedef struct {
    int id;
    char* role;
    char* title;
    char* value;
    char* description;
    float x, y, width, height;
    int enabled;
    int focused;
    int selected;
    int parentID;    // -1 for root elements
    int actionCount;
    char** actions;
} AXElementInfo;

// Read the element tree for a given app PID.
// If windowTitle is non-NULL, filters to windows matching that substring.
// If windowID > 0, filters to the specific window ID.
// maxDepth of 0 means unlimited.
// Returns 0 on success, -1 on failure.
int ax_read_elements(pid_t pid, const char* windowTitle, int windowID,
                     int maxDepth, AXElementInfo** outElements, int* outCount);

// Free the element array returned by ax_read_elements.
void ax_free_elements(AXElementInfo* elements, int count);

#endif
