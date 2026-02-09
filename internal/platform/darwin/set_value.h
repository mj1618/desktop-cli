#ifndef SET_VALUE_H
#define SET_VALUE_H

#include <ApplicationServices/ApplicationServices.h>

// Set an accessibility attribute value on the element at the given traversal index.
// Uses the same traversal order as ax_read_elements.
// pid: target process ID
// windowTitle: filter to window matching this title (NULL = no filter)
// windowID: filter to specific window ID (0 = no filter)
// maxDepth: max traversal depth (0 = unlimited), must match the read call
// elementIndex: element ID from read output (1-based)
// attributeName: AX attribute name (e.g. "AXValue", "AXSelected", "AXFocused")
// value: string representation of the value to set
// Returns 0 on success, -1 on failure.
int ax_set_value(pid_t pid, const char* windowTitle, int windowID,
                 int maxDepth, int elementIndex, const char* attributeName,
                 const char* value);

#endif
