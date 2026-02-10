package server

import (
	"sync"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/platform"
)

// cacheKey identifies a unique tree read scope.
type cacheKey struct {
	App      string
	Window   string
	WindowID int
	PID      int
}

// cacheEntry holds a cached element tree with its timestamp.
type cacheEntry struct {
	elements  []model.Element
	timestamp time.Time
}

// TreeCache provides a TTL-based cache for accessibility element trees.
type TreeCache struct {
	mu      sync.Mutex
	entries map[cacheKey]cacheEntry
	ttl     time.Duration
}

// NewTreeCache creates a new cache. A ttl of 0 disables caching.
func NewTreeCache(ttl time.Duration) *TreeCache {
	return &TreeCache{
		entries: make(map[cacheKey]cacheEntry),
		ttl:     ttl,
	}
}

// ReadElements returns cached elements if within TTL, otherwise reads fresh.
// The caller must hold the provider mutex.
func (c *TreeCache) ReadElements(reader platform.Reader, opts platform.ReadOptions) ([]model.Element, error) {
	if c.ttl == 0 {
		return reader.ReadElements(opts)
	}

	key := cacheKey{
		App:      opts.App,
		Window:   opts.Window,
		WindowID: opts.WindowID,
		PID:      opts.PID,
	}

	c.mu.Lock()
	if entry, ok := c.entries[key]; ok && time.Since(entry.timestamp) < c.ttl {
		elements := entry.elements
		c.mu.Unlock()
		return elements, nil
	}
	c.mu.Unlock()

	elements, err := reader.ReadElements(opts)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[key] = cacheEntry{elements: elements, timestamp: time.Now()}
	c.mu.Unlock()

	return elements, nil
}

// InvalidateApp removes all cache entries for the given app.
func (c *TreeCache) InvalidateApp(app string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if k.App == app {
			delete(c.entries, k)
		}
	}
}

// InvalidateAll clears the entire cache.
func (c *TreeCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[cacheKey]cacheEntry)
}
