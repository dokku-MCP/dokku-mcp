# Log Streaming Decision Summary

## TL;DR

**Build Logs:** Use polling (already implemented via DeploymentTracker)  
**Runtime Logs:** Use streaming for SSE, polling for stdio

## Quick Reference

| Log Type | Transport | Strategy | Status |
|----------|-----------|----------|--------|
| Build | All | Polling via `dokku://deployment/{id}/logs` | ✅ Implemented (needs MCP exposure) |
| Runtime | stdio | Polling via `dokku://app/{name}/logs` | 🔲 To Implement |
| Runtime | SSE | Streaming via SSE events | 🔲 Future Enhancement |

## Why This Approach?

### Build Logs → Polling

**Dokku Reality:**
- Build logs only exist during `git push` SSH session
- No persistent storage after push completes
- Ephemeral by design

**Our Solution:**
- DeploymentTracker captures logs during async execution
- Stores in memory with 5-minute TTL
- Clients poll every 1-2 seconds during build
- Simple, works everywhere, no streaming complexity

### Runtime Logs → Hybrid

**Dokku Reality:**
- `dokku logs <app>` pulls from Docker container logs
- `-t` flag enables continuous tailing
- Logs persist until container restart

**Our Solution:**
- **stdio transport:** Polling (MCP protocol limitation)
- **SSE transport:** Streaming (leverages Docker's native tailing)
- Transport-aware strategy
- Graceful degradation

## Current Implementation Status

### ✅ What's Working

**DeploymentTracker** (`internal/server-plugins/deployment/domain/deployment_tracker.go`):
```go
// Captures build logs during deployment
func (dt *DeploymentTracker) AddLogs(deploymentID string, logs string) error

// Retrieves deployment with logs
func (dt *DeploymentTracker) GetByID(deploymentID string) (*Deployment, error)
```

**Deployment Entity** (`internal/server-plugins/deployment/domain/deployment.entity.go`):
```go
// Stores build logs
func (d *Deployment) AddBuildLogs(logs string)

// Returns full build log
func (d *Deployment) BuildLogs() string
```

### 🔲 What's Needed

1. **Expose Build Logs via MCP Resource**
   - Add `dokku://deployment/{id}/logs` to deployment plugin
   - Return `Deployment.BuildLogs()` content
   - MIME type: `text/plain`

2. **Add Runtime Logs - Polling**
   - Implement `Client.GetLogs()` in dokku-api
   - Create LogsAdapter
   - Add `get_runtime_logs` tool
   - Expose via `dokku://app/{name}/logs`

3. **Add Runtime Logs - Streaming (Future)**
   - Implement `Client.StreamLogs()` in dokku-api
   - Add SSE streaming support
   - Transport detection
   - Backpressure handling

## Implementation Priority

### Sprint 1: Build Logs MCP Exposure
**Effort:** 2-4 hours  
**Value:** High (completes existing feature)

- Modify `deployment_adapter.go` to expose logs resource
- Add to `Resources()` method
- Add to `ReadResource()` method
- Test with MCP Inspector

### Sprint 2: Runtime Logs - Polling
**Effort:** 1-2 days  
**Value:** High (core feature)

- Add `GetLogs()` to dokku-api client
- Create LogsAdapter
- Add `get_runtime_logs` tool
- Add configuration
- Write tests
- Update documentation

### Sprint 3: Runtime Logs - Streaming
**Effort:** 2-3 days  
**Value:** Medium (enhancement)

- Add `StreamLogs()` to dokku-api client
- Implement SSE streaming
- Transport detection
- Backpressure handling
- Write tests
- Update documentation

## Configuration

```yaml
# config.yaml
logs:
  runtime:
    default_lines: 100      # Default log lines to retrieve
    max_lines: 1000         # Maximum log lines allowed
    stream_buffer_size: 1000 # Buffer for SSE streaming
  
  build:
    max_size_mb: 10         # Max build log size in memory
    retention_minutes: 5    # Cleanup TTL for completed deployments
```

## API Examples

### Build Logs (Polling)

**Resource:**
```
dokku://deployment/deploy_1234567890/logs
```

**Response:**
```
-----> Building test-app from Dockerfile
-----> Building image
Step 1/5 : FROM node:18-alpine
 ---> abc123def456
...
-----> Build complete
-----> Releasing test-app
=====> Application deployed
```

### Runtime Logs (Polling - stdio)

**Tool Call:**
```json
{
  "name": "get_runtime_logs",
  "arguments": {
    "app_name": "test-app",
    "lines": 100
  }
}
```

**Response:**
```json
{
  "app_name": "test-app",
  "lines": 100,
  "logs": "2025-12-13T01:30:00Z app[web.1]: Server started on port 5000\n..."
}
```

### Runtime Logs (Streaming - SSE)

**Resource:**
```
dokku://app/test-app/logs
```

**SSE Stream:**
```
event: log
data: {"timestamp": "2025-12-13T01:30:00Z", "container": "web.1", "message": "Request received"}

event: log
data: {"timestamp": "2025-12-13T01:30:01Z", "container": "web.1", "message": "Response sent"}
```

## Performance Characteristics

### Build Logs
- **Memory:** 1-10 MB per deployment (configurable limit)
- **Retention:** 5 minutes after completion (configurable)
- **Polling:** Client-driven, 1-2 second intervals recommended
- **Cleanup:** Automatic via DeploymentTracker

### Runtime Logs - Polling
- **Network:** One SSH command per poll
- **Latency:** ~100-500ms per request
- **Recommended Interval:** 2-5 seconds
- **Resource Usage:** Minimal (SSH connection reused)

### Runtime Logs - Streaming
- **Network:** One persistent SSH connection
- **Latency:** Real-time (< 100ms)
- **Buffer:** 1000 lines (configurable)
- **Resource Usage:** One goroutine per stream

## Security Considerations

1. **Log Sanitization**
   - Already implemented in `internal/server/log_sanitize.go`
   - Redacts API keys, tokens, passwords
   - Applied to all log output

2. **Access Control**
   - Multi-tenant: verify tenant owns app
   - Single-tenant: no restrictions
   - Logs may contain sensitive data

3. **Resource Limits**
   - Enforce `max_lines` to prevent DoS
   - Limit concurrent streams
   - Rate limit log requests

## Testing Strategy

### Unit Tests
- `DeploymentTracker.AddLogs()` ✅ Exists
- `Deployment.BuildLogs()` ✅ Exists
- `Client.GetLogs()` 🔲 To Add
- `Client.StreamLogs()` 🔲 To Add

### Integration Tests
- Build log capture during deployment ✅ Exists
- Build log retrieval via MCP resource 🔲 To Add
- Runtime log polling 🔲 To Add
- Runtime log streaming 🔲 To Add

## Documentation

- ✅ [ADR 001: Log Streaming Strategy](decisions/001-log-streaming-strategy.md)
- ✅ [Implementation Guide](implementation/log-streaming.md)
- 🔲 Update README.md with logs section
- 🔲 Create docs/LOGS.md with usage guide
- 🔲 Update API documentation

## Next Steps

1. **Review this decision** with team
2. **Implement Sprint 1** (Build logs MCP exposure)
3. **Test with MCP Inspector** to validate approach
4. **Gather feedback** from early users
5. **Proceed to Sprint 2** (Runtime logs polling)

## Questions?

- See [ADR 001](decisions/001-log-streaming-strategy.md) for detailed rationale
- See [Implementation Guide](implementation/log-streaming.md) for code examples
- Check existing implementation in `internal/server-plugins/deployment/`

---

**Decision Status:** ✅ Accepted  
**Last Updated:** 2025-12-13  
**Next Review:** After Sprint 1 completion
