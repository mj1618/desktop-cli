# Feature: Tree Diff Responses — Return Only What Changed

## Priority: MEDIUM (reduces token usage for repeated reads)

## Problem

When agents monitor dynamic UIs, they repeatedly call `read` and process the full output each time. Most of the tree is identical between reads — a page with 200 elements might have 3 that changed. The agent's LLM processes all 200 elements every time, wasting tokens and context window.

Example: monitoring Gmail inbox for a new email:
```bash
# Read 1: 150 elements, 300 tokens
desktop-cli read --app "Chrome" --format agent
# Read 2: 150 elements, 300 tokens (1 new email appeared)
desktop-cli read --app "Chrome" --format agent
# Read 3: 150 elements, 300 tokens (nothing changed)
desktop-cli read --app "Chrome" --format agent
```

Total: 900 tokens consumed for 3 reads, but only ~10 tokens of actual new information.

The `observe` command already does diffing internally, but it's a streaming command. Agents need a simpler request/response diff mode for `read`.

## What to Build

### 1. `--since` Flag on `read` Command

Add a `--since <timestamp>` flag that returns only elements that changed since a previous read. The previous read's timestamp is included in every read response:

```bash
# Initial read — full output
desktop-cli read --app "Chrome" --format agent
# Output includes: ts: 1707504000

# Subsequent read — only changes
desktop-cli read --app "Chrome" --format agent --since 1707504000
```

### 2. Diff Output Format

When `--since` is specified, the response includes only the differences:

```yaml
ok: true
action: read
ts: 1707504003
diff:
    added:
        - i: 201
          r: row
          t: "New email from Alice"
          b: [100, 400, 800, 40]
        - i: 202
          r: txt
          v: "Hey, are you free for lunch?"
          b: [150, 410, 600, 20]
    removed:
        - i: 95
    changed:
        - i: 12
          field: v
          from: "Inbox (23288)"
          to: "Inbox (23289)"
        - i: 45
          field: t
          from: "3 minutes ago"
          to: "6 minutes ago"
    unchanged_count: 148
```

In agent format:
```
# Gmail - Google Chrome (pid: 44037) [diff since 1707504000]

+ [201] row "New email from Alice" (100,400,800,40)
+ [202] txt "Hey, are you free for lunch?" (150,410,600,20)
- [95]
~ [12] txt val="Inbox (23289)" (was "Inbox (23288)")
~ [45] txt "6 minutes ago" (was "3 minutes ago")
# 148 elements unchanged
```

### 3. Implementation

The server/process needs to retain the previous tree snapshot to compute diffs. Two approaches:

**Approach A: Client-side snapshot (stateless — preferred for CLI mode)**
- The `--since` flag is a timestamp
- The CLI re-reads the full tree but compares against a snapshot file stored at `/tmp/desktop-cli-snapshot-{app}-{ts}.json`
- Each `read` writes the current tree to the snapshot file
- On `--since`, load the snapshot, compute diff, return only changes
- Snapshots auto-expire after 60 seconds

**Approach B: Server-side snapshot (for MCP mode)**
- The MCP server retains the last tree per app in memory
- `read --since` compares against the in-memory previous tree
- No file I/O needed; more efficient

Implement Approach A for CLI, Approach B for MCP server.

### 4. Diff Algorithm

```go
func DiffTrees(oldElements, newElements []model.Element) TreeDiff {
    oldMap := buildIDMap(oldElements)  // Flatten and index by content hash
    newMap := buildIDMap(newElements)

    diff := TreeDiff{}
    for id, newElem := range newMap {
        oldElem, exists := oldMap[id]
        if !exists {
            diff.Added = append(diff.Added, newElem)
        } else if !elementsEqual(oldElem, newElem) {
            diff.Changed = append(diff.Changed, describeDiff(oldElem, newElem))
        }
        // else: unchanged, skip
    }
    for id := range oldMap {
        if _, exists := newMap[id]; !exists {
            diff.Removed = append(diff.Removed, id)
        }
    }
    diff.UnchangedCount = len(newMap) - len(diff.Added) - len(diff.Changed)
    return diff
}
```

Note: Element IDs are re-assigned each read, so diffing by ID won't work. Instead, match elements by a content hash: `hash(role + title + value + description + parentHash)`. This creates a stable identity based on the element's semantic content and position in the tree.

### 5. Usage Examples

```bash
# Monitor Gmail for new emails
ts=$(desktop-cli read --app "Chrome" --format agent | grep '^ts:' | cut -d' ' -f2)
sleep 10
desktop-cli read --app "Chrome" --format agent --since $ts

# Check what changed after an action
desktop-cli read --app "Safari" --format agent  # ts: 1707504000
desktop-cli click --text "Submit" --app "Safari"
sleep 1
desktop-cli read --app "Safari" --format agent --since 1707504000

# Repeated monitoring loop (shell)
while true; do
    desktop-cli read --app "Chrome" --format agent --since $ts
    ts=$(date +%s)
    sleep 5
done
```

## Files to Create

- `internal/model/diff.go` — Tree diffing algorithm
- `internal/model/diff_test.go` — Unit tests for diffing

## Files to Modify

- `cmd/read.go` — Add `--since` flag, snapshot storage, diff output
- `internal/output/output.go` — Add diff formatting for YAML/JSON/agent formats
- `internal/output/output_test.go` — Tests for diff output formatting
- `README.md` — Document `--since` flag
- `SKILL.md` — Add diff examples

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `read --since <ts>` returns only changed elements
- [ ] Diff output includes added, removed, changed, and unchanged_count
- [ ] Agent format diff uses `+`, `-`, `~` prefixes for clarity
- [ ] YAML format includes structured diff with field-level changes
- [ ] Content-hash matching correctly identifies same elements across reads
- [ ] Snapshot files are written to `/tmp/` and auto-expire
- [ ] `read` without `--since` still returns full output (backwards compatible)
- [ ] Diff works with `--roles`, `--prune`, `--scope-id` filters
- [ ] README.md and SKILL.md updated

## Implementation Notes

- **Content hashing for stable identity**: Since element IDs are re-assigned each read, use a hash of `role + title + value + description + parent_path` to create a stable identity. This correctly handles reordered elements (e.g., a new list item inserted in the middle).
- **Snapshot expiry**: Use the timestamp in the filename and delete snapshots older than 60 seconds on each read. This prevents `/tmp/` pollution.
- **Interaction with `--post-read`**: When `post-action-state-in-responses` is implemented, the post-read state could also be returned as a diff against the pre-action state. This would show exactly what the action changed.
- **Token savings**: For a 200-element UI with 3 changes, the diff is ~20 tokens vs ~400 tokens for the full read. Over a 10-read monitoring session, that's 380 tokens saved × 10 = 3800 fewer tokens processed by the LLM.
- **Performance**: Diffing two 200-element trees is O(n) with hash maps. The snapshot file I/O adds ~1ms. The bottleneck remains the accessibility tree read itself (~100-500ms).
- **Depends on**: No dependencies on other tasks, but pairs well with MCP server mode (Approach B is more efficient in server mode).
