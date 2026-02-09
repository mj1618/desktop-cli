#ifndef WINDOW_FOCUS_H
#define WINDOW_FOCUS_H

#include <sys/types.h>

// Activate (focus) an application by PID.
// Returns 0 on success, -1 on failure.
int ns_activate_app(pid_t pid);

// Raise a specific window by PID and title substring match.
// If windowTitle is non-NULL, matches by title substring.
// If windowID > 0, matches by CGWindowID.
// Returns 0 on success, -1 on failure.
int ax_raise_window(pid_t pid, const char* windowTitle, int windowID);

// Get the frontmost application's name and PID.
// Caller must free *outName.
// Returns 0 on success, -1 on failure.
int ns_get_frontmost_app(char** outName, pid_t* outPid);

#endif
