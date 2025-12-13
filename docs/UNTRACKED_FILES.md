# Untracked Files - Multi-Tenant Infrastructure

## Overview

The following directories contain multi-tenant infrastructure code that was created but **not yet integrated** into the main codebase. These files are functional but unused.

## Untracked Directories

### 1. `internal/server-plugin/authorization/`

**Purpose:** Tool authorization wrapper for multi-tenant access control

**Files:**
- `authorized_tool.go` - Wraps tools with authorization checks
- `authorized_tool_test.go` - Tests for authorization wrapper

**Status:** ✅ Complete, not integrated

**Usage:**
```go
authorizedTool := authorization.WrapToolWithAuthorization(
    tool,
    "apps",           // resource
    "deploy",         // action
    authChecker,      // auth.AuthorizationChecker
    logger,
)
```

**Integration Required:**
- Wire into plugin registration
- Add to multi-tenant configuration
- Enable when `multi_tenant.authorization.enabled: true`

---

### 2. `internal/shared/audit/`

**Purpose:** Audit event logging for compliance and security

**Files:**
- `event.go` - Audit event structure and sink interface

**Status:** ✅ Complete, not integrated

**Features:**
- Event structure with tenant/user context
- EventSink interface for pluggable backends
- NoOpSink for disabled state

**Usage:**
```go
event := audit.Event{
    Timestamp:  time.Now(),
    TenantID:   "tenant-123",
    UserID:     "user-456",
    Action:     "deploy",
    Resource:   "app:my-app",
    Parameters: map[string]interface{}{"git_ref": "main"},
    Result:     "success",
    Duration:   2 * time.Second,
}

sink.Record(ctx, event)
```

**Integration Required:**
- Add to server configuration
- Implement file/database sink
- Wire into tool execution
- Enable when `multi_tenant.observability.audit_enabled: true`

---

### 3. `internal/shared/metrics/`

**Purpose:** Metrics collection for monitoring and observability

**Files:**
- `collector.go` - Metrics collector interface

**Status:** ✅ Complete, not integrated

**Features:**
- Tool execution metrics
- Dokku command metrics
- Tenant activity tracking
- Authentication/authorization metrics
- NoOpCollector for disabled state

**Usage:**
```go
collector.RecordToolExecution(ctx, "deploy_app", duration, success)
collector.RecordDokkuCommand(ctx, "apps:create", duration, success)
collector.RecordTenantActivity(ctx, tenantID)
```

**Integration Required:**
- Add to server configuration
- Implement Prometheus/StatsD backend
- Wire into tool execution
- Enable when `multi_tenant.observability.metrics_enabled: true`

---

## Why Not Committed?

These files were created as part of multi-tenant infrastructure exploration but:

1. **Not currently used** - No imports in active codebase
2. **Feature incomplete** - Multi-tenant mode not fully implemented
3. **Configuration missing** - No config.yaml support yet
4. **No tests in CI** - Would add unused code to coverage reports

## When to Commit?

Commit these files when:

1. ✅ Multi-tenant configuration is added to `config.yaml`
2. ✅ Integration points are implemented
3. ✅ Tests are added and passing
4. ✅ Documentation is updated
5. ✅ Feature flag is ready for use

## Integration Checklist

### Authorization (`internal/server-plugin/authorization/`)

- [ ] Add `authorization.enabled` to config
- [ ] Wire into plugin registration
- [ ] Add authorization checks to sensitive tools
- [ ] Test with multi-tenant setup
- [ ] Document permission model

### Audit Logging (`internal/shared/audit/`)

- [ ] Add `observability.audit_enabled` to config
- [ ] Implement file sink
- [ ] Implement database sink (optional)
- [ ] Wire into tool execution middleware
- [ ] Add audit log rotation
- [ ] Document audit event format

### Metrics (`internal/shared/metrics/`)

- [ ] Add `observability.metrics_enabled` to config
- [ ] Implement Prometheus exporter
- [ ] Wire into tool execution middleware
- [ ] Add metrics endpoint
- [ ] Document available metrics
- [ ] Add Grafana dashboard examples

## Alternative: Delete or Stash?

**Recommendation:** Keep untracked for now

**Reasoning:**
- Code is functional and well-structured
- Will be needed when multi-tenant is implemented
- Not causing any issues (untracked, not in builds)
- Easy to commit when ready

**If you want to clean up:**
```bash
# Option 1: Stash for later
git add internal/server-plugin/authorization/ internal/shared/audit/ internal/shared/metrics/
git stash push -m "Multi-tenant infrastructure (unused)"

# Option 2: Delete (can recover from this document)
rm -rf internal/server-plugin/authorization/ internal/shared/audit/ internal/shared/metrics/
```

## Related Files (Already Committed)

These multi-tenant files **are** committed and in use:

- ✅ `internal/server/auth/interfaces.go` - Auth interfaces
- ✅ `internal/server/auth/noop_auth.go` - No-op authenticator
- ✅ `internal/shared/tenant_context.go` - Tenant context
- ✅ `internal/shared/tenant_context_test.go` - Tests

These are used by `server.go` for optional multi-tenant support.

## Summary

**Untracked files:** 3 directories, 4 files  
**Status:** Complete but unused  
**Action:** Keep untracked until multi-tenant feature is ready  
**Risk:** None (not in builds, not imported)

---

**Last Updated:** 2025-12-13  
**Related:** Multi-tenant feature development
