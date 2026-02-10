# Test Result

## Status
FAIL

## Evidence

### Build fixes required
The code had build errors due to a partial refactoring (DoContext struct fields were exported but call sites in `do.go`, `do_test.go`, and `fill.go` still used lowercase names). Fixed those to get a clean build and all tests passing:

```
$ go test ./...
ok  	github.com/mj1618/desktop-cli	0.186s
ok  	github.com/mj1618/desktop-cli/cmd	0.602s
ok  	github.com/mj1618/desktop-cli/internal/model	(cached)
ok  	github.com/mj1618/desktop-cli/internal/output	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform/darwin	(cached)
```

### TextEdit (control case — works correctly)
```
$ ./desktop-cli type --text "Hello world" --app "TextEdit"
ok: true
action: type
text: Hello world
focused:
    i: 3
    r: input
    b: [185, 178, 586, 388]
```
The focused element correctly shows the text input area (`r: input`). The pre-capture approach works for apps with normal focus reporting.

### Reminders (the reported bug — still broken)
```
$ ./desktop-cli type --text " updated" --app "Reminders"
ok: true
action: type
text: ' updated'
focused:
    i: 9
    r: cell
    d: My Lists, Upgrade Available
    b: [231, 529, 260, 22]
```

The text was typed correctly into the reminder text field (confirmed via screenshot — "First reminder" became "First reminder updated"), but the `focused` field still reports `i: 9` ("My Lists, Upgrade Available" cell), which is exactly the misleading behavior described in the improvement.

### Root cause analysis
The pre-capture approach (`readFocusedElement` before typing) doesn't fix the issue because macOS Reminders marks **multiple** elements as `f: true` (focused) simultaneously in the accessibility tree:

- `i: 9` (cell "My Lists, Upgrade Available") — `f: true`
- `i: 14` (cell "Reminders") — `f: true`
- `i: 20` (cell "Test List") — `f: true`
- `i: 38` (cell "0 Completed") — `f: true`
- `i: 47` (cell "Incomplete") — `f: true`
- **`i: 50` (input, value "First reminder updated") — `f: true`** — actual target

The `findFocusedElement` function in `helpers.go:642` walks the tree depth-first and returns the **first** element with `Focused: true`, which is `i: 9`. The actual text input (`i: 50`) is deeper in the tree and never reached.

The fix should either:
1. Return the **deepest** (leaf-most) focused element instead of the first one found
2. Or prefer focused elements that are actionable input fields (`r: input`) over container elements (`r: cell`)

The pre-capture approach is directionally correct (it avoids post-typing focus drift), but the underlying `findFocusedElement` implementation returns the wrong element when multiple elements report focused.

## Notes
- The code change to `typecmd.go` is sound in principle (capture focus before typing), but the bug is in `findFocusedElement` picking the wrong element from multiple focused candidates
- TextEdit and other apps with single focused elements work correctly
- The build had several additional errors from a DoContext struct export refactoring that were fixed as part of testing (lowercase field names and function calls needed updating in `do.go`, `do_test.go`, `fill.go`)
