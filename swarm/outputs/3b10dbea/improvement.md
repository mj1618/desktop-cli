# Support typing numeric expressions into Calculator

## Problem

When performing calculations in Calculator.app, agents must click individual digit and operator buttons one at a time. For the calculation `347 * 29 + 156`, this required 9 separate `action` commands:

```bash
desktop-cli action --text "3" --app "Calculator"
desktop-cli action --text "4" --app "Calculator"
desktop-cli action --text "7" --app "Calculator"
desktop-cli action --text "Multiply" --app "Calculator"
desktop-cli action --text "2" --app "Calculator"
desktop-cli action --text "9" --app "Calculator"
desktop-cli action --text "Add" --app "Calculator"
desktop-cli action --text "1" --app "Calculator"
desktop-cli action --text "5" --app "Calculator"
desktop-cli action --text "6" --app "Calculator"
desktop-cli action --text "Equals" --app "Calculator"
```

This is inefficient for agents:
- Many round-trips (11 commands for a simple calculation)
- High token usage (each response includes full display element info)
- Requires knowing exact button names ("Multiply", "Add", "Equals")
- Slow to execute

## Proposed Fix

Allow agents to type numeric expressions directly into Calculator:

```bash
# Type the expression and press enter
desktop-cli type --app "Calculator" --text "347*29+156" --key "enter"

# Or just type the expression with trailing equals
desktop-cli type --app "Calculator" --text "347*29+156="
```

The tool should:
1. Detect when the target app is Calculator
2. Type each character (digits, operators) directly as keyboard input
3. Support basic operators: `*`, `+`, `-`, `/`, `%`
4. Support pressing `=` or `enter` to evaluate
5. Return the display element(s) in the response (as it does now for `action`)

This would reduce the example above from 11 commands to 1 command.

## Reproduction

1. Open Calculator.app
2. Try to perform any multi-step calculation (e.g., `347 * 29 + 156`)
3. Observe that you must use 1 command per digit/operator (11 total for this example)
4. Compare to typing the expression directly: `type --app "Calculator" --text "347*29+156=" ` (1 command)
