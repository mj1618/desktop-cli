# Test Result

## Status
PASS

## Evidence

### 1. All unit tests pass
```
$ go test ./...
ok  	github.com/mj1618/desktop-cli
ok  	github.com/mj1618/desktop-cli/cmd
ok  	github.com/mj1618/desktop-cli/internal/model
ok  	github.com/mj1618/desktop-cli/internal/output
ok  	github.com/mj1618/desktop-cli/internal/platform
ok  	github.com/mj1618/desktop-cli/internal/platform/darwin
```

### 2. Dedicated filter tests pass
```
$ go test ./cmd/ -run TestFilterVisibleElements -v
=== RUN   TestFilterVisibleElements_RemovesZeroDimension
--- PASS: TestFilterVisibleElements_RemovesZeroDimension (0.00s)
=== RUN   TestFilterVisibleElements_AllVisible
--- PASS: TestFilterVisibleElements_AllVisible (0.00s)
=== RUN   TestFilterVisibleElements_AllZero
--- PASS: TestFilterVisibleElements_AllZero (0.00s)
=== RUN   TestFilterVisibleElements_Empty
--- PASS: TestFilterVisibleElements_Empty (0.00s)
PASS
```

### 3. Live reproduction with Chrome (flights from Sydney to London)

**Before fix (described in issue):** `desktop-cli click --text "Flights" --roles "lnk" --app "Google Chrome"` would show 20+ matching elements, most with zero-height bounds (off-screen/virtualized).

**After fix:**
```
$ ./desktop-cli click --text "Flights" --roles "lnk" --app "Google Chrome"
Error: multiple elements match text "Flights" with roles "lnk" — use --id, --exact, or --scope-id to narrow:
  id=92 lnk (417,214,68,48) desc="Flights"
  id=197 lnk (310,832,228,31) desc="Sydney To London Flights Qantas..."
  id=216 lnk (310,978,474,31) desc="Sydney to London (SYD – LON)..."
```

Only **3 visible elements** matched instead of 20+. All zero-dimension elements were correctly filtered out. The raw accessibility tree still contains many zero-height elements (confirmed via `--raw` output), but they are excluded from matching.

**With --exact flag, single match resolved successfully:**
```
$ ./desktop-cli click --text "Flights" --roles "lnk" --app "Google Chrome" --exact
ok: true
action: click
x: 451
y: 238
button: left
count: 1
```

The click successfully hit the "Flights" filter tab (id=92) at coordinates (451, 238).

### 4. Visual verification
Screenshots taken at `/tmp/test-flights.png` and `/tmp/coords-flights.png` confirm Chrome navigated to the Flights search results page after the click, proving the correct element was targeted.

## Notes
- The `filterVisibleElements` function in `cmd/helpers.go` correctly filters elements with `Bounds[2] == 0` (zero width) or `Bounds[3] == 0` (zero height)
- The filter is applied in both `resolveElementByText` and `resolveElementByTextFromTree`, including within auto-scoped overlay searches
- The filter does NOT apply to `--id` targeting, which is correct since the user explicitly chose that element
- Edge case: if ALL matching elements have zero dimensions, the result set becomes empty and the user gets "no element found" error, which is the correct behavior (those elements aren't clickable anyway)
- The improvement dramatically reduces false ambiguity — from 20+ matches to just 3 visible ones in the flights scenario
