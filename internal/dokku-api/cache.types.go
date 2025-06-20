package dokkuApi

import (
	"sync"
	"time"
)

// CacheConfig defines caching behavior
type CacheConfig struct {
	Enabled    bool                     `yaml:"enabled"`
	DefaultTTL time.Duration            `yaml:"default_ttl"`
	Policies   map[string]time.Duration `yaml:"policies,omitempty"`
}

// DefaultCacheConfig returns sensible caching defaults
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enabled:    true,
		DefaultTTL: 5 * time.Minute,
		Policies: map[string]time.Duration{
			// Fast-changing data - short cache
			"logs":        30 * time.Second,
			"ps:scale":    1 * time.Minute,
			"apps:exists": 2 * time.Minute,

			// Semi-stable data - medium cache
			"config:show":    5 * time.Minute,
			"domains:report": 5 * time.Minute,

			// Stable data - longer cache
			"plugin:list":   15 * time.Minute,
			"version":       30 * time.Minute,
			"ssh-keys:list": 10 * time.Minute,
		},
	}
}

// GetTTLForCommand returns the appropriate TTL for a command
func (c *CacheConfig) GetTTLForCommand(command string) time.Duration {
	if ttl, exists := c.Policies[command]; exists {
		return ttl
	}
	return c.DefaultTTL
}

// cacheEntry stores cached command results with TTL (internal to cache manager)
type cacheEntry struct {
	result    []byte
	error     error
	expiresAt time.Time
}

// commandCache stores cached command results (internal to cache manager)
type commandCache struct {
	entries map[string]*cacheEntry
	mutex   sync.RWMutex
}
