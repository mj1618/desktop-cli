# Feature: Indexed Screenshot Read — Visual + Structured in One Response

## Priority: HIGH (single biggest accuracy improvement for agents)

## Problem

Agents have two ways to understand a UI:

1. **`read --format agent`** — structured, compact, but no spatial context. Agents must mentally reconstruct layout from `(x,y,w,h)` tuples, which they're terrible at. They can't see visual cues like colors, icons, separators, or content not in the accessibility tree.

2. **`screenshot`** / **`screenshot-coords`** — visual, rich context, but no structured IDs. The agent sees where things are, but can't directly act on them (`click --id` requires a `read` first).

This forces a common 2-call pattern:
```bash
desktop-cli screenshot-coords --app "Safari" --output /tmp/coords.png  # visual understanding
desktop-cli read --app "Safari" --format agent                          # structured IDs
```

Two calls, two LLM round-trips, and the agent must mentally correlate the screenshot with the element list. Worse, `screenshot-coords` currently labels elements with `(x,y)` coordinates — NOT element IDs — so the labels in the image don't correspond to anything the agent can use in commands.

## What to Build

### 1. `read --format screenshot` — Combined Visual + Structured Output

A new output format that returns BOTH an annotated screenshot AND a structured element list in a single response. The screenshot has numbered labels that ARE the element IDs, so the agent can look at the image and immediately say `click --id 42`.

```bash
desktop-cli read --app "Safari" --format screenshot
```

Output:
```yaml
ok: true
action: read
app: Safari
pid: 5678
image: "data:image/png;base64,iVBOR..."   # annotated screenshot with [ID] labels
elements:
    - i: 11
      r: btn
      t: "Back"
      b: [917,239,34,34]
    - i: 42
      r: input
      t: "Address"
      v: "https://example.com"
      b: [200,50,800,30]
    - i: 89
      r: btn
      t: "Submit"
      b: [200,400,100,32]
```

The image shows each element with a small `[11]`, `[42]`, `[89]` label overlay — the same IDs from the `elements` list. The agent sees the visual layout AND has IDs to act with.

### 2. Screenshot Annotation Changes

Modify `screenshot-coords` annotation logic to label elements with their IDs instead of `(x,y)` coordinates:

```
Current:  (500,300) on each element's bounding box
Proposed: [42] on each element's bounding box
```

The ID label is smaller, less cluttered, and directly actionable. The `(x,y)` coordinates are already in the structured `elements` list if needed.

### 3. Agent Format Variant with Inline Image

For multimodal agents (Claude, GPT-4V), the response interleaves the image and text naturally:

```
[image: annotated screenshot with ID labels]

# Safari — Contact Form (pid: 5678)
[11] btn "Back" (917,239,34,34)
[42] input "Address" val="https://example.com" (200,50,800,30)
[89] btn "Submit" (200,400,100,32)
```

The agent gets spatial understanding from the image and structured data from the text, all in one response.

### 4. Scaling and Token Efficiency

Screenshots are expensive in tokens. Smart defaults to minimize cost:

- **Default scale: 0.25** (quarter resolution) for `--format screenshot`. Enough to see layout and read labels. Override with `--scale 0.5` or `--scale 1.0` if more detail needed.
- **JPEG by default** for screenshot format (smaller than PNG for photos/UIs). Override with `--image-format png`.
- **Label only interactive elements** by default (same as `screenshot-coords`). Use `--all-elements` for everything.
- **Skip zero-bound elements** (off-screen/virtualized) — they add clutter without value.

### 5. Implementation

Reuse existing `screenshot-coords` annotation logic but change labels from `(x,y)` to `[id]`:

```go
func runRead(cmd *cobra.Command, args []string) error {
    format, _ := cmd.Flags().GetString("format")

    if format == "screenshot" {
        // 1. Read elements (same as normal read)
        elements, err := provider.Reader.ReadElements(readOpts)

        // 2. Capture screenshot
        imageData, err := provider.Screenshotter.CaptureWindow(screenshotOpts)

        // 3. Annotate with [ID] labels instead of (x,y)
        annotated := AnnotateScreenshotWithIDs(img, flatElements, windowBounds)

        // 4. Output both image + element list
        result := ScreenshotReadResult{
            Image:    base64Encode(annotated),
            Elements: formatAgentElements(elements),
        }
        return output.PrintYAML(result)
    }
    // ... existing format handling ...
}
```

### 6. Usage Examples

```bash
# Basic: visual + structured in one call
desktop-cli read --app "Safari" --format screenshot

# Higher resolution for complex UIs
desktop-cli read --app "Chrome" --format screenshot --scale 0.5

# Scoped to a dialog
desktop-cli read --app "Chrome" --format screenshot --scope-id 156

# Save image to file instead of base64 inline
desktop-cli read --app "Safari" --format screenshot --screenshot-output /tmp/safari.png
```

## Files to Modify

- `cmd/read.go` — Add `screenshot` format handling
- `cmd/screenshot_coords.go` — Refactor annotation to support ID labels (not just coordinates)
- `cmd/screenshot_coords_draw.go` — Update drawing logic for `[id]` labels
- `internal/output/output.go` — Add screenshot read result type
- `README.md` — Document `--format screenshot`
- `SKILL.md` — Add to agent workflow as recommended first step

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `read --format screenshot --app "Finder"` returns YAML with both `image` and `elements` fields
- [ ] Image contains `[id]` labels (not `(x,y)` coordinates) on each element
- [ ] Element IDs in image match IDs in the `elements` list
- [ ] Default scale is 0.25 for screenshot format (token efficient)
- [ ] `--scale` flag overrides default
- [ ] `--scope-id` works with screenshot format (annotates only scoped elements)
- [ ] `--screenshot-output /tmp/file.png` saves image to file instead of inline base64
- [ ] Interactive elements labeled by default, `--all-elements` labels everything
- [ ] Zero-bound elements are excluded from annotations
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Why IDs not coordinates**: `(x,y)` labels are useless — agents can't click coordinates without the `click` command, and `click --id` is faster and more reliable than `click --x --y`. IDs are the universal currency for acting on elements.
- **Token cost**: A 0.25 scale JPEG screenshot of a typical window is ~5K-15K tokens. The structured element list adds ~200-500 tokens. Total ~6K-16K tokens per call. This replaces TWO calls (screenshot + read) that would cost ~10K-20K tokens total. Net: similar token cost, half the round-trips, dramatically better comprehension.
- **Multimodal compatibility**: Claude, GPT-4V, and Gemini all support inline images in tool responses. The base64 image will render natively in the agent's context.
- **Refactoring `screenshot-coords`**: The existing `AnnotateScreenshot` function draws red boxes and `(x,y)` labels. Add a parameter to switch to `[id]` labels. Both `screenshot-coords` and `read --format screenshot` call the same underlying function.
- **Accessibility tree gaps become visible**: When elements are missing from the tree (Apple Notes checkboxes, Google Sheets cells), the screenshot shows them even though they're not in the element list. The agent can fall back to coordinate clicking for visible-but-unlabeled elements.
