package model

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HashChange represents a changed element detected by hash-based diffing.
type HashChange struct {
	ID      int                  `yaml:"i"                json:"i"`
	Role    string               `yaml:"r,omitempty"      json:"r,omitempty"`
	Title   string               `yaml:"t,omitempty"      json:"t,omitempty"`
	Changes map[string][2]string `yaml:"changes"          json:"changes"`
}

// TreeDiff is the result of comparing two element snapshots by content hash.
type TreeDiff struct {
	Added          []FlatElement `yaml:"added,omitempty"   json:"added,omitempty"`
	Removed        []FlatElement `yaml:"removed,omitempty" json:"removed,omitempty"`
	Changed        []HashChange  `yaml:"changed,omitempty" json:"changed,omitempty"`
	UnchangedCount int           `yaml:"unchanged_count"   json:"unchanged_count"`
}

// ElementHash computes a stable identity hash for an element based on its
// semantic content and position in the tree. This allows matching elements
// across separate reads where sequential IDs may shift.
func ElementHash(el FlatElement) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s", el.Role, el.Title, el.Description, el.Subrole, el.Path)
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// DiffElementsByHash compares two flat element lists using content hashing
// for stable identity. Unlike DiffElements (which matches by sequential ID),
// this handles ID shifts caused by elements being added or removed.
func DiffElementsByHash(prev, curr []FlatElement) TreeDiff {
	// Build maps keyed by content hash
	prevByHash := make(map[string]FlatElement, len(prev))
	for _, el := range prev {
		h := ElementHash(el)
		prevByHash[h] = el
	}
	currByHash := make(map[string]FlatElement, len(curr))
	for _, el := range curr {
		h := ElementHash(el)
		currByHash[h] = el
	}

	var diff TreeDiff

	// Check for added and changed elements
	for _, el := range curr {
		h := ElementHash(el)
		prevEl, existed := prevByHash[h]
		if !existed {
			diff.Added = append(diff.Added, el)
			continue
		}
		changes := diffSnapshotProperties(prevEl, el)
		if len(changes) > 0 {
			diff.Changed = append(diff.Changed, HashChange{
				ID:      el.ID,
				Role:    el.Role,
				Title:   el.Title,
				Changes: changes,
			})
		} else {
			diff.UnchangedCount++
		}
	}

	// Check for removed elements
	for _, el := range prev {
		h := ElementHash(el)
		if _, exists := currByHash[h]; !exists {
			diff.Removed = append(diff.Removed, el)
		}
	}

	return diff
}

// diffSnapshotProperties compares mutable properties between two elements
// that were matched by content hash. Title, role, description, and path are
// part of the hash so they won't differ. We check value, bounds, focused,
// selected, and enabled.
func diffSnapshotProperties(prev, curr FlatElement) map[string][2]string {
	diffs := make(map[string][2]string)

	if prev.Value != curr.Value {
		diffs["v"] = [2]string{prev.Value, curr.Value}
	}
	if prev.Bounds != curr.Bounds {
		diffs["b"] = [2]string{
			fmt.Sprintf("%v", prev.Bounds),
			fmt.Sprintf("%v", curr.Bounds),
		}
	}
	if prev.Focused != curr.Focused {
		diffs["f"] = [2]string{
			fmt.Sprintf("%v", prev.Focused),
			fmt.Sprintf("%v", curr.Focused),
		}
	}
	if prev.Selected != curr.Selected {
		diffs["s"] = [2]string{
			fmt.Sprintf("%v", prev.Selected),
			fmt.Sprintf("%v", curr.Selected),
		}
	}

	if len(diffs) == 0 {
		return nil
	}
	return diffs
}

// snapshotDir is the directory for snapshot files.
const snapshotDir = "/tmp"

// snapshotPrefix is the filename prefix for snapshot files.
const snapshotPrefix = "desktop-cli-snapshot-"

func snapshotPath(app string, ts int64) string {
	safe := strings.ReplaceAll(app, "/", "_")
	safe = strings.ReplaceAll(safe, " ", "_")
	return filepath.Join(snapshotDir, fmt.Sprintf("%s%s-%d.json", snapshotPrefix, safe, ts))
}

// SaveSnapshot writes a flat element list to a snapshot file for later diffing.
func SaveSnapshot(app string, ts int64, elements []FlatElement) error {
	data, err := json.Marshal(elements)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	return os.WriteFile(snapshotPath(app, ts), data, 0644)
}

// LoadSnapshot reads a previously saved snapshot from disk.
func LoadSnapshot(app string, ts int64) ([]FlatElement, error) {
	data, err := os.ReadFile(snapshotPath(app, ts))
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}
	var elements []FlatElement
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	return elements, nil
}

// CleanSnapshots removes snapshot files for the given app that are older than maxAge.
func CleanSnapshots(app string, maxAge time.Duration) {
	safe := strings.ReplaceAll(app, "/", "_")
	safe = strings.ReplaceAll(safe, " ", "_")
	prefix := snapshotPrefix + safe + "-"

	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(snapshotDir, entry.Name()))
		}
	}
}
