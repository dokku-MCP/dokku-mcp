# Log Streaming Implementation Guide

This guide provides implementation details for the log streaming strategy defined in [ADR 001](../decisions/001-log-streaming-strategy.md).

## Overview

dokku-mcp uses a hybrid approach for logs:
- **Build Logs:** Polling via MCP resources (already implemented in DeploymentTracker)
- **Runtime Logs:** Streaming for SSE transport, polling for stdio transport

## Current State

### ✅ Build Logs (Implemented)

The deployment tracking infrastructure already captures build logs:

```go
// internal/server-plugins/deployment/domain/deployment_tracker.go
type DeploymentTracker struct {
    deployments map[string]*TrackedDeployment
    mu          sync.RWMutex
    cleanupTTL  time.Duration
}

// AddLogs appends logs to a tracked deployment
func (dt *DeploymentTracker) AddLogs(deploymentID string, logs string) error
```

**What's Working:**
- Logs captured during SSH command execution
- Stored incrementally in `Deployment.buildLogs`
- Accessible via `Deployment.BuildLogs()`
- Automatic cleanup after 5 minutes

**What's Needed:**
- Expose via MCP resource `dokku://deployment/{id}/logs`
- Add to deployment plugin's resource list

## Implementation Tasks

### Task 1: Expose Build Logs as MCP Resource

**File:** `internal/server-plugins/deployment/adapter/deployment_adapter.go`

```go
// Add to Resources() method
func (a *DeploymentAdapter) Resources(ctx context.Context) ([]*mcp.Resource, error) {
    deployments := a.tracker.GetAll()
    
    resources := make([]*mcp.Resource, 0, len(deployments)*2)
    
    for _, deployment := range deployments {
        // Existing deployment resource
        resources = append(resources, &mcp.Resource{
            URI:         fmt.Sprintf("dokku://deployment/%s", deployment.ID()),
            Name:        fmt.Sprintf("Deployment: %s", deployment.AppName()),
            Description: fmt.Sprintf("Status: %s", deployment.Status()),
            MimeType:    "application/json",
        })
        
        // NEW: Build logs resource
        resources = append(resources, &mcp.Resource{
            URI:         fmt.Sprintf("dokku://deployment/%s/logs", deployment.ID()),
            Name:        fmt.Sprintf("Build Logs: %s", deployment.AppName()),
            Description: fmt.Sprintf("Build logs for deployment %s", deployment.ID()),
            MimeType:    "text/plain",
        })
    }
    
    return resources, nil
}

// Add to ReadResource() method
func (a *DeploymentAdapter) ReadResource(ctx context.Context, uri string) (string, error) {
    // Parse URI
    if strings.HasPrefix(uri, "dokku://deployment/") {
        parts := strings.Split(strings.TrimPrefix(uri, "dokku://deployment/"), "/")
        deploymentID := parts[0]
        
        deployment, err := a.tracker.GetByID(deploymentID)
        if err != nil {
            return "", fmt.Errorf("deployment not found: %w", err)
        }
        
        // Check if requesting logs
        if len(parts) > 1 && parts[1] == "logs" {
            return deployment.BuildLogs(), nil
        }
        
        // Existing: return deployment JSON
        return marshalDeployment(deployment)
    }
    
    return "", fmt.Errorf("unknown resource URI: %s", uri)
}
```

### Task 2: Add Runtime Logs to dokku-api Client

**File:** `internal/dokku-api/client.go`

```go
// GetLogs retrieves application logs
func (c *Client) GetLogs(ctx context.Context, appName string, options LogOptions) (string, error) {
    if appName == "" {
        return "", fmt.Errorf("application name cannot be empty")
    }
    
    args := []string{"logs", appName}
    
    if options.Lines > 0 {
        args = append(args, "--num", fmt.Sprintf("%d", options.Lines))
    }
    
    if options.Tail {
        return "", fmt.Errorf("use StreamLogs for tailing logs")
    }
    
    output, err := c.executeCommand(ctx, args...)
    if err != nil {
        return "", fmt.Errorf("failed to get logs: %w", err)
    }
    
    return string(output), nil
}

// StreamLogs streams application logs (for SSE transport)
func (c *Client) StreamLogs(ctx context.Context, appName string) (<-chan LogLine, <-chan error, error) {
    if appName == "" {
        return nil, nil, fmt.Errorf("application name cannot be empty")
    }
    
    logChan := make(chan LogLine, 100)
    errChan := make(chan error, 1)
    
    go func() {
        defer close(logChan)
        defer close(errChan)
        
        args := []string{"logs", appName, "-t"}
        
        cmd := c.buildCommand(ctx, args...)
        
        stdout, err := cmd.StdoutPipe()
        if err != nil {
            errChan <- fmt.Errorf("failed to create stdout pipe: %w", err)
            return
        }
        
        if err := cmd.Start(); err != nil {
            errChan <- fmt.Errorf("failed to start command: %w", err)
            return
        }
        
        scanner := bufio.NewScanner(stdout)
        for scanner.Scan() {
            line := scanner.Text()
            
            select {
            case logChan <- parseLogLine(line):
            case <-ctx.Done():
                cmd.Process.Kill()
                return
            }
        }
        
        if err := scanner.Err(); err != nil {
            errChan <- fmt.Errorf("error reading logs: %w", err)
        }
        
        cmd.Wait()
    }()
    
    return logChan, errChan, nil
}
```

**File:** `internal/dokku-api/client.types.go`

```go
// LogOptions configures log retrieval
type LogOptions struct {
    Lines int  // Number of lines to retrieve (0 = all)
    Tail  bool // Follow log output (use StreamLogs instead)
}

// LogLine represents a single log line
type LogLine struct {
    Timestamp time.Time
    Container string
    Message   string
}

// parseLogLine parses a Dokku log line
// Format: "2025-12-13T01:30:00.000000000Z app[web.1]: message"
func parseLogLine(line string) LogLine {
    // Simple parsing - enhance as needed
    parts := strings.SplitN(line, " ", 3)
    if len(parts) < 3 {
        return LogLine{
            Timestamp: time.Now(),
            Message:   line,
        }
    }
    
    timestamp, _ := time.Parse(time.RFC3339Nano, parts[0])
    container := strings.Trim(parts[1], ":")
    
    return LogLine{
        Timestamp: timestamp,
        Container: container,
        Message:   parts[2],
    }
}
```

### Task 3: Add Runtime Logs Resource (stdio - Polling)

**File:** `internal/server-plugins/app/adapter/logs_adapter.go` (new file)

```go
package adapter

import (
    "context"
    "fmt"
    "encoding/json"
    
    "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
    "github.com/mark3labs/mcp-go/mcp"
)

type LogsAdapter struct {
    client dokku.Client
    config LogsConfig
}

type LogsConfig struct {
    DefaultLines int
    MaxLines     int
}

func NewLogsAdapter(client dokku.Client, config LogsConfig) *LogsAdapter {
    if config.DefaultLines == 0 {
        config.DefaultLines = 100
    }
    if config.MaxLines == 0 {
        config.MaxLines = 1000
    }
    
    return &LogsAdapter{
        client: client,
        config: config,
    }
}

// GetLogsResource returns MCP resource for app logs
func (a *LogsAdapter) GetLogsResource(appName string) *mcp.Resource {
    return &mcp.Resource{
        URI:         fmt.Sprintf("dokku://app/%s/logs", appName),
        Name:        fmt.Sprintf("Runtime Logs: %s", appName),
        Description: fmt.Sprintf("Application runtime logs for %s", appName),
        MimeType:    "application/json",
    }
}

// ReadLogsResource retrieves logs for an app
func (a *LogsAdapter) ReadLogsResource(ctx context.Context, appName string, lines int) (string, error) {
    if lines == 0 {
        lines = a.config.DefaultLines
    }
    if lines > a.config.MaxLines {
        lines = a.config.MaxLines
    }
    
    logs, err := a.client.GetLogs(ctx, appName, dokku.LogOptions{
        Lines: lines,
    })
    if err != nil {
        return "", fmt.Errorf("failed to get logs: %w", err)
    }
    
    // Return as JSON for structured access
    response := map[string]interface{}{
        "app_name": appName,
        "lines":    lines,
        "logs":     logs,
    }
    
    data, err := json.MarshalIndent(response, "", "  ")
    if err != nil {
        return "", fmt.Errorf("failed to marshal logs: %w", err)
    }
    
    return string(data), nil
}
```

### Task 4: Add Runtime Logs Tool

**File:** `internal/server-plugins/app/domain/logs.commands.go` (new file)

```go
package domain

import (
    "context"
    "fmt"
    
    "github.com/mark3labs/mcp-go/mcp"
)

// GetRuntimeLogsTool returns the MCP tool definition
func GetRuntimeLogsTool() *mcp.Tool {
    return &mcp.Tool{
        Name:        "get_runtime_logs",
        Description: "Retrieve runtime logs from a Dokku application",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "app_name": map[string]interface{}{
                    "type":        "string",
                    "description": "Name of the application",
                },
                "lines": map[string]interface{}{
                    "type":        "integer",
                    "description": "Number of log lines to retrieve (default: 100, max: 1000)",
                    "default":     100,
                },
            },
            Required: []string{"app_name"},
        },
    }
}

// GetRuntimeLogsCommand executes the get_runtime_logs tool
type GetRuntimeLogsCommand struct {
    AppName string
    Lines   int
}

func NewGetRuntimeLogsCommand(appName string, lines int) (*GetRuntimeLogsCommand, error) {
    if appName == "" {
        return nil, fmt.Errorf("app_name is required")
    }
    
    if lines == 0 {
        lines = 100
    }
    
    return &GetRuntimeLogsCommand{
        AppName: appName,
        Lines:   lines,
    }, nil
}
```

### Task 5: Add Configuration

**File:** `pkg/config/config.go`

```go
type LogsConfig struct {
    Runtime RuntimeLogsConfig `mapstructure:"runtime"`
    Build   BuildLogsConfig   `mapstructure:"build"`
}

type RuntimeLogsConfig struct {
    DefaultLines     int `mapstructure:"default_lines"`
    MaxLines         int `mapstructure:"max_lines"`
    StreamBufferSize int `mapstructure:"stream_buffer_size"`
}

type BuildLogsConfig struct {
    MaxSizeMB         int `mapstructure:"max_size_mb"`
    RetentionMinutes  int `mapstructure:"retention_minutes"`
}

// Add to ServerConfig
type ServerConfig struct {
    // ... existing fields ...
    Logs LogsConfig `mapstructure:"logs"`
}

// Add to DefaultConfig()
func DefaultConfig() *ServerConfig {
    return &ServerConfig{
        // ... existing defaults ...
        Logs: LogsConfig{
            Runtime: RuntimeLogsConfig{
                DefaultLines:     100,
                MaxLines:         1000,
                StreamBufferSize: 1000,
            },
            Build: BuildLogsConfig{
                MaxSizeMB:        10,
                RetentionMinutes: 5,
            },
        },
    }
}
```

**File:** `config.yaml.example`

```yaml
# Logs configuration
logs:
  runtime:
    default_lines: 100      # Default number of log lines to retrieve
    max_lines: 1000         # Maximum number of log lines allowed
    stream_buffer_size: 1000 # Buffer size for log streaming (SSE only)
  
  build:
    max_size_mb: 10         # Maximum build log size in memory
    retention_minutes: 5    # How long to keep completed deployment logs
```

## Testing Strategy

### Unit Tests

**Build Logs:**
```go
// internal/server-plugins/deployment/domain/deployment_tracker_test.go
func TestDeploymentTracker_AddLogs(t *testing.T) {
    tracker := NewDeploymentTracker()
    deployment, _ := NewDeployment("test-app", "main")
    tracker.Track(deployment)
    
    err := tracker.AddLogs(deployment.ID(), "Building...\n")
    assert.NoError(t, err)
    
    err = tracker.AddLogs(deployment.ID(), "Complete!\n")
    assert.NoError(t, err)
    
    retrieved, _ := tracker.GetByID(deployment.ID())
    assert.Equal(t, "Building...\nComplete!\n", retrieved.BuildLogs())
}
```

**Runtime Logs:**
```go
// internal/dokku-api/client_test.go
func TestClient_GetLogs(t *testing.T) {
    client := setupTestClient()
    
    logs, err := client.GetLogs(context.Background(), "test-app", LogOptions{
        Lines: 10,
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, logs)
}
```

### Integration Tests

```go
// internal/server-plugins/app/integration_test.go
func TestLogsResource_Polling(t *testing.T) {
    // Setup test app with logs
    // Read logs resource
    // Verify log content
}
```

## Performance Considerations

### Build Logs

**Memory Usage:**
- Logs stored in memory during deployment
- Typical build: 1-5 MB
- Large builds: up to 10 MB (configurable limit)
- Cleanup after 5 minutes (configurable)

**Mitigation:**
- Set `logs.build.max_size_mb` to limit memory
- Adjust `logs.build.retention_minutes` for faster cleanup
- Consider truncating very large logs

### Runtime Logs

**Polling (stdio):**
- Each poll executes `dokku logs <app> --num N`
- Recommended poll interval: 2-5 seconds
- Network overhead: minimal (SSH connection reused)

**Streaming (SSE):**
- One SSH connection per stream
- Buffer size: 1000 lines (configurable)
- Backpressure handling: drop old lines if buffer full
- Automatic cleanup on client disconnect

## Security Considerations

1. **Log Sanitization**
   - Already implemented in `internal/server/log_sanitize.go`
   - Redacts sensitive patterns (API keys, tokens, passwords)
   - Applied to both build and runtime logs

2. **Access Control**
   - Multi-tenant mode: verify tenant owns app
   - Single-tenant mode: no restrictions
   - Logs may contain sensitive data—warn users

3. **Resource Limits**
   - Enforce `max_lines` to prevent DoS
   - Limit concurrent streams per client
   - Rate limit log requests

## Migration Path

### Phase 1: Build Logs (Current Sprint)
- ✅ DeploymentTracker captures logs
- 🔲 Expose via MCP resource
- 🔲 Add to deployment plugin
- 🔲 Update documentation

### Phase 2: Runtime Logs - Polling (Next Sprint)
- 🔲 Add `GetLogs()` to dokku-api
- 🔲 Create LogsAdapter
- 🔲 Add `get_runtime_logs` tool
- 🔲 Expose via MCP resource
- 🔲 Add configuration

### Phase 3: Runtime Logs - Streaming (Future)
- 🔲 Add `StreamLogs()` to dokku-api
- 🔲 Implement SSE streaming in server
- 🔲 Add transport detection
- 🔲 Update resource to support streaming
- 🔲 Add backpressure handling

## Documentation Updates

1. **README.md**
   - Add "Logs" section
   - Explain polling vs streaming
   - Show example usage

2. **docs/LOGS.md** (new)
   - Detailed log access guide
   - Configuration options
   - Best practices
   - Troubleshooting

3. **API Documentation**
   - Document `dokku://deployment/{id}/logs` resource
   - Document `dokku://app/{name}/logs` resource
   - Document `get_runtime_logs` tool

## References

- [ADR 001: Log Streaming Strategy](../decisions/001-log-streaming-strategy.md)
- [Dokku Logs Documentation](https://dokku.com/docs/deployment/logs/)
- [MCP Resources Specification](https://spec.modelcontextprotocol.io/)
