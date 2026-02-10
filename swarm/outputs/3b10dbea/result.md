# Test Result

## Status
PASS

## Evidence

The `type` command already supports typing numeric expressions directly into Calculator.app, reducing 11 commands to 1.

### Test 1: Expression with `=` in text
```
$ ./desktop-cli type --app "Calculator" --text "347*29+156="

ok: true
action: type
text: 347*29+156=
display:
    - i: 9
      r: txt
      v: ‎347‎×‎29‎+‎156
    - i: 11
      r: txt
      v: ‎10219
```
Result: 10219 (correct: 347*29+156 = 10219)

### Test 2: Expression with `--key enter`
```
$ ./desktop-cli type --app "Calculator" --text "100/4" --key "enter"

ok: true
action: type+key
text: 100/4
key: enter
display:
    - i: 9
      r: txt
      v: ‎100‎÷‎4
    - i: 11
      r: txt
      v: ‎25
```
Result: 25 (correct: 100/4 = 25)

### Visual Confirmation
Screenshots taken at `/tmp/calc_after.png` and `/tmp/calc_div.png` both show correct results on the Calculator display.

## Notes
- This feature already works with the existing `type` command — no code changes were needed. Calculator.app accepts keyboard input for digits and operators (`+`, `-`, `*`, `/`, `=`, `%`).
- Both `--text "expr="` and `--text "expr" --key "enter"` patterns work correctly.
- The display elements are returned in the response, showing both the expression and the result.
- The improvement is a documentation/workflow improvement: agents should use `type` for Calculator instead of multiple `action` commands.
