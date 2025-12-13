# Phase 1: Expose Build Logs via MCP Resources

**Effort:** 2-4 hours  
**Status:** Ready to implement  
**Prerequisites:** DeploymentTracker already captures logs ✅

## Goal

Expose build logs captured by DeploymentTracker as MCP resources so clients can poll for deployment progress.

## What's Already Working

```go
// internal/server-plugins/deployment/domain/deployment_tracker.go

// ✅ Logs are captured during deployment
func (dt *DeploymentTracker) AddLogs(deploymentID string, logs string) error

// ✅ Deployments can be retrieved with logs
func (dt *DeploymentTracker) GetByID(deploymentID string) (*Deployment, error)

// ✅ Deployment entity stores logs
func (d *Deployment) BuildLogs() string
```

**Evidence:**
- `deployment_poller.go` calls `tracker.AddLogs()` during deployment
- Tests exist in `deployment_tracker_test.go`
- Logs are stored incrementally as deployment progresses

## Implementation Steps

### Step 1: Update Deployment Adapter Resources

**File:** `internal/server-plugins/deployment/adapter/deployment_adapter.go`

**Current Code:**
```go
func (a *DeploymentAdapter) Resources(ctx context.Context) ([]*mcp.Resource, error) {
    deployments := a.tracker.GetAll()
    
    resources := make([]*mcp.Resource, 0, len(deployments))
    
    for _, deployment := range deployments {
        resources = append(resources, &mcp.Resource{
            URI:         fmt.Sprintf("dokku://deployment/%s", deployment.ID()),
            Name:        fmt.Sprintf("Deployment: %s", deployment.AppName()),
            Description: fmt.Sprintf("Status: %s", deployment.Status()),
            MimeType:    "application/json",
        })
    }
    
    return resources, nil
}
```

**Add After Deployment Resource:**
```go
func (a *DeploymentAdapter) Resources(ctx context.Context) ([]*mcp.Resource, error) {
    deployments := a.tracker.GetAll()
    
    // Allocate space for both deployment and logs resources
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
            Description: fmt.Sprintf("Build logs for deployment %s (Status: %s)", 
                                    deployment.ID(), deployment.Status()),
            MimeType:    "text/plain",
        })
    }
    
    return resources, nil
}
```

### Step 2: Update ReadResource to Handle Logs

**File:** `internal/server-plugins/deployment/adapter/deployment_adapter.go`

**Current Code:**
```go
func (a *DeploymentAdapter) ReadResource(ctx context.Context, uri string) (string, error) {
    if !strings.HasPrefix(uri, "dokku://deployment/") {
        return "", fmt.Errorf("unknown resource URI: %s", uri)
    }
    
    deploymentID := strings.TrimPrefix(uri, "dokku://deployment/")
    
    deployment, err := a.tracker.GetByID(deploymentID)
    if err != nil {
        return "", fmt.Errorf("deployment not found: %w", err)
    }
    
    // Return deployment as JSON
    return marshalDeployment(deployment)
}
```

**Update to Handle /logs Suffix:**
```go
func (a *DeploymentAdapter) ReadResource(ctx context.Context, uri string) (string, error) {
    if !strings.HasPrefix(uri, "dokku://deployment/") {
        return "", fmt.Errorf("unknown resource URI: %s", uri)
    }
    
    // Parse URI: dokku://deployment/{id} or dokku://deployment/{id}/logs
    path := strings.TrimPrefix(uri, "dokku://deployment/")
    parts := strings.Split(path, "/")
    
    if len(parts) == 0 || parts[0] == "" {
        return "", fmt.Errorf("invalid deployment URI: %s", uri)
    }
    
    deploymentID := parts[0]
    
    deployment, err := a.tracker.GetByID(deploymentID)
    if err != nil {
        return "", fmt.Errorf("deployment not found: %w", err)
    }
    
    // Check if requesting logs
    if len(parts) > 1 && parts[1] == "logs" {
        // Return raw build logs as text
        logs := deployment.BuildLogs()
        if logs == "" {
            return "No build logs available yet.\n", nil
        }
        return logs, nil
    }
    
    // Return deployment as JSON (existing behavior)
    return marshalDeployment(deployment)
}
```

### Step 3: Test with MCP Inspector

**Start Server:**
```bash
make build
./build/dokku-mcp
```

**In Another Terminal:**
```bash
make inspect
```

**In Browser:**
1. Open the Inspector URL (printed to console)
2. Navigate to "Resources"
3. Look for resources like:
   - `dokku://deployment/deploy_1234567890`
   - `dokku://deployment/deploy_1234567890/logs`
4. Click on a logs resource
5. Verify you see build log output

### Step 4: Test Programmatically

**Create Test File:** `internal/server-plugins/deployment/adapter/deployment_adapter_test.go`

```go
package adapter_test

import (
    "context"
    "testing"
    
    "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/adapter"
    "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
    "github.com/stretchr/testify/assert"
)

func TestDeploymentAdapter_BuildLogsResource(t *testing.T) {
    // Setup
    tracker := domain.NewDeploymentTracker()
    adapter := adapter.NewDeploymentAdapter(tracker)
    
    // Create deployment
    deployment, err := domain.NewDeployment("test-app", "main")
    assert.NoError(t, err)
    
    err = tracker.Track(deployment)
    assert.NoError(t, err)
    
    // Add some logs
    err = tracker.AddLogs(deployment.ID(), "-----> Building test-app\n")
    assert.NoError(t, err)
    
    err = tracker.AddLogs(deployment.ID(), "-----> Build complete\n")
    assert.NoError(t, err)
    
    // Test: Resources should include logs resource
    resources, err := adapter.Resources(context.Background())
    assert.NoError(t, err)
    
    // Should have 2 resources: deployment + logs
    assert.Len(t, resources, 2)
    
    // Find logs resource
    var logsResource *mcp.Resource
    for _, r := range resources {
        if strings.HasSuffix(r.URI, "/logs") {
            logsResource = r
            break
        }
    }
    
    assert.NotNil(t, logsResource)
    assert.Equal(t, "text/plain", logsResource.MimeType)
    assert.Contains(t, logsResource.URI, deployment.ID())
    
    // Test: ReadResource should return logs
    logsURI := fmt.Sprintf("dokku://deployment/%s/logs", deployment.ID())
    logs, err := adapter.ReadResource(context.Background(), logsURI)
    assert.NoError(t, err)
    assert.Contains(t, logs, "Building test-app")
    assert.Contains(t, logs, "Build complete")
}

func TestDeploymentAdapter_EmptyLogs(t *testing.T) {
    // Setup
    tracker := domain.NewDeploymentTracker()
    adapter := adapter.NewDeploymentAdapter(tracker)
    
    // Create deployment without logs
    deployment, err := domain.NewDeployment("test-app", "main")
    assert.NoError(t, err)
    
    err = tracker.Track(deployment)
    assert.NoError(t, err)
    
    // Test: Should return friendly message for empty logs
    logsURI := fmt.Sprintf("dokku://deployment/%s/logs", deployment.ID())
    logs, err := adapter.ReadResource(context.Background(), logsURI)
    assert.NoError(t, err)
    assert.Equal(t, "No build logs available yet.\n", logs)
}

func TestDeploymentAdapter_InvalidLogsURI(t *testing.T) {
    tracker := domain.NewDeploymentTracker()
    adapter := adapter.NewDeploymentAdapter(tracker)
    
    // Test: Non-existent deployment
    _, err := adapter.ReadResource(context.Background(), "dokku://deployment/invalid/logs")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "deployment not found")
}
```

**Run Tests:**
```bash
go test ./internal/server-plugins/deployment/adapter -v -run TestDeploymentAdapter_BuildLogs
```

## Verification Checklist

- [ ] Resources include both deployment and logs URIs
- [ ] Logs resource has MIME type `text/plain`
- [ ] ReadResource returns raw log text for `/logs` URIs
- [ ] Empty logs return friendly message
- [ ] Invalid deployment IDs return error
- [ ] Tests pass
- [ ] MCP Inspector shows logs resources
- [ ] Logs update as deployment progresses

## Expected Behavior

### During Deployment

**Client polls every 2 seconds:**

```
GET dokku://deployment/deploy_1234567890/logs

Response (t=0s):
"No build logs available yet.\n"

Response (t=2s):
"-----> Building test-app from Dockerfile\n"

Response (t=4s):
"-----> Building test-app from Dockerfile\n-----> Building image\nStep 1/5 : FROM node:18-alpine\n"

Response (t=6s):
"-----> Building test-app from Dockerfile\n-----> Building image\nStep 1/5 : FROM node:18-alpine\n ---> abc123def456\n..."
```

### After Deployment Completes

```
GET dokku://deployment/deploy_1234567890/logs

Response:
"-----> Building test-app from Dockerfile
-----> Building image
Step 1/5 : FROM node:18-alpine
 ---> abc123def456
Step 2/5 : WORKDIR /app
 ---> Running in xyz789
...
-----> Build complete
-----> Releasing test-app
=====> Application deployed
       http://test-app.dokku.me
"
```

## Performance Considerations

**Memory:**
- Logs stored in `Deployment.buildLogs` string
- Typical size: 1-5 MB per deployment
- Cleanup after 5 minutes (existing TTL)

**Polling:**
- Client-driven, no server overhead
- Recommended interval: 1-2 seconds during active deployment
- Stop polling when status is `succeeded` or `failed`

## Next Steps

After Phase 1 is complete:

1. **Update Documentation**
   - Add logs section to README.md
   - Document polling strategy
   - Show example client code

2. **Gather Feedback**
   - Test with real deployments
   - Measure log sizes
   - Adjust TTL if needed

3. **Proceed to Phase 2**
   - Implement runtime logs polling
   - Add `get_runtime_logs` tool
   - See [log-streaming.md](log-streaming.md)

## Troubleshooting

**Logs not appearing:**
- Check `deployment_poller.go` is calling `tracker.AddLogs()`
- Verify deployment is tracked: `tracker.GetByID()`
- Check logs aren't empty: `deployment.BuildLogs()`

**Resources not showing:**
- Verify `Resources()` returns logs resources
- Check URI format: `dokku://deployment/{id}/logs`
- Ensure MIME type is `text/plain`

**ReadResource errors:**
- Verify URI parsing logic handles `/logs` suffix
- Check deployment exists in tracker
- Ensure error messages are clear

## Questions?

- See [ADR 001](../decisions/001-log-streaming-strategy.md) for rationale
- See [Implementation Guide](log-streaming.md) for full details
- Check existing code in `internal/server-plugins/deployment/`
