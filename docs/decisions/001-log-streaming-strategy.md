# ADR 001: Log Streaming Strategy

**Status:** Accepted  
**Date:** 2025-12-13  
**Deciders:** Development Team

## Context

dokku-mcp needs to provide access to two types of logs:

1. **Build Logs** - Output from `git push` deployments
2. **Runtime Logs** - Application container logs

### Dokku's Native Infrastructure

**Build Logs:**
- Stream to stdout during `git push` SSH session
- No persistent storage by default
- `logs:failed` retrieves logs from failed deploys (containers kept until GC)
- Build output is ephemeral—exists only during git push

**Runtime Logs:**
- `dokku logs <app>` pulls from Docker container logs
- `-t, --tail` flag enables continuous streaming via Docker's log driver
- Logs persist until container restart or garbage collection
- Vector integration available for external log shipping
- Event logs written to `/var/log/dokku/events.log` via syslog

### MCP Protocol Capabilities

**Streamable HTTP (Recommended - 2025-03-26 spec):**
- Modern standard for MCP
- Single endpoint design
- Supports SSE streams within responses
- Session management via `Mcp-Session-Id` header
- Connection resumption with Event IDs and `Last-Event-ID`

**SSE (Deprecated but functional):**
- Still works for legacy clients
- Persistent connections for server-to-client streaming

**stdio:**
- No native streaming—must poll resources

### Key Insight

Dokku doesn't persist build logs anywhere accessible after `git push` completes. Build output is only visible during the SSH session. For runtime logs, Docker provides native tailing capabilities.

## Decision

**Hybrid Approach:**

```
┌─────────────────────────────────────────────────────────────────┐
│                    LOG STREAMING STRATEGY                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  BUILD LOGS                                                      │
│  ──────────                                                      │
│  • Use POLLING via MCP Resources                                 │
│  • Why: Build logs are captured during async deployment          │
│         execution and stored in deployment tracker               │
│  • Implementation:                                               │
│    - Capture stdout/stderr during SSH command execution          │
│    - Store incrementally in DeploymentTracker                    │
│    - Expose via resource: dokku://deployment/{id}/logs           │
│    - Clients poll during build, get full log on completion       │
│                                                                  │
│  RUNTIME LOGS                                                    │
│  ────────────                                                    │
│  • Use SSE STREAMING for SSE transport                           │
│  • Use POLLING for stdio transport                               │
│  • Why: Runtime logs are continuous and benefit from             │
│         real-time streaming when transport supports it           │
│  • Implementation:                                               │
│    - Wrap `dokku logs <app> -t` for streaming                   │
│    - Detect transport type and adapt strategy                    │
│    - Expose via resource: dokku://app/{name}/logs                │
│    - For stdio: expose as pollable resource                      │
│    - For SSE: stream via SSE events                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Rationale

### Build Logs - Polling

**Why Polling:**
1. Build logs don't exist in Dokku after git push completes
2. DeploymentTracker already captures logs during async execution
3. Logs are finite—build completes, logs are complete
4. Polling is simple and works across all transports
5. No need for complex streaming infrastructure

**Implementation:**
- `DeploymentTracker.AddLogs()` already captures incremental logs
- `Deployment.BuildLogs()` provides full log access
- Expose via MCP resource: `dokku://deployment/{id}/logs`
- Clients poll every 1-2 seconds during active deployment
- Final poll after completion gets full log

### Runtime Logs - Hybrid

**Why Hybrid:**
1. Runtime logs are continuous and unbounded
2. SSE transport can efficiently stream logs
3. stdio transport (majority of clients) needs polling anyway
4. Docker's native tailing maps well to SSE streaming
5. Keeps complexity low—no custom streaming for builds

**Implementation:**
- Detect transport type at runtime
- **SSE Transport:**
  - Wrap `dokku logs <app> -t` in goroutine
  - Stream output via SSE events
  - Handle client disconnection gracefully
  - Implement backpressure if client is slow
- **stdio Transport:**
  - Expose as pollable resource
  - Return last N lines (configurable, default 100)
  - Include timestamp for client-side deduplication

## Consequences

### Positive

1. **Simple Build Log Handling**
   - No streaming complexity for ephemeral logs
   - Works across all transports
   - Leverages existing DeploymentTracker

2. **Efficient Runtime Logs**
   - Real-time streaming when transport supports it
   - Graceful degradation to polling for stdio
   - Leverages Docker's native capabilities

3. **Low Complexity**
   - No custom streaming infrastructure for builds
   - Transport-aware strategy
   - Clear separation of concerns

4. **Good UX**
   - Build logs: clients see progress during deployment
   - Runtime logs: real-time when possible, polling when not
   - Consistent API across transports

### Negative

1. **No Real-Time Build Logs**
   - Clients must poll during builds
   - 1-2 second delay in log visibility
   - Acceptable tradeoff for simplicity

2. **Transport-Specific Behavior**
   - Runtime logs behave differently on SSE vs stdio
   - Clients must handle both patterns
   - Documented in API

3. **Memory Considerations**
   - Build logs stored in memory during deployment
   - Large builds may consume significant memory
   - Mitigated by cleanup TTL (5 minutes)

## Implementation Plan

### Phase 1: Build Logs (Already Implemented)

- ✅ `DeploymentTracker` captures logs
- ✅ `Deployment.AddBuildLogs()` stores incrementally
- ✅ `Deployment.BuildLogs()` provides access
- 🔲 Expose via MCP resource `dokku://deployment/{id}/logs`

### Phase 2: Runtime Logs - Polling (stdio)

1. Add `GetLogs()` method to dokku-api client
   ```go
   func (c *Client) GetLogs(ctx context.Context, appName string, lines int) (string, error)
   ```

2. Expose via MCP resource `dokku://app/{name}/logs`
   - Include `lines` parameter (default 100)
   - Include timestamp in response
   - Support filtering by container

3. Add tool `get_runtime_logs`
   - Parameters: `app_name`, `lines`, `container` (optional)
   - Returns: log lines with timestamps

### Phase 3: Runtime Logs - Streaming (SSE)

1. Add `StreamLogs()` method to dokku-api client
   ```go
   func (c *Client) StreamLogs(ctx context.Context, appName string) (<-chan string, error)
   ```

2. Implement SSE streaming in server
   - Detect SSE transport
   - Wrap `dokku logs <app> -t`
   - Stream via SSE events
   - Handle disconnection

3. Update resource to support streaming
   - Check transport type
   - Return streaming response for SSE
   - Return static response for stdio

### Phase 4: Configuration

1. Add log configuration to `config.yaml`
   ```yaml
   logs:
     runtime:
       default_lines: 100
       max_lines: 1000
       stream_buffer_size: 1000
     build:
       max_size_mb: 10
       retention_minutes: 5
   ```

2. Add environment variables
   - `DOKKU_MCP_LOGS_RUNTIME_DEFAULT_LINES`
   - `DOKKU_MCP_LOGS_RUNTIME_MAX_LINES`
   - `DOKKU_MCP_LOGS_BUILD_MAX_SIZE_MB`

## Alternatives Considered

### Alternative 1: Stream Everything

**Approach:** Stream both build and runtime logs via SSE

**Rejected Because:**
- Build logs are ephemeral in Dokku
- Would require custom log persistence
- Adds complexity for marginal benefit
- Polling works fine for finite logs

### Alternative 2: Poll Everything

**Approach:** Use polling for both build and runtime logs

**Rejected Because:**
- Runtime logs are continuous and unbounded
- Polling is inefficient for long-running logs
- SSE transport capability would be underutilized
- Poor UX for log monitoring

### Alternative 3: External Log Aggregation

**Approach:** Require Vector or similar for log shipping

**Rejected Because:**
- Adds external dependency
- Not all users have log aggregation
- Increases setup complexity
- Should be optional, not required

## References

- [MCP Specification - Streamable HTTP](https://spec.modelcontextprotocol.io/)
- [Dokku Logs Documentation](https://dokku.com/docs/deployment/logs/)
- [Docker Logs API](https://docs.docker.com/engine/api/v1.43/#tag/Container/operation/ContainerLogs)
- [Server-Sent Events Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)

## Notes

- DeploymentTracker already implements build log capture
- Runtime log streaming can be added incrementally
- stdio clients will always use polling (MCP protocol limitation)
- SSE streaming is optional enhancement, not requirement
