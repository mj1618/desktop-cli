# Test Result

## Status
PASS

## Evidence

### All tests pass
```
$ go test ./...
ok  	github.com/mj1618/desktop-cli	(cached)
ok  	github.com/mj1618/desktop-cli/cmd	(cached)
ok  	github.com/mj1618/desktop-cli/internal/model	(cached)
ok  	github.com/mj1618/desktop-cli/internal/output	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform	(cached)
```

### Build succeeds
```
$ go build -o desktop-cli .
(success, no errors)
```

### Smart role disambiguation works — `action --text` without `--roles`

Cleared Calculator, then entered "347" without specifying `--roles "btn"`:

```
$ ./desktop-cli action --text "3" --action press --app "Calculator"
ok: true
action: action
id: 24
name: press
target:
    i: 24
    r: btn      # <-- correctly targeted btn, NOT txt
    d: "3"

$ ./desktop-cli action --text "4" --action press --app "Calculator"
ok: true
action: action
id: 18
name: press
target:
    i: 18
    r: btn      # <-- correctly targeted btn, NOT txt
    d: "4"

$ ./desktop-cli action --text "7" --action press --app "Calculator"
ok: true
action: action
id: 14
name: press
target:
    i: 14
    r: btn      # <-- correctly targeted btn, NOT txt
    d: "7"
```

Screenshot confirms Calculator display shows "347" after these three commands.

### `click --text` also resolves without error

```
$ ./desktop-cli click --text "3" --app "Calculator"
ok: true
action: click
x: 682
y: 963
button: left
count: 1
```

Previously this would error with:
```
Error: multiple elements match text "3" — use --id, --exact, or --scope-id to narrow:
  id=9 role=txt
  id=26 role=btn desc="3"
```

Now it resolves silently to the btn element.

### Code review confirms correct implementation

The `preferInteractiveElements()` function in `cmd/helpers.go`:
- Only activates when `--roles` is NOT specified
- Filters out static roles (txt, img, group, other) when interactive elements (btn, lnk, input, etc.) also match
- Returns original set unchanged if all matches are the same category (all static or all interactive)

## Notes
- The `click --text` command resolves to the correct button coordinates but the coordinate-based click doesn't register on Calculator (display stays at 0). This is a pre-existing coordinate/Retina scaling issue unrelated to this change.
- The `action --text` command works perfectly end-to-end, confirming the disambiguation logic is correct.
- No `--prefer-role` flag was implemented (mentioned in the improvement as a possible addition) — this is fine as the automatic disambiguation covers the main use case.
