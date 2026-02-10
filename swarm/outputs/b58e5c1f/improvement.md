# `--format agent` documented in SKILL.md but not implemented

## Problem
SKILL.md documents a `--format agent` option for the `read` command with detailed examples:

```
desktop-cli read --app "Chrome" --format agent   # agent format: only clickable elements, one per line
```

The docs describe it as: "Ultra-compact output for AI agents — shows only interactive elements, one per line: `[id] role "label" (x,y,w,h) [flags]`. Typically 20-30x fewer lines than YAML."

However, running this command produces an error:

```
$ desktop-cli read --app "Notes" --format agent
Error: unsupported format: agent (use yaml or json)
```

The `--help` output confirms only `yaml` and `json` are valid format values. The entire "Agent format" section of SKILL.md and multiple `--format agent` examples throughout are misleading — an agent following the documented workflow will fail on its first `read` call.

## Proposed Fix
Either:
1. **Implement `--format agent`** — Add a new output format to the `read` command that outputs one line per interactive element: `[id] role "label" (x,y,w,h) [flags]`. This would filter to actionable roles (btn, lnk, input, chk, radio, menuitem, tab, etc.) and output a flat, token-efficient format. This is the better option since the format would be genuinely useful — YAML trees are verbose and expensive in tokens for AI agents.
2. **Remove from SKILL.md** — If the feature is not yet ready, remove all `--format agent` references from SKILL.md to avoid confusing agents. This includes the format description paragraph, all example commands using `--format agent`, and the "Agent format" callout.

## Reproduction
```bash
# 1. Read SKILL.md — note the documented --format agent examples
# 2. Try to use it:
desktop-cli read --app "Notes" --format agent
# Output: Error: unsupported format: agent (use yaml or json)

# 3. Verify with --help:
desktop-cli read --help
# Shows: --format string   Output format: yaml, json (default "yaml")
```
