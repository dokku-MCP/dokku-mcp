package dokkuApi

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"time"
)

// CommandCacheManager handles command result caching with TTL and cleanup
type CommandCacheManager struct {
	config  *CacheConfig
	cache   *commandCache
	logger  *slog.Logger
	cleanup *time.Ticker
}

// NewCommandCacheManager creates a new cache manager with the given configuration
func NewCommandCacheManager(config *CacheConfig, logger *slog.Logger) *CommandCacheManager {
	if config == nil || !config.Enabled {
		return nil
	}

	manager := &CommandCacheManager{
		config: config,
		cache: &commandCache{
			entries: make(map[string]*cacheEntry),
		},
		logger: logger,
	}

	// Start background cleanup
	manager.startCleanup()

	logger.Debug("Command cache manager initialized",
		"default_ttl", config.DefaultTTL,
		"policies", len(config.Policies))

	return manager
}

// Get retrieves a cached result if available and not expired
func (cm *CommandCacheManager) Get(command string, args []string) ([]byte, error, bool) {
	if cm == nil {
		return nil, nil, false
	}

	key := cm.generateCacheKey(command, args)

	cm.cache.mutex.RLock()
	defer cm.cache.mutex.RUnlock()

	entry, exists := cm.cache.entries[key]
	if !exists {
		return nil, nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return nil, nil, false
	}

	cm.logger.Debug("Cache hit",
		"command", command,
		"args", args,
		"key", key)

	return entry.result, entry.error, true
}

// Set stores a command result in the cache with appropriate TTL
func (cm *CommandCacheManager) Set(command string, args []string, result []byte, err error) {
	if cm == nil {
		return
	}

	key := cm.generateCacheKey(command, args)
	ttl := cm.config.GetTTLForCommand(command)

	cm.cache.mutex.Lock()
	defer cm.cache.mutex.Unlock()

	cm.cache.entries[key] = &cacheEntry{
		result:    result,
		error:     err,
		expiresAt: time.Now().Add(ttl),
	}

	cm.logger.Debug("Cached command result",
		"command", command,
		"key", key,
		"ttl", ttl)
}

// Invalidate clears all cached entries
func (cm *CommandCacheManager) Invalidate() {
	if cm == nil {
		return
	}

	cm.cache.mutex.Lock()
	defer cm.cache.mutex.Unlock()

	cm.cache.entries = make(map[string]*cacheEntry)
	cm.logger.Debug("Cache invalidated")
}

// Stop stops the background cleanup process
func (cm *CommandCacheManager) Stop() {
	if cm != nil && cm.cleanup != nil {
		cm.cleanup.Stop()
	}
}

// Internal methods

// generateCacheKey creates a unique key for command + args combination
func (cm *CommandCacheManager) generateCacheKey(command string, args []string) string {
	hasher := sha256.New()
	hasher.Write([]byte(command))
	for _, arg := range args {
		hasher.Write([]byte(arg))
	}
	return hex.EncodeToString(hasher.Sum(nil))[:16] // First 16 chars
}

// startCleanup starts a background goroutine to clean expired entries
func (cm *CommandCacheManager) startCleanup() {
	cm.cleanup = time.NewTicker(cm.config.DefaultTTL / 2)
	go func() {
		defer cm.cleanup.Stop()
		for range cm.cleanup.C {
			cm.cleanupExpired()
		}
	}()
}

// cleanupExpired removes expired cache entries
func (cm *CommandCacheManager) cleanupExpired() {
	cm.cache.mutex.Lock()
	defer cm.cache.mutex.Unlock()

	now := time.Now()
	cleaned := 0
	for key, entry := range cm.cache.entries {
		if now.After(entry.expiresAt) {
			delete(cm.cache.entries, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		cm.logger.Debug("Cleaned expired cache entries", "count", cleaned)
	}
}
