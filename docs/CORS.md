# CORS Configuration

## Overview

dokku-mcp supports Cross-Origin Resource Sharing (CORS) configuration for SSE transport. By default, CORS is **disabled** and the underlying mcp-go library handles CORS with `Access-Control-Allow-Origin: *`.

## Default Behavior

**When CORS is disabled (default):**
- mcp-go sets `Access-Control-Allow-Origin: *`
- All origins can connect to the SSE endpoint
- Token-based authentication provides security
- Suitable for self-hosted deployments with various client setups

## Security Considerations

### ✅ CORS `*` is Acceptable When:

1. **Token-based authentication is required**
   - Bearer tokens in Authorization header
   - Credentials not sent with `*` origin (browser security)
   - Token must be explicitly provided

2. **Clients are non-browser tools**
   - Claude Desktop
   - VS Code extensions
   - CLI tools
   - Backend services
   - These don't have same-origin restrictions

3. **SSE endpoint is read-only**
   - No cookies/credentials sent automatically
   - Actual commands via POST can be restricted separately

### ⚠️ Enable CORS Restrictions For:

1. **Remote deployment**
   - Known client origins
   - Need to track/block malicious origins
   - Compliance requirements

2. **Production deployments with web clients**
   - Browser-based dashboards
   - Web applications accessing the API
   - Need origin allowlist

## Configuration

### Enable CORS Middleware

Add to your `config.yaml`:

```yaml
transport:
  type: sse
  host: localhost
  port: 8080
  cors:
    enabled: true
    allowed_origins:
      - "https://app.example.com"
      - "https://dashboard.example.com"
      - "*.example.com"  # Wildcard subdomain
    allowed_methods:
      - GET
      - POST
      - OPTIONS
    allowed_headers:
      - Content-Type
      - Authorization
    max_age: 300  # 5 minutes
```

### Environment Variables

```bash
export DOKKU_MCP_TRANSPORT_CORS_ENABLED=true
export DOKKU_MCP_TRANSPORT_CORS_ALLOWED_ORIGINS="https://app.example.com,https://dashboard.example.com"
export DOKKU_MCP_TRANSPORT_CORS_ALLOWED_METHODS="GET,POST,OPTIONS"
export DOKKU_MCP_TRANSPORT_CORS_ALLOWED_HEADERS="Content-Type,Authorization"
export DOKKU_MCP_TRANSPORT_CORS_MAX_AGE=300
```

## Configuration Options

### `enabled` (boolean)
- **Default:** `false`
- **Description:** Enable custom CORS middleware. When disabled, mcp-go's default CORS (`*`) is used.

### `allowed_origins` (array of strings)
- **Default:** `[]` (empty = allow all when enabled)
- **Description:** List of allowed origins. Supports:
  - Exact matches: `https://app.example.com`
  - Wildcard subdomains: `*.example.com`
  - Wildcard all: `*`

### `allowed_methods` (array of strings)
- **Default:** `["GET", "POST", "OPTIONS"]`
- **Description:** HTTP methods allowed for CORS requests

### `allowed_headers` (array of strings)
- **Default:** `["Content-Type", "Authorization"]`
- **Description:** HTTP headers allowed in CORS requests

### `max_age` (integer)
- **Default:** `300` (5 minutes)
- **Description:** How long browsers can cache preflight responses (in seconds)

## Examples

### Allow All Origins (Explicit)

```yaml
transport:
  cors:
    enabled: true
    allowed_origins: []  # Empty = allow all
```

### Restrict to Specific Domains

```yaml
transport:
  cors:
    enabled: true
    allowed_origins:
      - "https://app.mydomain.com"
      - "https://dashboard.mydomain.com"
```

### Allow Wildcard Subdomains

```yaml
transport:
  cors:
    enabled: true
    allowed_origins:
      - "*.mydomain.com"
      - "https://mydomain.com"
```

### Development Setup

```yaml
transport:
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
      - "http://localhost:5173"  # Vite dev server
      - "http://127.0.0.1:3000"
```

## Best Practices

### For Self-Hosted Deployments

**Recommendation:** Keep CORS disabled (default)

**Reasoning:**
- Maximizes compatibility
- Users may have various client setups
- Token auth provides security
- Users can add reverse proxy for restrictions if needed

### For Remote deployments

**Recommendation:** Enable CORS with specific origins

**Configuration:**
```yaml
transport:
  cors:
    enabled: true
    allowed_origins:
      - "*.mydomain.com"
    allowed_methods:
      - GET
      - POST
      - OPTIONS
    allowed_headers:
      - Content-Type
      - Authorization
    max_age: 300
```

**Additional Security:**
1. Enforce Authorization header (reject query param tokens)
2. Add rate limiting per origin
3. Enable audit logging
4. Monitor CORS violations

### Token Security

**Always prefer Authorization header over query parameters:**

```bash
# ✅ GOOD: Authorization header
curl -H "Authorization: Bearer YOUR_TOKEN" https://api.example.com/sse

# ❌ BAD: Query parameter (can leak in logs/history)
curl https://api.example.com/sse?token=YOUR_TOKEN
```

## Reverse Proxy Alternative

Instead of enabling CORS in dokku-mcp, you can configure CORS at the reverse proxy level:

### Nginx Example

```nginx
location /sse {
    proxy_pass http://localhost:8080;
    
    # CORS headers
    add_header 'Access-Control-Allow-Origin' 'https://app.example.com' always;
    add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;
    add_header 'Access-Control-Allow-Headers' 'Content-Type, Authorization' always;
    add_header 'Access-Control-Max-Age' '300' always;
    
    # Handle preflight
    if ($request_method = 'OPTIONS') {
        return 204;
    }
    
    # SSE-specific headers
    proxy_set_header Connection '';
    proxy_http_version 1.1;
    chunked_transfer_encoding off;
    proxy_buffering off;
    proxy_cache off;
}
```

### Caddy Example

```caddyfile
example.com {
    @sse path /sse*
    
    handle @sse {
        header {
            Access-Control-Allow-Origin https://app.example.com
            Access-Control-Allow-Methods "GET, POST, OPTIONS"
            Access-Control-Allow-Headers "Content-Type, Authorization"
            Access-Control-Max-Age 300
        }
        
        reverse_proxy localhost:8080
    }
}
```

## Troubleshooting

### CORS Errors in Browser Console

**Error:** `Access to fetch at 'https://api.example.com/sse' from origin 'https://app.example.com' has been blocked by CORS policy`

**Solutions:**
1. Enable CORS middleware: `transport.cors.enabled: true`
2. Add your origin to `allowed_origins`
3. Ensure `allowed_methods` includes the HTTP method you're using
4. Check that `allowed_headers` includes all headers you're sending

### Preflight Requests Failing

**Error:** `Response to preflight request doesn't pass access control check`

**Solutions:**
1. Ensure `OPTIONS` is in `allowed_methods`
2. Add all custom headers to `allowed_headers`
3. Check that origin is in `allowed_origins`

### Wildcard Subdomain Not Working

**Error:** Subdomain requests blocked despite `*.example.com` in config

**Check:**
1. Origin includes protocol: `https://app.example.com` not `app.example.com`
2. Wildcard pattern matches: `*.example.com` matches `app.example.com` but not `example.com`
3. Add base domain separately if needed

## Monitoring

### Log CORS Violations

When CORS is enabled, dokku-mcp logs origin validation:

```json
{
  "level": "info",
  "msg": "CORS middleware enabled",
  "allowed_origins": ["https://app.example.com"],
  "allowed_methods": ["GET", "POST", "OPTIONS"]
}
```

### Audit Logging

Enable audit logging to track CORS requests:

```yaml
multi_tenant:
  observability:
    audit_enabled: true
```

This logs all requests including origin information for security analysis.

## References

- [MDN: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [OWASP: CORS Security](https://owasp.org/www-community/attacks/CORS_OriginHeaderScrutiny)
