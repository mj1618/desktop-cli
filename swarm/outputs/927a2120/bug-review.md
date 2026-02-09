# Bug Review — Iteration 2

## Bugs Found and Fixed

### 1. Incorrect JSON-to-YAML documentation changes (README.md, SKILL.md, swarm/PLAN.md)

All three documentation files had "JSON" replaced with "yaml" throughout, but the actual codebase still uses JSON:
- `internal/model/element.go` uses `json:"..."` struct tags
- `internal/model/window.go` uses `json:"..."` struct tags
- `internal/output/json.go` uses `encoding/json` and `PrintJSON`

The documentation examples still showed JSON syntax (`{}`, `[]`, `":"`) despite being relabeled as "yaml". Additionally, "JSONL" was incorrectly changed to "yamlL" in PLAN.md.

**Fix**: Reverted all three files to match the actual JSON-based implementation.

### 2. swarm/swarm.yaml — no bugs

The addition of the context window size note is a legitimate, correct change. No action needed.

## Summary

Reverted `README.md`, `SKILL.md`, and `swarm/PLAN.md` to fix documentation that was incorrectly changed from JSON to YAML terminology while the codebase still uses JSON.
