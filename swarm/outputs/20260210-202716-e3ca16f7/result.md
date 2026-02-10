# Test Result

## Status
PASS

## Evidence

### Problem confirmed (before fix)
Opened TextEdit normally with `open -a TextEdit` — the Open file dialog appeared as expected:
- Screenshot at `/tmp/textedit_before.png` shows the Open dialog with "New Document", "Show Options", "Cancel", and "Open" buttons.

### Fix verified: `focus --new-document`
```
$ osascript -e 'quit app "TextEdit"'; sleep 2
$ open -a TextEdit && sleep 3 && ./desktop-cli focus --app "TextEdit" --new-document
ok: true
action: focus
app: TextEdit
new_document: true
```

Screenshot at `/tmp/textedit_after.png` confirms:
- The Open dialog was dismissed
- A blank "Untitled" document was created with the standard TextEdit toolbar (font selector, formatting buttons, ruler)
- No file-open dialog is visible

### New `open` command also works
```
$ ./desktop-cli open --app "TextEdit" --wait
ok: true
action: open
app: TextEdit
```

### Help output confirms the flag
```
$ ./desktop-cli focus --help
  --new-document    After focusing, dismiss any open dialog (Escape) and create a new document (Cmd+N)
```

### Build
`go build` succeeds without errors.

### Tests
`go test ./...` has one pre-existing failure in `TestCheckAssert_Gone_ElementNotFound` (nil pointer dereference in `resolveElementByText` when provider is nil) — this is unrelated to the improvement being tested. It's a bug in the assert test setup (zero-value `assertOptions` with no provider).

## Notes
- The `--new-document` flag works by pressing Escape (to dismiss dialogs) then Cmd+N (to create a new document), with sleeps between actions. This is a pragmatic approach.
- The `open` command is a separate addition that wraps macOS `open` for URLs, files, and apps — complementary to `focus --new-document`.
- Edge case: if the app doesn't support Cmd+N, the Escape + Cmd+N sequence may have unintended effects. This is acceptable since the flag is opt-in.
- The pre-existing test failure in `assert_test.go` should be fixed separately (provider nil check needed in `findAssertElement`).
