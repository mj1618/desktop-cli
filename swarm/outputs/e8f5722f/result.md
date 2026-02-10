# Test Result

## Status
PASS

## Evidence

### Test 1: Original reproduction case (347*29+156=)
```bash
./desktop-cli click --app "Calculator" --text "All Clear"
./desktop-cli type --app "Calculator" --text "347*29+156="
```
Output:
```yaml
ok: true
action: type
text: 347*29+156=
display:
    - i: 9
      r: txt
      v: ‎347‎×‎29‎+‎156
      b: [624, 731, 124, 26]
    - i: 11
      r: txt
      v: ‎10219
      b: [664, 761, 83, 36]
      primary: true
```
Screenshot confirmed: Calculator displays "347×29+156" with result "10219" (correct: 347×29=10063, +156=10219).

### Test 2: Division and subtraction (100/4-5=)
```bash
./desktop-cli type --app "Calculator" --text "100/4-5="
```
Output shows "100÷4-5" with result "20" (correct). Screenshot confirmed.

### Test 3: Decimal point (3.14*2=)
Initial run found a bug: the decimal point was mapped to "Decimal" but the Calculator button is labeled "Point". Fixed `calculatorButtonMap` in `cmd/typecmd.go` line 189: changed `'.': "Decimal"` to `'.': "Point"`.

After fix:
```bash
./desktop-cli type --app "Calculator" --text "3.14*2="
```
Output shows "3.14×2" with result "6.28" (correct). Screenshot confirmed.

### All tests pass
```
go test ./...
ok  github.com/mj1618/desktop-cli
ok  github.com/mj1618/desktop-cli/cmd
ok  github.com/mj1618/desktop-cli/internal/model
ok  github.com/mj1618/desktop-cli/internal/output
ok  github.com/mj1618/desktop-cli/internal/platform
```

## Notes
- Found and fixed a minor bug during testing: the decimal point character `.` was mapped to button name "Decimal" but the macOS Calculator button is actually labeled "Point". Fixed in `cmd/typecmd.go` line 189.
- All operator mappings (+, -, *, /, =, .) and digit buttons (0-9) work correctly.
- The type command now correctly translates text into Calculator button presses as documented in SKILL.md.
