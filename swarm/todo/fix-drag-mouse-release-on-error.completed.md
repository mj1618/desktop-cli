# Bug Fix: Drag `cg_drag` Doesn't Release Mouse on Error During Drag Loop

## Priority: HIGH (leaves system in broken state on failure)

## Problem

In `internal/platform/darwin/inputter.go`, the `cg_drag` C function posts a mouse-down event at the start of a drag (line 127), then iterates through interpolated drag steps. If `CGEventCreateMouseEvent` returns NULL during the drag loop (line 143-144), the function returns -1 immediately **without posting a mouse-up event**.

This leaves the system in a stuck "mouse down" state. All subsequent mouse interactions will behave as if the left button is held down until the user physically clicks to clear the state.

## Root Cause

```c
// inputter.go lines 137-148 (inside cg_drag C function)
for (int i = 1; i <= steps; i++) {
    float t = (float)i / (float)steps;
    float x = fromX + (toX - fromX) * t;
    float y = fromY + (toY - fromY) * t;
    CGPoint pt = CGPointMake(x, y);

    CGEventRef drag = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDragged, pt, kCGMouseButtonLeft);
    if (!drag) return -1;  // BUG: mouse-down already posted, mouse-up never sent
    CGEventPost(kCGHIDEventTap, drag);
    CFRelease(drag);

    usleep(delay_per_step);
}
```

The mouse-down was posted at line 127 but the error return at line 144 skips the mouse-up at lines 152-155.

## Fix

Replace the early `return -1` in the drag loop with code that releases the mouse button first:

```c
CGEventRef drag = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDragged, pt, kCGMouseButtonLeft);
if (!drag) {
    // Release mouse to avoid stuck mouse-down state
    CGEventRef upErr = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, pt, kCGMouseButtonLeft);
    if (upErr) {
        CGEventPost(kCGHIDEventTap, upErr);
        CFRelease(upErr);
    }
    return -1;
}
```

Also apply the same fix for the mouse-up creation failure at lines 152-155:

```c
CGEventRef up = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, endPoint, kCGMouseButtonLeft);
if (!up) return -1;  // This is less critical since it's the last step, but still bad
```

## Files to Modify

- `internal/platform/darwin/inputter.go` — Fix the `cg_drag` C function error handling in the CGo block

## Dependencies

- None (this is a standalone bug fix in existing code)

## Acceptance Criteria

- [ ] If `CGEventCreateMouseEvent` fails during the drag loop, mouse-up is posted before returning error
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
