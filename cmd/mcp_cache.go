package cmd

import (
	"sync"
	"time"

	"github.com/mj1618/desktop-cli/internal/model"
	"github.com/mj1618/desktop-cli/internal/platform"
)

// mcpCacheKey identifies a unique tree read scope.
type mcpCacheKey struct {
	App      string
	Window   string
	WindowID int
	PID      int
}

// mcpCacheEntry holds a cached element tree with its timestamp.
type mcpCacheEntry struct {
	elements  []model.Element
	timestamp time.Time
}

// mcpTreeCache provides a TTL-based cache for accessibility element trees.
type mcpTreeCache struct {
	mu      sync.Mutex
	entries map[mcpCacheKey]mcpCacheEntry
	ttl     time.Duration
}

// newMCPTreeCache creates a new cache. A ttl of 0 disables caching.
func newMCPTreeCache(ttl time.Duration) *mcpTreeCache {
	return &mcpTreeCache{
		entries: make(map[mcpCacheKey]mcpCacheEntry),
		ttl:     ttl,
	}
}

// readElements returns cached elements if within TTL, otherwise reads fresh.
func (c *mcpTreeCache) readElements(reader platform.Reader, opts platform.ReadOptions) ([]model.Element, error) {
	if c.ttl == 0 {
		return reader.ReadElements(opts)
	}

	key := mcpCacheKey{
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
	c.entries[key] = mcpCacheEntry{elements: elements, timestamp: time.Now()}
	c.mu.Unlock()

	return elements, nil
}

// invalidateApp removes all cache entries for the given app.
func (c *mcpTreeCache) invalidateApp(app string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if k.App == app {
			delete(c.entries, k)
		}
	}
}

// invalidateAll clears the entire cache.
func (c *mcpTreeCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[mcpCacheKey]mcpCacheEntry)
}
