#ifndef ACCESSIBILITY_H
#define ACCESSIBILITY_H

#include <ApplicationServices/ApplicationServices.h>

typedef struct {
    int id;
    char* role;
    char* subrole;
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

// Window title info returned by ax_list_window_titles.
typedef struct {
    int windowID;
    char* title;
} AXWindowTitle;

// Get window titles for all windows of an application via accessibility API.
// Returns 0 on success, -1 on failure.
int ax_list_window_titles(pid_t pid, AXWindowTitle** outTitles, int* outCount);

// Free the window title array returned by ax_list_window_titles.
void ax_free_window_titles(AXWindowTitle* titles, int count);

#endif
