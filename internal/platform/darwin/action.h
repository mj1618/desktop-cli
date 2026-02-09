#ifndef ACTION_H
#define ACTION_H

#include <ApplicationServices/ApplicationServices.h>

// Perform an accessibility action on the element at the given traversal index
// within the specified app. Uses the same traversal order as ax_read_elements.
// pid: target process ID
// windowTitle: filter to window matching this title (NULL = no filter)
// windowID: filter to specific window ID (0 = no filter)
// maxDepth: max traversal depth (0 = unlimited), must match the read call
// elementIndex: element ID from read output (1-based)
// actionName: AX action name (e.g. "AXPress", "AXCancel")
// Returns 0 on success, -1 on failure.
int ax_perform_action(pid_t pid, const char* windowTitle, int windowID,
                      int maxDepth, int elementIndex, const char* actionName);

#endif
