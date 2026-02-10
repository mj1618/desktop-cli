# Bug: `do --stop-on-error=false` incorrectly reports success when steps fail

## Problem

When running `desktop-cli do --stop-on-error=false` and one or more steps fail, the output reports:
- `ok: true` (should be `false`)
- `completed: N` counts all steps including failed ones (should only count successful ones)

## Steps to reproduce

```bash
echo '- sleep: { ms: 50 }
- unknown-cmd: {}
- sleep: { ms: 50 }' | ./desktop-cli do --stop-on-error=false
```

### Actual output

```yaml
ok: true
action: do
steps: 3
completed: 3
results:
    - step: 1
      ok: true
      action: sleep
      elapsed: 50ms
    - step: 2
      ok: false
      action: unknown-cmd
      error: 'unknown step type "unknown-cmd" ...'
    - step: 3
      ok: true
      action: sleep
      elapsed: 50ms
```

### Expected output

```yaml
ok: false
action: do
steps: 3
completed: 2
results: ...
```

## Root cause

In `cmd/do.go`, the `lastErr` variable is only set inside the `if stopOnError` branch (line ~130). When `stopOnError=false`, `lastErr` remains empty string. Then at line ~142-144:

```go
if lastErr == "" {
    completed = len(results)
}
```

This overwrites the correct `completed` count (which was incremented only for successes) with the total result count. And `allOK` at line ~152 is `lastErr == ""`, which is always true when `stopOnError=false`.

## Suggested fix

Track whether any step failed independently of `lastErr`:

```go
hasFailure := false
// ... in the error handling for stopOnError=false:
hasFailure = true

// After the loop:
allOK := !hasFailure && lastErr == ""
// Don't overwrite completed when stopOnError=false
```
