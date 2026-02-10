# Test Result

## Status
FAIL

## Evidence

### Tests pass and build succeeds
```
$ go test ./...
ok  github.com/mj1618/desktop-cli	(cached)
ok  github.com/mj1618/desktop-cli/cmd	(cached)
ok  github.com/mj1618/desktop-cli/internal/model	(cached)
ok  github.com/mj1618/desktop-cli/internal/output	(cached)
ok  github.com/mj1618/desktop-cli/internal/platform	(cached)

$ go build -o desktop-cli .
(success)
```

### `--children` flag exists and works in YAML format
```
$ ./desktop-cli read --app "Google Chrome" --text "Results" --children
app: Google Chrome
ts: 1770720141
elements:
    - i: 170
      r: group
      d: MCA Cafe at Tallawoladah
      ...
    - i: 203
      r: group
      d: Dutch Smuggler Coffee Brewers
      ...
    - i: 238
      r: group
      d: Bennie's Cafe · Bistro
      ...
    (all 9 results listed)
```

### `--children` with `--format agent` returns empty output
```
$ ./desktop-cli read --app "Google Chrome" --text "Results" --children --format agent
# Google Chrome

(no elements shown)
```

This is because `--children` strips grandchildren, leaving only group elements with no "press" actions and no display text — both of which agent format requires. The agent format filter (line 135-138 of output.go) skips elements that are not interactive and not display text.

### Root cause
The `directChildrenOnly()` function sets `Children = nil` on each element. When the agent format then flattens and filters, it only sees the top-level group elements (which are anonymous groups with `showmenu`/`scrolltovisible` actions but no `press` action and no title/value text), so it filters them all out.

## Notes

- The `--children` flag is a good concept and partially works — YAML output correctly shows list items with their descriptions in a single command
- The primary use case in the improvement (reducing commands for agents using `--format agent`) is broken because agent format filters out the direct children which are typically intermediate group elements
- To fix: either (a) make `--children` keep one level deeper (preserving links/text inside each child), or (b) adjust agent format to include group elements that have a description, or (c) when `--children` is used, flatten each child's subtree to show its interactive elements
