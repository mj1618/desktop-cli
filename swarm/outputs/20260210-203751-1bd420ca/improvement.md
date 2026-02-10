# Agent format outputs 725KB for Wikipedia article page

## Problem
When reading a Wikipedia article page with agent format (the default when piped), the output is extremely large - 725.9KB, which gets truncated and saved to a file instead of being returned inline.

Command executed:
```bash
desktop-cli read --app "Safari" --format agent
```

Expected behavior: Agent format should be "ultra-compact" and "20-30x fewer lines than YAML" according to SKILL.md. For a Wikipedia article, this should be manageable (under 100KB at most).

Actual behavior: Output is 725.9KB and gets saved to a file with message "Output too large (725.9KB)". This defeats the purpose of agent format being token-efficient for LLMs.

## Proposed Fix
Agent format should be more aggressive about filtering out non-interactive elements on complex web pages. Possible solutions:

1. **Auto-enable pruning for web pages in agent format** - The `--prune` flag already exists for web apps and reduces output 5-8x. It should be automatically enabled when using agent format on web content, not just when output is piped.

2. **Add a max-elements cap for agent format** - Limit agent format to the first N interactive elements (e.g. 500) with a flag like `--max-elements 500` or make it the default for agent format.

3. **Smarter depth limiting for agent format** - Agent format could auto-limit depth based on element count, stopping when it hits a reasonable threshold (e.g. stop at depth 3 if > 300 elements already).

4. **Add `--roles` auto-filtering in agent format** - When not explicitly specified, agent format could default to only showing highly interactive roles (btn, lnk, input, chk, toggle) and exclude less useful ones (txt, img, group).

5. **Add agent format output size warnings** - If output will exceed a threshold (e.g. 50KB), print a warning suggesting the user add `--roles`, `--depth`, or `--prune` flags before proceeding.

The goal: agent format should ALWAYS be compact enough to fit in an LLM context without truncation, even on complex web pages like Wikipedia articles.

## Reproduction
```bash
# Open Safari to a Wikipedia article (or navigate to one manually)
desktop-cli open --url "https://en.wikipedia.org/wiki/Artificial_intelligence" --app "Safari" --wait --timeout 10

# Wait for page load
sleep 3

# Read with agent format - will produce 725KB+ output
desktop-cli read --app "Safari" --format agent
```

This will output hundreds of KB instead of the expected compact format suitable for LLM consumption.
