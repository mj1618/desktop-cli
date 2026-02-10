# Coder Agent Summary

## Status
No pending or processing tasks were available. Proposed a new feature.

## Action Taken
Wrote a feature plan for an `assert` command and `do` batch step to `swarm/todo/assert-command.pending.md`.

## Feature: `assert` Command

**Problem**: Agents must do a full `read` + parse + compare loop every time they need to verify UI state after an action. This costs ~500-2000 extra tokens and a full LLM round-trip per validation.

**Solution**: An `assert` command that atomically checks element existence and properties (value, checked, disabled, focused, gone) with structured pass/fail output and exit codes. Also adds `assert` as a step type in the `do` batch command for inline validation.

**Key examples**:
```bash
desktop-cli assert --app "Safari" --text "Success"
desktop-cli assert --app "Safari" --id 42 --value "hello"
desktop-cli assert --app "Safari" --text "Loading..." --gone --timeout 5
```

**In batch**:
```yaml
- click: { text: "Submit" }
- assert: { text: "Success", timeout: 5 }
```

**Value**: Eliminates the most common agent automation bottleneck (state verification) with zero new platform dependencies.
