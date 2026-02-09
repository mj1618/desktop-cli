# Bug Review Summary

## Bug Found and Fixed: Missing nil guard on `provider.Reader`

**Files affected:** `cmd/list.go`, `cmd/read.go`

**Issue:** Both `runList` and `runRead` called methods on `provider.Reader` without checking if it was nil first. The `Provider` struct allows `Reader` to be nil (all fields are interfaces with no non-nil guarantee), and `cmd/click.go` already correctly guards against `provider.Reader == nil` before using it. Without this guard, if `NewProvider()` returns a provider with a nil `Reader`, both commands would panic with a nil pointer dereference.

**Fix:** Added `provider.Reader == nil` checks (with a descriptive error message) before the first use of `provider.Reader` in both `runList` and `runRead`, consistent with the existing pattern in `runClick`.

## No Other Bugs Found

The rest of the changes are correct:
- JSON-to-YAML migration in model structs, output, and tests is clean
- `findElementByID` recursive search is correct
- Bounding box center calculation in click.go is correct (`x + width/2`, `y + height/2`)
- Mouse button parsing and double-click logic are correct
- Focus command nil-checks `provider.WindowManager` properly
- YAML encoder is correctly closed after encoding
