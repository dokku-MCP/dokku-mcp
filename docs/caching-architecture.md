# Caching Architecture

## Overview

This document explains the **proper** implementation of caching in the Dokku MCP server, which is implemented at the **infrastructure level** rather than the plugin level.

## ✅ Correct Architecture: Infrastructure-Level Caching

### Why Infrastructure-Level?

1. **Separation of Concerns**: Plugins focus on business logic, not caching details
2. **Reusability**: All plugins benefit from caching automatically  
3. **Consistency**: Same caching behavior across all SSH commands
4. **Configuration**: Centrally managed cache policies
5. **Performance**: Cache at the command level, not data aggregation level

### Implementation Location

```
┌─────────────────────────────────────┐
│           Plugin Layer              │
│  (Business Logic - No Caching)     │
├─────────────────────────────────────┤
│        Application Layer            │
│     (Use Cases - No Caching)       │
├─────────────────────────────────────┤
│       Infrastructure Layer          │
│  ✅ CachedDokkuClient (HERE!)      │
│    └── DokkuClient                 │
│        └── SSH Commands            │
└─────────────────────────────────────┘
```

### Key Components

#### 1. CachedDokkuClient
- **Location**: `internal/dokku-api/cached_client.go`
- **Purpose**: Wraps any DokkuClient with command-level caching
- **Interface**: Implements the same `DokkuClient` interface

#### 2. Cache Policies
- **Smart TTL**: Different commands have different cache lifetimes
- **Configurable**: Policies can be customized per deployment

```go
// Example cache policies
var policies = map[string]time.Duration{
    "logs":         30 * time.Second,  // Fast-changing
    "plugin:list":  15 * time.Minute,  // Stable
    "version":      30 * time.Minute,  // Very stable
}
```

#### 3. Configuration Integration
- **Enabled**: Via `cache_enabled: true` in config
- **TTL**: Via `cache_ttl: "5m"` in config
- **Automatic**: No plugin changes required

## 🚫 What We Avoided: Plugin-Level Caching

### Why Plugin-Level is Wrong

```go
// ❌ BAD: Caching in plugin adapter
type DokkuCoreAdapter struct {
    client dokkuApi.DokkuClient
    
    // This is wrong! Caching doesn't belong here
    serverInfoCache *domain.ServerInfo
    cacheExpiry     time.Time
    cacheMutex      sync.RWMutex
}
```

Problems with this approach:
- **Duplication**: Every plugin must implement its own caching
- **Inconsistency**: Different cache behaviors across plugins  
- **Complexity**: Business logic mixed with caching concerns
- **Maintenance**: Cache bugs affect multiple plugins

## Configuration

### Enable Caching

```yaml
# config.yaml
cache_enabled: true
cache_ttl: "5m"
```

### Cache Behavior

| Command Type | Default TTL | Rationale |
|--------------|-------------|-----------|
| `logs` | 30s | Fast-changing data |
| `config:show` | 5m | Semi-stable |
| `plugin:list` | 15m | Stable |
| `version` | 30m | Very stable |

## Performance Impact

### Before (No Caching)
```
dokku://core/server/info request:
├── dokku version              (SSH call 1)
├── dokku plugin:list         (SSH call 2) 
├── dokku domains:report      (SSH call 3)
├── dokku ssh-keys:list       (SSH call 4)
├── dokku proxy:report        (SSH call 5)
├── dokku scheduler:report    (SSH call 6)
└── dokku git:report          (SSH call 7)

Total: 7 SSH calls every time
```

### After (Infrastructure Caching)
```
dokku://core/server/info request:
├── dokku version              (cached 30m)
├── dokku plugin:list         (cached 15m)
├── dokku domains:report      (cached 5m)
├── dokku ssh-keys:list       (cached 10m)
├── dokku proxy:report        (cached 5m)
├── dokku scheduler:report    (cached 5m)
└── dokku git:report          (cached 5m)

Total: 0-7 SSH calls depending on cache state
```

## Usage Examples

### Automatic Usage

No code changes required! Caching happens transparently:

```go
// This automatically uses caching if enabled
plugins, err := a.client.ExecuteCommand(ctx, "plugin:list", []string{})
```

### Cache Control

```go
// Get cached client if you need cache control
if cachedClient, ok := client.(*dokkuApi.CachedDokkuClient); ok {
    cachedClient.InvalidateCache()
}
```

## Benefits

### For Developers
- **No Changes Required**: Existing plugin code works unchanged
- **Automatic Performance**: All commands benefit from caching
- **Configurable**: Easy to tune cache behavior

### For Users  
- **Faster Responses**: Especially for server/info and plugin lists
- **Reduced Load**: Less stress on Dokku server
- **Reliability**: Fewer SSH connection issues

### For Operations
- **Centralized Config**: One place to configure caching
- **Monitoring**: Easy to track cache hit rates
- **Debugging**: Clear separation of caching vs business logic

## Migration Guide

### From Plugin-Level Caching

If you had plugin-level caching:

```diff
- // Remove cache fields from adapters
- type DokkuCoreAdapter struct {
-     client dokkuApi.DokkuClient
-     logger *slog.Logger
-     
-     serverInfoCache *domain.ServerInfo
-     cacheExpiry     time.Time
-     cacheMutex      sync.RWMutex
-     cacheTTL        time.Duration
- }

+ // Simple adapter - no caching concerns  
+ type DokkuCoreAdapter struct {
+     client dokkuApi.DokkuClient
+     logger *slog.Logger
+ }
```

### Configuration

```diff
# config.yaml
+ cache_enabled: true
+ cache_ttl: "5m"
```

That's it! The infrastructure handles the rest.

## Best Practices

### For Plugin Developers
1. **Don't Cache**: Let the infrastructure handle it
2. **Focus on Logic**: Concentrate on business rules
3. **Trust the Client**: DokkuClient handles performance

### For Operators
1. **Monitor Cache Hit Rates**: Check logs for cache effectiveness
2. **Tune TTL**: Adjust based on your change frequency
3. **Consider Workload**: High-frequency access benefits more from caching

### For Performance
1. **Enable Caching**: Unless you need real-time data
2. **Appropriate TTL**: Balance freshness vs performance
3. **Monitor SSH Load**: Caching should reduce SSH connections significantly 