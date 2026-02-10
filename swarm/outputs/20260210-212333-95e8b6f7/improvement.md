# Excessive output from `--post-read` requires file storage for large responses

## Problem

When using the `--post-read` flag with `click` (or other action commands), the returned UI state can easily exceed 30KB, causing the tool to save output to a temporary file instead of displaying it inline. This breaks the agent workflow because:

1. The agent expects to see the UI state directly in the command output
2. Instead, they get a truncated preview and a file path
3. They must make an additional `Read` tool call to see the full state
4. This adds latency and complexity to the workflow

**Example command:**
```bash
desktop-cli click --id 171 --app "Google Chrome" --post-read --post-read-delay 1000
```

**Actual output:**
```
<persisted-output>
Output too large (30.5KB). Full output saved to: /Users/matt/.claude/projects/...

Preview (first 2KB):
ok: true
action: click
...
```

**Expected behavior:** The agent expects to see the full UI state directly, as the `--post-read` flag is explicitly meant to eliminate a follow-up `read` call.

## Proposed Fix

Add a `--post-read-format agent` flag (or auto-apply agent format when `--post-read` is used with piped output) to return the compact agent format instead of full YAML. This would:

1. Reduce output from 30KB+ to ~2-3KB (10-15x reduction based on existing agent format behavior)
2. Keep responses inline and parseable without file storage
3. Maintain the benefit of eliminating follow-up `read` calls
4. Provide the same structured, one-element-per-line output that agents already use

**Implementation options:**
- Add `--post-read-format agent|yaml|screenshot` flag to control post-read output format
- Auto-apply agent format for `--post-read` when stdout is piped (matching existing smart defaults behavior)
- Default to agent format for all `--post-read` operations (breaking change but more useful)

**Alternative workaround:** Add `--post-read-max-elements 50` to cap the number of elements returned, but this risks missing important UI elements. The agent format is better because it's comprehensive while staying compact.

## Reproduction

1. Open Google Chrome to any complex web page (e.g. Google Maps)
2. Run: `desktop-cli click --id <any-id> --app "Google Chrome" --post-read`
3. Observe output is saved to file instead of printed inline
4. Note that reading the file requires an additional tool call, defeating the purpose of `--post-read`

**Why this matters:** The `--post-read` flag's entire purpose is to eliminate a follow-up `read` call. When the output is too large to display inline, agents must make an additional `Read` call anyway, which adds round-trip latency and negates the benefit of `--post-read`.
