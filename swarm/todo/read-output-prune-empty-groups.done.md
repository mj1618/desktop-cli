# Enhancement: Prune Empty Group Elements from `read` Output

## Priority: MEDIUM (Major token savings — typical Gmail read returns 80%+ useless group nodes)

## Problem

When running `desktop-cli read --app "Google Chrome" --text "Subject" --flat`, the output includes hundreds of intermediate `group` elements that have no title, value, or description:

```yaml
- i: 3
  r: group
  b: [77, 38, 981, 1079]
  a: [showmenu]
  p: window > group > group
- i: 4
  r: group
  b: [77, 38, 981, 1079]
  p: window > group > group > group
- i: 5
  r: group
  b: [77, 38, 981, 1079]
  p: window > group > group > group > group
# ... dozens more empty groups ...
- i: 3463
  r: input
  d: Subject
  b: [550, 792, 421, 20]
  p: window > group > group > group > group > group > group > group > web > group > group > group > group > group > group > input
```

In a typical Gmail `read --text` query, over 80% of the returned elements are anonymous `group` nodes that provide no useful information to the agent. These waste tokens, obscure the useful elements, and make output harder to parse.

## What to Build

### 1. Add `--prune` Flag to `read` Command

```bash
desktop-cli read --app "Google Chrome" --text "Subject" --flat --prune
```

When `--prune` is set, remove elements from the output that:
- Have role `group` or `other`
- AND have no title (`t`), value (`v`), or description (`d`)
- AND are not the direct parent of a matching/useful element (in tree mode)

This would reduce the Gmail Subject search from ~40 elements to ~5 useful ones.

### 2. Consider Making This the Default for `--text` Searches

When `--text` is specified, the user is looking for specific elements. The anonymous groups in the ancestry chain are almost never useful. Consider:
- Making `--prune` the default when `--text` is used
- Adding `--no-prune` to override and get the full tree

### 3. Alternative: `--compact` Output Mode

Instead of pruning, a `--compact` flag could:
- In flat mode: only return elements that have at least one of title/value/description
- In tree mode: collapse chains of anonymous groups into single path entries
- This is less aggressive than pruning and preserves structural context

### 4. Implementation Approach

After building the element tree and applying text/role filters:
1. Walk the output elements
2. Remove any element where `role ∈ {group, other}` AND `title == "" && value == "" && description == ""`
3. In tree mode: if a group's only purpose is as a container and it has one child, collapse it
4. In flat mode: simply omit the element from the list (the `p` path breadcrumb already captures the structure)

## Files to Modify

- `cmd/read.go` — Add `--prune` flag and filtering logic
- `README.md` — Document `--prune` flag
- `SKILL.md` — Add `--prune` to examples, especially for `--text` searches

## Acceptance Criteria

- [ ] `desktop-cli read --app "Chrome" --text "Subject" --flat --prune` returns only elements with meaningful text content
- [ ] Anonymous groups with no title/value/description are removed from output
- [ ] Elements with title, value, or description are preserved regardless of role
- [ ] The `p` (path) field in flat mode still shows the full ancestry for context
- [ ] `--prune` works with both tree and flat output modes
- [ ] Without `--prune`, output is unchanged (backward compatible)
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] README.md and SKILL.md updated

## Implementation Notes

- In flat mode, pruning is straightforward — just filter the list. In tree mode, it's more complex because removing a group may orphan its children. The tree pruner should "promote" children of removed groups up to the next surviving ancestor.
- The `--text` flag already limits output to matching subtrees, but it includes the full ancestry chain. Pruning complements this by removing the useless parts of that chain.
- Token savings estimate: In the Gmail compose test, a `read --text "Subject"` returned ~40 elements. With pruning, this would be ~5-8. That's a 5-8x reduction in output tokens for text searches.
