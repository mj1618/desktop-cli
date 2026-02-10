# Test Result

## Status
FAIL

## Evidence

### Tests and build pass
```
$ go test ./...
ok  	github.com/mj1618/desktop-cli	(cached)
ok  	github.com/mj1618/desktop-cli/cmd	(cached)
ok  	github.com/mj1618/desktop-cli/internal/model	(cached)
ok  	github.com/mj1618/desktop-cli/internal/output	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform/darwin	(cached)

$ go build -o desktop-cli .
(success)
```

### Reproduction: --near flag still clicks wrong location

Command 1 (default --near):
```bash
$ ./desktop-cli click --text "Task 1: Buy groceries" --near --app "Notes"
ok: true
action: click
x: 1115
y: 576
```
Result: Click at (1115, 576) — far bottom-right, nowhere near the checkbox. Checkbox NOT checked.

Command 2 (--near-direction left):
```bash
$ ./desktop-cli click --text "Task 1: Buy groceries" --near --near-direction left --app "Notes"
ok: true
action: click
x: 846
y: 576
```
Result: Click at (846, 576) — still wrong location. Checkbox NOT checked.

### Visual confirmation
Screenshots taken before and after both click attempts confirm all three checkboxes (Task 1, 2, 3) remain unchecked.

### Root cause analysis
The `--near` logic has two fundamental problems in this scenario:

1. **Wrong text match**: The text "Task 1: Buy groceries" is matched from the **sidebar** note list (element 60 at bounds 672, 454, 161, 17), not from the actual note content. The note content is a single large `input` element (ID 2131) containing all checklist text as one value — individual lines are NOT separate accessibility elements.

2. **Checkboxes not in accessibility tree**: Apple Notes checklist checkboxes are not exposed as accessibility elements at all. So `findNearestInteractiveElement()` can never find them, and the fallback `nearFallbackOffset()` computes coordinates relative to the sidebar element (which is in the wrong part of the UI entirely).

The `--near` and `--near-direction` flags and the fallback offset logic were implemented correctly in terms of code, but they cannot solve this problem because the checkbox elements simply don't exist in the accessibility tree and the text match resolves to the sidebar rather than the note content area.

## Notes
- The code changes (findNearestInteractiveElement, nearFallbackOffset, --near-direction flag) are well-structured and the logic is sound for cases where interactive elements ARE in the tree but just not directly clickable by text
- For the Apple Notes checklist case specifically, a different approach is needed — likely coordinate-based clicking using screenshot analysis (e.g. screenshot-coords) to find the visual checkbox positions
- This bug is inherent to Apple Notes' accessibility tree limitations, not just a heuristic issue
