#ifndef WINDOW_LIST_H
#define WINDOW_LIST_H

#include <CoreGraphics/CoreGraphics.h>

typedef struct {
    int pid;
    int windowID;
    char* appName;
    char* title;
    float x, y, width, height;
    int onScreen;
    int layer;
} CGWindowInfo;

int cg_list_windows(CGWindowInfo** outWindows, int* outCount);
void cg_free_windows(CGWindowInfo* windows, int count);
int cg_get_frontmost_pid(void);

#endif
