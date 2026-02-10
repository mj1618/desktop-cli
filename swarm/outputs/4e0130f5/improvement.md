# Action command returns wrong target element in response

## Problem

When using `action --text "All Clear" --app "Calculator"`, the command successfully performs the action (clears the calculator), but the response includes the WRONG target element details.

**Expected behavior:** Response should show the element that was actually acted upon (ID 10, "All Clear")

**Actual behavior:** Response shows a different element (ID 12, "Per cent")

Command run:
```bash
desktop-cli action --text "All Clear" --app "Calculator"
```

Response received:
```yaml
ok: true
action: action
id: 12
name: press
target:
    i: 12
    r: btn
    d: Per cent
    b: [662, 805, 40, 40]
```

However, from `read --app "Calculator" --format agent`, we can see:
- ID 10 is "All Clear" button at position (570,805,40,40)
- ID 12 is "Per cent" button at position (662,805,40,40)

The action WORKED correctly (calculator was cleared), but the response metadata is wrong. The `target` object in the response should reflect ID 10 ("All Clear"), not ID 12 ("Per cent").

## Proposed Fix

Fix the `action` command to return the correct target element in the response. The bug appears to be in how the response is constructed after finding and acting on the matching element.

The issue is likely in `cmd/action.go` where the response builds the `target` field - it may be reading the wrong element ID or offset when constructing the response object.

The response's `target` field must accurately reflect the element that was actually found and acted upon, matching both the ID and all other properties (role, description, bounds).

## Reproduction

1. Open Calculator app
2. Run: `desktop-cli read --app "Calculator" --format agent` to see element IDs
3. Run: `desktop-cli action --text "All Clear" --app "Calculator"`
4. Observe that response shows ID 12 with description "Per cent" instead of ID 10 with description "All Clear"
5. Verify the action worked by taking a screenshot or reading the display value
