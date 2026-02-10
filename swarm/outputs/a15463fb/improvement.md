# Display elements lack semantic distinction between expression and result

## Problem

When typing a calculation into Calculator, the response includes multiple `display` elements without indicating which is the expression vs. the result:

```bash
$ desktop-cli type --app "Calculator" --text "347*29+156="
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
```

Both elements have `r: txt` with no semantic distinction. An AI agent must guess which is the result — by position (last element), by font size (height 36 vs 26), or by heuristic parsing. This is error-prone and requires extra logic.

## Proposed Fix

Add semantic metadata to distinguish display elements. Options:

**Option 1** — Add a `purpose` or `type` field to display elements:
```yaml
display:
    - i: 9
      r: txt
      type: expression
      v: ‎347‎×‎29‎+‎156
      b: [624, 731, 124, 26]
    - i: 11
      r: txt
      type: result
      v: ‎10219
      b: [664, 761, 83, 36]
```

**Option 2** — Add a top-level `result` field for Calculator:
```yaml
ok: true
action: type
text: 347*29+156=
result: ‎10219
display:
    - i: 9
      r: txt
      v: ‎347‎×‎29‎+‎156
      b: [624, 731, 124, 26]
    - i: 11
      r: txt
      v: ‎10219
      b: [664, 761, 83, 36]
```

**Option 3** — Mark the primary display element:
```yaml
display:
    - i: 9
      r: txt
      v: ‎347‎×‎29‎+‎156
      b: [624, 731, 124, 26]
    - i: 11
      r: txt
      v: ‎10219
      primary: true
      b: [664, 761, 83, 36]
```

Recommendation: **Option 1** is most flexible (works beyond Calculator) or **Option 2** is simplest for agents (direct result access).

## Reproduction

```bash
# Open Calculator
open -a Calculator
sleep 1

# Type a multi-step calculation
desktop-cli type --app "Calculator" --text "347*29+156="

# Observe: Two display elements, no way to distinguish which is the result
# Agent must use heuristics (last element, larger font, etc.)
```
