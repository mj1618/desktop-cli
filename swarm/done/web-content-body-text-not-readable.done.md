# Bug/Limitation: Web Content Editable Body Text Not Readable in Accessibility Tree

## Priority: LOW (Chrome-specific limitation — workarounds exist)

## Problem

When composing an email in Gmail via Google Chrome, the email body (a `contenteditable` div) does not expose its text content through the macOS accessibility tree. After typing "Test" into the body field:

```bash
desktop-cli type --text "Test"  # typed into Gmail compose body
```

A subsequent `read` of the compose area shows no element containing the body text "Test". The compose body appears as anonymous `group` elements with no `v` (value) or `t` (title) field.

This means agents cannot verify:
- Whether text was actually typed into the body field
- What the current body content is
- Whether the body is empty or pre-filled

This contrasts with the Subject field (`input` role with `d: Subject`), which is readable, and the To field, which shows recipient chips as text elements.

## Steps to Reproduce

1. Open Gmail in Google Chrome
2. Click Compose to open a new email
3. Click in the body area and type some text
4. Run: `desktop-cli read --app "Google Chrome" --text "your typed text" --flat`
5. The typed text does not appear in any element's title, value, or description

## Expected Behavior

The body text should be exposed through the accessibility tree, either as:
- A `value` on the contenteditable element
- A `txt` child element containing the text content
- An `input` element with the value set

## Actual Behavior

The compose body area is represented as nested `group` elements with no text content. Only structural elements (the container groups) are visible.

## Impact

- Agents cannot verify body content was typed correctly
- Agents cannot read existing email body content (e.g., in reply drafts)
- No way to programmatically check the body before sending

## Likely Cause

This is likely a Chrome accessibility limitation with `contenteditable` divs. Chrome may not expose the inner text of rich-text editing areas through the AX API in the same way it does for standard `<input>` and `<textarea>` elements.

## Potential Workarounds

- Use `screenshot` + vision model to verify body content visually
- Trust that `type` succeeded (fragile but works for simple cases)
- Use `set-value` on the body element if it accepts value setting
- Use clipboard: type text, then Cmd+A, Cmd+C to copy body, then read clipboard

## Investigation Needed

- Does Safari expose contenteditable text in its accessibility tree?
- Does Chrome with `--force-renderer-accessibility` improve this?
- Does the `AXValue` attribute exist on the contenteditable element but just isn't being read?
- Is this related to the broader "chrome-web-content-not-in-accessibility-tree" bug, or a separate issue?

## Files to Investigate

- `internal/platform/darwin_reader.go` — Check if `AXValue` is being read for all element types
- `internal/platform/darwin_types.go` — Check if contenteditable elements have a different role mapping

## Acceptance Criteria

- [ ] Investigate whether the AX API exposes contenteditable text at all
- [ ] If yes, update the reader to capture it
- [ ] If no, document the limitation and recommend workarounds in SKILL.md
