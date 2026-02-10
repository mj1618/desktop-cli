# Test Result

## Status
FAIL

## Evidence

### Setup
Created a checklist in Apple Notes with three items:
```
desktop-cli focus --app "Notes"
desktop-cli type --key "cmd+n"
desktop-cli type --text "Shopping List"
desktop-cli type --key "enter"
desktop-cli type --key "shift+cmd+l"
desktop-cli type --text "Buy milk" --key "enter"
desktop-cli type --text "Buy bread" --key "enter"
desktop-cli type --text "Buy eggs"
```

### Confirmed original problem still exists
```bash
desktop-cli read --app "Notes" --roles "chk" --flat
# Result: elements: []   (no checkboxes found)
```

Screenshot-coords shows NO bounding boxes around the visible checkboxes — they are completely absent from the accessibility tree.

### Tested --near flag
```bash
desktop-cli click --text "Buy milk" --near --app "Notes"
# Result: ok: true, clicked at x:1531 y:709
```

The click landed at (1531, 709) which is the center of the large text input element (the note body), NOT on the checkbox next to "Buy milk". The checkbox at approximately (534, 118) relative to the note area was NOT clicked.

Visual verification via screenshot confirmed: all three checkboxes remain unchecked after the `--near` click.

### Root cause of failure
The `--near` flag finds the nearest **interactive element in the accessibility tree** to the text match. However, Apple Notes checkboxes are **not in the accessibility tree at all**. Since there are no checkbox elements to find, `--near` selects the nearest available interactive element — which happens to be the text input (note body) or some toolbar button, not the checkbox.

The unit tests pass because they use a mock tree that includes checkbox elements with `r: chk`. In reality, Apple Notes does not expose these elements.

### Unit tests
All unit tests pass:
```
=== RUN   TestFindNearestInteractiveElement_ChecklistCheckbox  --- PASS
=== RUN   TestFindNearestInteractiveElement_SecondRow           --- PASS
=== RUN   TestFindNearestInteractiveElement_NoInteractive       --- PASS
```

### Build
`go build` and `go test ./...` both succeed.

## Notes

The `--near` flag is a well-implemented feature that could be useful in cases where interactive elements DO exist in the accessibility tree but lack text labels (e.g., unlabeled buttons near text). However, it does **not** solve the Apple Notes checkbox problem because the checkboxes are entirely absent from the accessibility tree.

To actually solve this, one would need either:
1. A vision-based approach (like the proposed `--find-visual` flag) that locates checkboxes visually in screenshots
2. A coordinate offset approach (e.g., `--click-offset-x -30` to click 30px left of matched text)
3. Documenting it as a known macOS accessibility limitation with a workaround using `click --x <x> --y <y>`

The `--near` flag itself is not harmful and the code quality is good, but it does not fix the stated problem.
