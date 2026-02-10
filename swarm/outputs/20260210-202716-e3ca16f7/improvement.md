# Application startup helper command

## Problem

When opening TextEdit (and likely other macOS apps), the app shows an "Open" file dialog by default. The current workflow requires:
1. Opening the app: `open -a TextEdit`
2. Waiting for it to load
3. Taking a screenshot to see what dialog appeared
4. Reading the UI with `desktop-cli read --app "TextEdit"`
5. Clicking "New Document" button OR pressing Escape to dismiss the dialog
6. If "New Document" is clicked, the dialog remains open (unexpected behavior)
7. Must use `Cmd+N` or close the dialog manually to get a blank document

This is 6-7 round trips just to start using the app with a blank document. The "New Document" button in the Open dialog is particularly confusing because clicking it doesn't actually create a new document and close the dialog - it does nothing visible.

## Proposed Fix

Add a `--new-document` or `--blank` flag to the `focus` command (or create a new `open` command) that:
1. Opens or focuses the application
2. Automatically dismisses any file-open dialogs with Escape
3. Creates a new blank document with `Cmd+N`
4. Waits for the document window to appear
5. Returns the window info

Example usage:
```bash
desktop-cli focus --app "TextEdit" --new-document
desktop-cli open --app "TextEdit" --blank
```

This would reduce the workflow from 6-7 commands to 1 command, making it much faster for agents to start working with applications.

Alternatively, add an `--auto-dismiss-dialogs` flag that automatically presses Escape when a file dialog appears after opening an app.

## Reproduction

1. Quit TextEdit if running: `osascript -e 'quit app "TextEdit"'`
2. Open TextEdit: `open -a TextEdit`
3. Wait 2 seconds: `sleep 2`
4. Take screenshot: `desktop-cli screenshot --app "TextEdit" --output /tmp/textedit.png`
5. Observe the "Open" dialog is showing
6. Click "New Document": `desktop-cli click --text "New Document" --app "TextEdit"`
7. Read UI again: `desktop-cli read --app "TextEdit"`
8. Observe the dialog is still open (button did nothing useful)
9. Must press Escape or use keyboard shortcuts to actually get a working document
