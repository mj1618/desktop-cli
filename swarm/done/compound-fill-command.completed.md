# Feature: `fill` Command — Set Multiple Form Fields in One Call

## Priority: MEDIUM (saves 2-4 round-trips per form interaction)

## Problem

Filling a form is the most common multi-step agent workflow. Today it requires one command per field:

```bash
desktop-cli type --target "Full Name" --app "Safari" --text "John Doe"
desktop-cli type --target "Email" --app "Safari" --text "john@example.com"
desktop-cli type --target "Phone" --app "Safari" --text "+1-555-0100"
desktop-cli type --target "Message" --app "Safari" --text "Hello, I'd like to schedule a meeting."
```

That's 4 LLM round-trips (~12 seconds of inference), and each call re-reads the full accessibility tree to resolve the target element. The tree is traversed 4 times for what is conceptually a single operation: "fill this form."

## What to Build

### 1. Command Definition — `cmd/fill.go` (new file)

A `fill` command that sets multiple fields in one call:

```
desktop-cli fill [flags]

Flags:
  --app <name>          Target application (required)
  --window <title>      Target window
  --pid <pid>           Target by PID
  --field <label=value> Set a field (repeatable)
  --submit <text>       After filling, click element with this text (e.g. "Submit")
  --tab-between         Use Tab key to move between fields instead of direct targeting (default: false)
  --method <method>     How to set values: "type" (keystrokes) or "set-value" (direct, default)
```

### 2. Usage Examples

```bash
# Fill multiple fields by label
desktop-cli fill --app "Safari" \
  --field "Full Name=John Doe" \
  --field "Email=john@example.com" \
  --field "Phone=+1-555-0100" \
  --field "Message=Hello, I'd like to schedule a meeting."

# Fill and submit in one call
desktop-cli fill --app "Safari" \
  --field "Full Name=John Doe" \
  --field "Email=john@example.com" \
  --submit "Submit"

# Use Tab navigation between fields (for forms without clear labels)
desktop-cli fill --app "Safari" \
  --field "Full Name=John Doe" \
  --tab-between \
  --submit "Submit"

# Use keystroke typing instead of direct value setting
desktop-cli fill --app "Chrome" \
  --field "Search=coffee shops" \
  --method type

# Stdin YAML input for many fields
desktop-cli fill --app "Safari" <<'EOF'
fields:
  - label: "Full Name"
    value: "John Doe"
  - label: "Email"
    value: "john@example.com"
  - label: "Phone"
    value: "+1-555-0100"
  - label: "Message"
    value: "Hello, I'd like to schedule a meeting."
submit: "Submit"
EOF
```

### 3. Output Format

```yaml
ok: true
action: fill
fields_set: 4
results:
  - label: "Full Name"
    ok: true
    target: { i: 42, r: input, t: "Full Name", v: "John Doe" }
  - label: "Email"
    ok: true
    target: { i: 43, r: input, t: "Email", v: "john@example.com" }
  - label: "Phone"
    ok: true
    target: { i: 44, r: input, t: "Phone", v: "+1-555-0100" }
  - label: "Message"
    ok: true
    target: { i: 45, r: input, t: "Message", v: "Hello, I'd like to schedule a meeting." }
submitted:
    ok: true
    target: { i: 89, r: btn, t: "Submit" }
```

### 4. Implementation

```go
func runFill(cmd *cobra.Command, args []string) error {
    provider, err := platform.NewProvider()
    // ...

    // 1. Read the tree ONCE for all fields
    elements, err := provider.Reader.ReadElements(readOpts)

    // 2. For each --field "label=value":
    //    a. Find the element by label text (same logic as --target in type)
    //    b. Set the value using set-value (default) or type (if --method type)
    for _, field := range fields {
        label, value := parseField(field)
        elem := resolveElementByText(elements, label, ...)
        if method == "set-value" {
            provider.ValueSetter.SetValue(elem, value)
        } else {
            // Focus element, then type
            provider.ValueSetter.SetAttribute(elem, "focused", "true")
            provider.Inputter.TypeText(value)
        }
    }

    // 3. If --submit specified, click the submit button
    if submitText != "" {
        submitElem := resolveElementByText(elements, submitText, ...)
        provider.ActionPerformer.PerformAction(submitElem, "press")
    }

    // 4. Output results
}
```

Key optimization: **read the tree once**, resolve all elements from the same snapshot, then perform all actions. This avoids N tree reads for N fields.

### 5. Handling Field Resolution

Fields are matched by label text using the same logic as `--target` in the `type` command:
- Substring match on title/value/description
- Case-insensitive
- Leaf-node preference (deepest match)
- Interactive role preference

For ambiguous labels, the error message includes all candidates with IDs, so the agent can retry with `--field "id:42=John Doe"` (ID-based targeting).

### 6. Stdin Input for Many Fields

When there are many fields, `--field` flags become unwieldy. Support YAML input on stdin:

```yaml
fields:
  - label: "Full Name"
    value: "John Doe"
  - id: 42           # Target by ID instead of label
    value: "john@example.com"
  - label: "Phone"
    value: "+1-555-0100"
    method: type      # Override method per-field
submit: "Submit"
```

Stdin is used when `--field` flags are not provided and stdin is not a terminal.

## Files to Create

- `cmd/fill.go` — New `fill` command implementation

## Files to Modify

- `cmd/helpers.go` — May need shared field-resolution helpers
- `README.md` — Add `fill` command documentation
- `SKILL.md` — Add `fill` command to quick reference and agent workflow

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `desktop-cli fill --help` shows usage and flags
- [ ] `--field "Name=John"` sets a field by label
- [ ] Multiple `--field` flags set multiple fields
- [ ] Tree is read only ONCE for all fields (not per-field)
- [ ] `--submit "Submit"` clicks the submit element after filling
- [ ] `--method type` uses keystrokes instead of direct value setting
- [ ] `--tab-between` uses Tab key to navigate between fields
- [ ] YAML stdin input works when `--field` is not provided
- [ ] Per-field results are returned with target element info
- [ ] Errors on individual fields don't prevent other fields from being set
- [ ] ID-based targeting works: `--field "id:42=value"`
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Single tree read**: The biggest optimization over calling `type --target` N times. Reading the tree once and resolving all elements from the same snapshot is both faster and more consistent (elements won't shift between reads).
- **`set-value` as default method**: Direct value setting is instant and doesn't require focus management. Use `--method type` for apps that intercept keystroke events (e.g., auto-formatting phone numbers on each keystroke).
- **`--tab-between` mode**: For forms without clear per-field labels, the agent can click the first field, fill it, Tab to the next, fill it, etc. This mode focuses the first field, types the first value, presses Tab, types the second value, etc.
- **Depends on `do` command refactoring**: The `fill` command benefits from the same extracted internal functions. But it can also be implemented independently since it has a different optimization strategy (single tree read + batch resolution).
- **Future: `--verify` flag**: After filling, re-read the tree and verify each field's value matches what was set. Useful for apps that transform input (e.g., auto-formatting phone numbers). But this is a nice-to-have for a second pass.
