# Test Result

## Status
PASS

## Evidence

### Tests pass
```
$ go test ./...
ok  	github.com/mj1618/desktop-cli	(cached)
ok  	github.com/mj1618/desktop-cli/cmd	(cached)
ok  	github.com/mj1618/desktop-cli/internal/model	(cached)
ok  	github.com/mj1618/desktop-cli/internal/output	(cached)
ok  	github.com/mj1618/desktop-cli/internal/platform	(cached)
```

### Reproduction
```
$ ./desktop-cli type --app "Calculator" --text "347*29+156="
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

The result element (`v: 10219`, height=36) is now marked with `primary: true`, while the expression element (`v: 347×29+156`, height=26) is not. This allows agents to unambiguously identify the calculation result.

### Visual confirmation
Screenshots of Calculator confirm the expression `347×29+156` displayed above in smaller text, and the result `10219` displayed below in larger text — matching the `primary: true` annotation on the larger element.

### Implementation approach
The fix uses **Option 3** from the improvement proposal — marking the primary display element. When multiple display elements exist, the one with the largest height (bounding box `b[3]`) gets `primary: true`. Single display elements are not marked (avoiding noise). This is a general-purpose heuristic that works beyond Calculator.

## Notes
- Edge case: if two display elements have equal height, only the first one found gets `primary`. This seems fine since equal-height displays typically don't have an expression/result distinction.
- The `primary` field uses `omitempty` so it only appears in output when true, keeping single-element output clean.
