# Feature: `assert` Command and Batch Step

## Problem

Agents frequently need to verify UI state after performing actions (e.g., confirm a button click resulted in "Success" text appearing, verify a checkbox is checked, confirm an input has the expected value). Currently this requires a multi-step workflow:

1. `read --app "Safari" --text "Success"` — read the full tree or search for text
2. Parse the YAML/agent format output in the agent's own logic
3. Compare extracted values against expected state
4. Handle the "not found" case differently from "found but wrong value"

This is token-expensive, slow (extra round-trip), and error-prone. The `wait` command partially addresses this for element presence, but it can't assert property values (e.g., "is checked", "has value X", "is disabled") and doesn't provide structured pass/fail output.

## Proposed Solution

### New `assert` Command

```bash
# Assert element with text "Success" exists
desktop-cli assert --app "Safari" --text "Success"

# Assert element exists with specific role
desktop-cli assert --app "Safari" --text "Submit" --roles "btn"

# Assert element by ID has a specific value
desktop-cli assert --app "Safari" --id 42 --value "hello world"

# Assert element is checked
desktop-cli assert --app "Safari" --text "Remember me" --checked

# Assert element is NOT checked
desktop-cli assert --app "Safari" --text "Remember me" --unchecked

# Assert element is disabled
desktop-cli assert --app "Safari" --text "Submit" --disabled

# Assert element is enabled
desktop-cli assert --app "Safari" --text "Submit" --enabled

# Assert element is focused
desktop-cli assert --app "Safari" --text "Search" --focused

# Assert element does NOT exist (gone)
desktop-cli assert --app "Safari" --text "Loading..." --gone

# Assert with timeout (poll until condition met or fail)
desktop-cli assert --app "Safari" --text "Success" --timeout 5

# Assert by element ID exists
desktop-cli assert --app "Safari" --id 42

# Assert element value contains substring
desktop-cli assert --app "Safari" --id 42 --value-contains "expected"
```

### Response Format

**Pass:**
```yaml
ok: true
action: assert
pass: true
element:
    i: 42
    r: btn
    t: Submit
    b: [200, 400, 100, 32]
```

**Fail (element not found):**
```yaml
ok: false
action: assert
pass: false
error: "element not found: text=\"Success\" roles=\"btn\""
```

**Fail (property mismatch):**
```yaml
ok: false
action: assert
pass: false
error: "expected value \"hello\" but got \"world\""
element:
    i: 42
    r: input
    t: Search
    v: world
    b: [200, 400, 100, 32]
```

Exit code: 0 on pass, 1 on fail. This lets agents use shell-level `&&` chaining.

### Batch Step in `do`

```yaml
# In a do batch:
- click: { text: "Submit" }
- assert: { text: "Success", timeout: 5 }
- assert: { text: "Loading...", gone: true }
- click: { text: "Continue" }
```

This eliminates agent round-trips for validation — the most common agent automation pattern.

## Implementation Plan

### 1. Create `cmd/assert.go`
- New cobra command `assertCmd` following the pattern of `wait.go`
- Flags: `--text`, `--roles`, `--id`, `--app`, `--window`, `--pid`, `--window-id`, `--exact`, `--scope-id`
- Property assertion flags: `--value`, `--value-contains`, `--checked`, `--unchecked`, `--disabled`, `--enabled`, `--focused`, `--gone`
- Timing flag: `--timeout` (seconds, default 0 = single check, no polling)
- Reuse existing `findElement()` / `resolveApp()` / `resolveWindow()` helpers from `helpers.go`

### 2. Update `cmd/do.go`
- Add `"assert"` to the supported step types list
- Add `executeAssert()` function
- Map assert YAML keys to command flags

### 3. Update `README.md` and `SKILL.md`
- Document assert command with examples
- Add to agent workflow section (step 4 alternative to wait)

### Files to Create/Modify
- **Create**: `cmd/assert.go`
- **Modify**: `cmd/do.go` (add assert step type)
- **Modify**: `README.md` (add assert section)
- **Modify**: `SKILL.md` (add assert reference)

## Value

- **Token savings**: Eliminates a full `read` + parse round-trip for every validation (saves ~500-2000 tokens per check)
- **Reduced latency**: One CLI call instead of read → agent parse → decide, saving one LLM round-trip
- **Batch integration**: Most powerful in `do` pipelines where agents chain multiple actions and need inline validation
- **Clean semantics**: Pass/fail with structured error messages, unlike read which returns raw data requiring interpretation
- **Exit codes**: Enables shell-level `&&` chaining for simple scripts

## Dependencies
- None — all platform capabilities (element reading, filtering, property access) already exist
- Reuses existing helpers: `findElement`, `resolveApp`, `resolveWindow`, `formatElement`
