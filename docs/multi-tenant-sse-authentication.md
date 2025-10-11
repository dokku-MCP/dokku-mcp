# Dokku-MCP Multi-Tenant SSE Authentication Architecture

## BLUF (Bottom Line Up Front)

**Recommendation:** Use mcp-go's **native `WithSSEContextFunc`** option to inject tenant authentication context directly into the MCP server, following Domain-Driven Design principles with a dedicated Authentication bounded context.

**Key Insight:** The mcp-go library provides `SSEContextFunc` for customizing request context—this is the **protocol-native approach** for authentication, eliminating the need for external proxies while maintaining clean DDD architecture.

---

## 🎯 MCP-Native DDD Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   Domain Layer (Pure Business Logic)             │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │         Authentication Domain (Bounded Context)           │   │
│  │                                                            │   │
│  │  • TenantIdentity (Entity)                                │   │
│  │  • AccessToken (Value Object)                             │   │
│  │  • AuthenticationService (Domain Service)                 │   │
│  │  • TenantRepository (Interface)                           │   │
│  │  • SecretProvider (Interface)                             │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │         Dokku Management Domain (Existing)                │   │
│  │                                                            │   │
│  │  • Application, Deployment entities                       │   │
│  │  • Requires tenant context for authorization              │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              ↑
                              │ Uses domain interfaces
                              │
┌─────────────────────────────────────────────────────────────────┐
│              Application Layer (Use Case Orchestration)          │
│                                                                   │
│  • AuthenticateSSEConnection (Use Case)                          │
│  • ValidateTenantAccess (Use Case)                               │
│  • ProvisionTenantSecrets (Use Case)                             │
└─────────────────────────────────────────────────────────────────┘
                              ↑
                              │ Implements domain interfaces
                              │
┌─────────────────────────────────────────────────────────────────┐
│           Infrastructure Layer (External Systems)                │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  MCP Transport Infrastructure (SSE with Auth Context)     │   │
│  │                                                            │   │
│  │  • SSEContextAuthenticator (implements SSEContextFunc)    │   │
│  │  • Injects tenant context into mcp-go                     │   │
│  │  • Validates tokens from query params or headers          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Repository Implementations                               │   │
│  │                                                            │   │
│  │  • VaultTenantRepository (HashiCorp Vault)                │   │
│  │  • PostgresTenantRepository (Database)                    │   │
│  │  • VaultSecretProvider (Dynamic SSH keys)                 │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔐 Authentication Flow with SSEContextFunc

### Phase 1: Client Connection Request

```javascript
// Client initiates SSE connection with authentication token
const eventSource = new EventSource(
  'https://dokku-api.example.com/sse?' + 
  'token=eyJhbGciOiJIUzI1NiIs...' +
  '&tenant=abc'
);

eventSource.onopen = () => {
  console.log('Connected to Dokku MCP');
};

eventSource.onmessage = (event) => {
  const mcpMessage = JSON.parse(event.data);
  handleMCPProtocolMessage(mcpMessage);
};
```

### Phase 2: Server-Side Context Injection (MCP-Native)

```go
// internal/server/server.go - Enhanced with authentication

func registerServerHooks(
    lc fx.Lifecycle,
    cfg *config.ServerConfig,
    mcpServer *server.MCPServer,
    adapter *MCPAdapter,
    dynamicRegistry *plugins.DynamicServerPluginRegistry,
    authService *domain.AuthenticationService, // NEW: Inject auth service
    logger *slog.Logger,
) {
    var httpServer *http.Server

    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            // ... existing plugin registration ...

            switch cfg.Transport.Type {
            case "sse":
                logger.Info("Starting MCP server with SSE transport and authentication")
                
                // Create SSE context authenticator (MCP-native approach)
                contextAuth := infrastructure.NewSSEContextAuthenticator(
                    authService,
                    logger,
                )
                
                // Create SSE server with authentication context function
                sseServer := server.NewSSEServer(
                    mcpServer,
                    server.WithSSEContextFunc(contextAuth.InjectTenantContext),
                    server.WithSSEEndpoint("/sse"),
                    server.WithMessageEndpoint("/message"),
                    server.WithKeepAlive(true),
                    server.WithKeepAliveInterval(30*time.Second),
                )
                
                // Start SSE server
                go func() {
                    addr := fmt.Sprintf("%s:%d", cfg.Transport.Host, cfg.Transport.Port)
                    logger.Info("SSE server listening with authentication",
                        "address", addr)
                    
                    if err := sseServer.Start(addr); err != nil && err != http.ErrServerClosed {
                        logger.Error("SSE server failed", "error", err)
                    }
                }()
                
                httpServer = sseServer.HTTPServer
                
            // ... stdio case unchanged ...
            }
            return nil
        },
        // ... OnStop unchanged ...
    })
}
```

---

## 📦 DDD Implementation: Authentication Bounded Context

### Domain Layer: Authentication Entities and Value Objects

```go
// internal/authentication/domain/tenant_identity.go

package domain

import (
    "time"
    "github.com/google/uuid"
)

// TenantIdentity represents an authenticated tenant in the system (Entity)
type TenantIdentity struct {
    id          uuid.UUID
    tenantID    string
    clientID    string
    permissions []Permission
    dokkuConfig DokkuConnectionConfig
    createdAt   time.Time
    expiresAt   time.Time
}

// NewTenantIdentity creates a new authenticated tenant identity
func NewTenantIdentity(
    tenantID string,
    clientID string,
    permissions []Permission,
    dokkuConfig DokkuConnectionConfig,
    ttl time.Duration,
) (*TenantIdentity, error) {
    if tenantID == "" {
        return nil, ErrInvalidTenantID
    }
    if clientID == "" {
        return nil, ErrInvalidClientID
    }
    
    return &TenantIdentity{
        id:          uuid.New(),
        tenantID:    tenantID,
        clientID:    clientID,
        permissions: permissions,
        dokkuConfig: dokkuConfig,
        createdAt:   time.Now(),
        expiresAt:   time.Now().Add(ttl),
    }, nil
}

// ID returns the unique identifier for this identity
func (t *TenantIdentity) ID() uuid.UUID { return t.id }

// TenantID returns the tenant identifier
func (t *TenantIdentity) TenantID() string { return t.tenantID }

// ClientID returns the client identifier
func (t *TenantIdentity) ClientID() string { return t.clientID }

// IsExpired checks if the identity has expired
func (t *TenantIdentity) IsExpired() bool {
    return time.Now().After(t.expiresAt)
}

// HasPermission checks if tenant has specific permission
func (t *TenantIdentity) HasPermission(required Permission) bool {
    for _, p := range t.permissions {
        if p.Allows(required) {
            return true
        }
    }
    return false
}

// DokkuConfig returns the Dokku connection configuration
func (t *TenantIdentity) DokkuConfig() DokkuConnectionConfig {
    return t.dokkuConfig
}

// DokkuConnectionConfig holds tenant-specific Dokku connection details
type DokkuConnectionConfig struct {
    Host        string
    Port        int
    User        string
    SSHKeyPath  string
}
```

```go
// internal/authentication/domain/access_token.go

package domain

import (
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "time"
)

// AccessToken represents an authentication token (Value Object)
type AccessToken struct {
    value     string
    tenantID  string
    scope     []Permission
    issuedAt  time.Time
    expiresAt time.Time
}

// NewAccessToken creates a new access token with validation
func NewAccessToken(
    value string,
    tenantID string,
    scope []Permission,
    expiresAt time.Time,
) (*AccessToken, error) {
    if value == "" {
        return nil, ErrInvalidToken
    }
    if tenantID == "" {
        return nil, ErrInvalidTenantID
    }
    if expiresAt.Before(time.Now()) {
        return nil, ErrTokenExpired
    }
    
    return &AccessToken{
        value:     value,
        tenantID:  tenantID,
        scope:     scope,
        issuedAt:  time.Now(),
        expiresAt: expiresAt,
    }, nil
}

// Value returns the token string (never logged)
func (t *AccessToken) Value() string { return t.value }

// TenantID returns the associated tenant
func (t *AccessToken) TenantID() string { return t.tenantID }

// IsExpired checks token expiration
func (t *AccessToken) IsExpired() bool {
    return time.Now().After(t.expiresAt)
}

// HasScope checks if token has required permission
func (t *AccessToken) HasScope(required Permission) bool {
    for _, s := range t.scope {
        if s.Allows(required) {
            return true
        }
    }
    return false
}

// Fingerprint returns a safe hash for logging
func (t *AccessToken) Fingerprint() string {
    hash := sha256.Sum256([]byte(t.value))
    return base64.RawURLEncoding.EncodeToString(hash[:8])
}

// Equals compares two access tokens securely
func (t *AccessToken) Equals(other *AccessToken) bool {
    if other == nil {
        return false
    }
    // Constant-time comparison to prevent timing attacks
    return secureCompare(t.value, other.value)
}
```

```go
// internal/authentication/domain/permission.go

package domain

// Permission represents a specific authorization scope
type Permission string

const (
    PermissionAppsRead       Permission = "apps:read"
    PermissionAppsCreate     Permission = "apps:create"
    PermissionAppsDeploy     Permission = "apps:deploy"
    PermissionAppsDestroy    Permission = "apps:destroy"
    PermissionAppsScale      Permission = "apps:scale"
    PermissionConfigRead     Permission = "config:read"
    PermissionConfigWrite    Permission = "config:write"
    PermissionDomainsManage  Permission = "domains:manage"
    PermissionLogsRead       Permission = "logs:read"
    PermissionPluginsManage  Permission = "plugins:manage"
)

// Allows checks if this permission grants access to the required permission
func (p Permission) Allows(required Permission) bool {
    // Direct match
    if p == required {
        return true
    }
    
    // Wildcard permissions
    if p == "apps:*" && isAppsPermission(required) {
        return true
    }
    if p == "*" {
        return true
    }
    
    return false
}

func isAppsPermission(p Permission) bool {
    return p == PermissionAppsRead ||
           p == PermissionAppsCreate ||
           p == PermissionAppsDeploy ||
           p == PermissionAppsDestroy ||
           p == PermissionAppsScale
}
```

### Domain Layer: Authentication Service

```go
// internal/authentication/domain/authentication_service.go

package domain

import (
    "context"
    "fmt"
)

// AuthenticationService handles tenant authentication (Domain Service)
type AuthenticationService struct {
    tenantRepo     TenantRepository
    secretProvider SecretProvider
}

// NewAuthenticationService creates a new authentication service
func NewAuthenticationService(
    tenantRepo TenantRepository,
    secretProvider SecretProvider,
) *AuthenticationService {
    return &AuthenticationService{
        tenantRepo:     tenantRepo,
        secretProvider: secretProvider,
    }
}

// AuthenticateToken validates a token and returns tenant identity
func (s *AuthenticationService) AuthenticateToken(
    ctx context.Context,
    token string,
) (*TenantIdentity, error) {
    // Validate token structure and signature
    accessToken, err := s.parseAndValidateToken(token)
    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }
    
    // Check expiration
    if accessToken.IsExpired() {
        return nil, ErrTokenExpired
    }
    
    // Retrieve tenant from repository
    tenant, err := s.tenantRepo.GetByID(ctx, accessToken.TenantID())
    if err != nil {
        return nil, fmt.Errorf("tenant not found: %w", err)
    }
    
    // Check tenant status
    if !tenant.IsActive() {
        return nil, ErrTenantInactive
    }
    
    // Get tenant-specific Dokku credentials from secret provider
    dokkuConfig, err := s.secretProvider.GetDokkuConfig(ctx, tenant.ID())
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve tenant credentials: %w", err)
    }
    
    // Create authenticated identity
    identity, err := NewTenantIdentity(
        tenant.ID(),
        accessToken.ClientID(),
        accessToken.Scope(),
        dokkuConfig,
        15*time.Minute, // Session TTL
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create identity: %w", err)
    }
    
    return identity, nil
}

// AuthorizeAction checks if tenant has permission for an action
func (s *AuthenticationService) AuthorizeAction(
    identity *TenantIdentity,
    required Permission,
) error {
    if identity == nil {
        return ErrUnauthorized
    }
    
    if identity.IsExpired() {
        return ErrSessionExpired
    }
    
    if !identity.HasPermission(required) {
        return ErrInsufficientPermissions
    }
    
    return nil
}

// parseAndValidateToken parses JWT and creates AccessToken
func (s *AuthenticationService) parseAndValidateToken(tokenStr string) (*AccessToken, error) {
    // Parse JWT, validate signature, extract claims
    // Implementation depends on JWT library (e.g., golang-jwt/jwt)
    // ... JWT parsing logic ...
    
    return NewAccessToken(tokenStr, tenantID, permissions, expiresAt)
}
```

```go
// internal/authentication/domain/repository.go

package domain

import "context"

// TenantRepository defines storage operations for tenants (Interface)
type TenantRepository interface {
    GetByID(ctx context.Context, tenantID string) (*Tenant, error)
    GetByClientID(ctx context.Context, clientID string) (*Tenant, error)
    Save(ctx context.Context, tenant *Tenant) error
}

// SecretProvider defines operations for retrieving tenant secrets (Interface)
type SecretProvider interface {
    GetDokkuConfig(ctx context.Context, tenantID string) (DokkuConnectionConfig, error)
    RotateSSHKey(ctx context.Context, tenantID string) error
}

// Tenant represents a tenant in the system
type Tenant struct {
    id        string
    name      string
    isActive  bool
    createdAt time.Time
}

func (t *Tenant) ID() string { return t.id }
func (t *Tenant) IsActive() bool { return t.isActive }
```

---

### Infrastructure Layer: SSE Context Authenticator

```go
// internal/authentication/infrastructure/sse_context_authenticator.go

package infrastructure

import (
    "context"
    "log/slog"
    "net/http"
    
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/domain"
)

// SSEContextAuthenticator implements SSE context injection for authentication
// This is the bridge between mcp-go's SSEContextFunc and our domain logic
type SSEContextAuthenticator struct {
    authService *domain.AuthenticationService
    logger      *slog.Logger
}

// NewSSEContextAuthenticator creates a new SSE context authenticator
func NewSSEContextAuthenticator(
    authService *domain.AuthenticationService,
    logger *slog.Logger,
) *SSEContextAuthenticator {
    return &SSEContextAuthenticator{
        authService: authService,
        logger:      logger,
    }
}

// InjectTenantContext is the SSEContextFunc implementation
// This function is called by mcp-go for every SSE connection request
func (a *SSEContextAuthenticator) InjectTenantContext(
    ctx context.Context,
    r *http.Request,
) context.Context {
    // Extract authentication token from query parameter
    // (SSE EventSource API doesn't support custom headers)
    token := r.URL.Query().Get("token")
    
    if token == "" {
        // No token provided - log and continue with empty context
        a.logger.Warn("SSE connection attempted without authentication token",
            "remote_addr", r.RemoteAddr)
        return ctx
    }
    
    // Authenticate using domain service
    identity, err := a.authService.AuthenticateToken(ctx, token)
    if err != nil {
        // Authentication failed - log error
        a.logger.Error("SSE authentication failed",
            "error", err,
            "remote_addr", r.RemoteAddr)
        
        // Inject error into context so handlers can reject the request
        return context.WithValue(ctx, "auth_error", err)
    }
    
    // Successfully authenticated - inject tenant identity into context
    ctx = context.WithValue(ctx, "tenant_identity", identity)
    
    // Also inject Dokku configuration for this tenant
    dokkuConfig := identity.DokkuConfig()
    ctx = context.WithValue(ctx, "dokku_ssh_host", dokkuConfig.Host)
    ctx = context.WithValue(ctx, "dokku_ssh_port", dokkuConfig.Port)
    ctx = context.WithValue(ctx, "dokku_ssh_user", dokkuConfig.User)
    ctx = context.WithValue(ctx, "dokku_ssh_key_path", dokkuConfig.SSHKeyPath)
    
    // Create tenant-scoped logger
    tenantLogger := a.logger.With(
        "tenant_id", identity.TenantID(),
        "client_id", identity.ClientID(),
    )
    ctx = context.WithValue(ctx, "logger", tenantLogger)
    
    a.logger.Info("SSE connection authenticated",
        "tenant_id", identity.TenantID(),
        "client_id", identity.ClientID(),
        "remote_addr", r.RemoteAddr)
    
    return ctx
}

// Helper function to extract tenant identity from context
func GetTenantIdentity(ctx context.Context) (*domain.TenantIdentity, error) {
    // Check for authentication error first
    if authErr, ok := ctx.Value("auth_error").(error); ok {
        return nil, authErr
    }
    
    identity, ok := ctx.Value("tenant_identity").(*domain.TenantIdentity)
    if !ok || identity == nil {
        return nil, domain.ErrUnauthorized
    }
    
    if identity.IsExpired() {
        return nil, domain.ErrSessionExpired
    }
    
    return identity, nil
}
```

---

### Infrastructure Layer: Vault Secret Provider

```go
// internal/authentication/infrastructure/vault_secret_provider.go

package infrastructure

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    
    vault "github.com/hashicorp/vault/api"
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/domain"
)

// VaultSecretProvider implements SecretProvider using HashiCorp Vault
type VaultSecretProvider struct {
    vaultClient *vault.Client
    mountPath   string
}

// NewVaultSecretProvider creates a new Vault secret provider
func NewVaultSecretProvider(vaultAddr, mountPath string) (*VaultSecretProvider, error) {
    config := vault.DefaultConfig()
    config.Address = vaultAddr
    
    client, err := vault.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create Vault client: %w", err)
    }
    
    return &VaultSecretProvider{
        vaultClient: client,
        mountPath:   mountPath,
    }, nil
}

// GetDokkuConfig retrieves tenant-specific Dokku configuration from Vault
func (v *VaultSecretProvider) GetDokkuConfig(
    ctx context.Context,
    tenantID string,
) (domain.DokkuConnectionConfig, error) {
    // Read tenant configuration from Vault
    secretPath := fmt.Sprintf("%s/tenants/%s/dokku", v.mountPath, tenantID)
    secret, err := v.vaultClient.Logical().ReadWithContext(ctx, secretPath)
    if err != nil {
        return domain.DokkuConnectionConfig{}, fmt.Errorf("failed to read Vault secret: %w", err)
    }
    
    if secret == nil || secret.Data == nil {
        return domain.DokkuConnectionConfig{}, fmt.Errorf("no configuration found for tenant %s", tenantID)
    }
    
    data := secret.Data["data"].(map[string]interface{})
    
    // Materialize SSH private key to temporary file
    privateKey := data["ssh_private_key"].(string)
    keyPath, err := v.materializeSSHKey(tenantID, privateKey)
    if err != nil {
        return domain.DokkuConnectionConfig{}, fmt.Errorf("failed to materialize SSH key: %w", err)
    }
    
    return domain.DokkuConnectionConfig{
        Host:       data["ssh_host"].(string),
        Port:       int(data["ssh_port"].(float64)),
        User:       data["ssh_user"].(string),
        SSHKeyPath: keyPath,
    }, nil
}

// materializeSSHKey writes SSH private key to secure temporary file
func (v *VaultSecretProvider) materializeSSHKey(tenantID, privateKey string) (string, error) {
    // Create tenant-specific directory in /tmp with secure permissions
    keyDir := filepath.Join("/tmp", "dokku-mcp-keys", tenantID)
    if err := os.MkdirAll(keyDir, 0700); err != nil {
        return "", err
    }
    
    // Write key file with restrictive permissions
    keyPath := filepath.Join(keyDir, "id_rsa")
    if err := os.WriteFile(keyPath, []byte(privateKey), 0400); err != nil {
        return "", err
    }
    
    return keyPath, nil
}

// RotateSSHKey requests a new SSH key from Vault's dynamic secrets engine
func (v *VaultSecretProvider) RotateSSHKey(ctx context.Context, tenantID string) error {
    // Trigger Vault SSH secret engine to generate new key
    credsPath := fmt.Sprintf("ssh/creds/dokku-tenant-%s", tenantID)
    _, err := v.vaultClient.Logical().WriteWithContext(ctx, credsPath, map[string]interface{}{
        "ip": "dokku-host", // Vault SSH engine parameter
    })
    return err
}
```

---

### Application Layer: Tool Authorization Wrapper

```go
// internal/authentication/application/authorized_tool_handler.go

package application

import (
    "context"
    "fmt"
    
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/domain"
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/infrastructure"
    "github.com/mark3labs/mcp-go/mcp"
)

// AuthorizedToolHandler wraps MCP tool handlers with authorization checks
type AuthorizedToolHandler struct {
    authService    *domain.AuthenticationService
    wrappedHandler mcp.ToolHandlerFunc
    requiredPerm   domain.Permission
}

// NewAuthorizedToolHandler creates an authorization-checking tool handler
func NewAuthorizedToolHandler(
    authService *domain.AuthenticationService,
    handler mcp.ToolHandlerFunc,
    requiredPerm domain.Permission,
) *AuthorizedToolHandler {
    return &AuthorizedToolHandler{
        authService:    authService,
        wrappedHandler: handler,
        requiredPerm:   requiredPerm,
    }
}

// Handle implements mcp.ToolHandlerFunc with authorization
func (h *AuthorizedToolHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract tenant identity from context (injected by SSEContextFunc)
    identity, err := infrastructure.GetTenantIdentity(ctx)
    if err != nil {
        return &mcp.CallToolResult{
            Content: []interface{}{
                mcp.TextContent{
                    Type: "text",
                    Text: "❌ Unauthorized: " + err.Error(),
                },
            },
            IsError: boolPtr(true),
        }, nil // Return as MCP error, not Go error
    }
    
    // Check authorization for this action
    if err := h.authService.AuthorizeAction(identity, h.requiredPerm); err != nil {
        return &mcp.CallToolResult{
            Content: []interface{}{
                mcp.TextContent{
                    Type: "text",
                    Text: fmt.Sprintf("❌ Forbidden: %v", err),
                },
            },
            IsError: boolPtr(true),
        }, nil
    }
    
    // Authorization passed - call wrapped handler
    return h.wrappedHandler(ctx, request)
}

func boolPtr(b bool) *bool { return &b }
```

---

## 🔧 Integration with Existing Dokku-MCP Plugins

### Enhance Dokku Client to Use Tenant Context

```go
// internal/dokku-api/client.go - Enhanced with tenant context

func (c *SSHDokkuClient) Execute(ctx context.Context, cmd string, args ...string) (string, error) {
    // Extract tenant-specific SSH configuration from context
    if sshHost, ok := ctx.Value("dokku_ssh_host").(string); ok && sshHost != "" {
        c.config.SSH.Host = sshHost
    }
    if sshPort, ok := ctx.Value("dokku_ssh_port").(int); ok && sshPort > 0 {
        c.config.SSH.Port = sshPort
    }
    if sshUser, ok := ctx.Value("dokku_ssh_user").(string); ok && sshUser != "" {
        c.config.SSH.User = sshUser
    }
    if sshKeyPath, ok := ctx.Value("dokku_ssh_key_path").(string); ok && sshKeyPath != "" {
        c.config.SSH.KeyPath = sshKeyPath
    }
    
    // Continue with normal execution using tenant-specific config
    return c.executeSSHCommand(ctx, cmd, args...)
}
```

### Wrap Plugin Tools with Authorization

```go
// internal/server-plugins/app/plugin.go - Enhanced with authorization

func (p *ApplicationPlugin) GetTools(ctx context.Context) ([]*domain.Tool, error) {
    // Create authorized versions of each tool
    return []*domain.Tool{
        {
            Name: "create_app",
            Handler: application.NewAuthorizedToolHandler(
                p.authService,
                p.handleCreateApp,
                domain.PermissionAppsCreate,
            ).Handle,
        },
        {
            Name: "deploy_app",
            Handler: application.NewAuthorizedToolHandler(
                p.authService,
                p.handleDeployApp,
                domain.PermissionAppsDeploy,
            ).Handle,
        },
        {
            Name: "destroy_app",
            Handler: application.NewAuthorizedToolHandler(
                p.authService,
                p.handleDestroyApp,
                domain.PermissionAppsDestroy,
            ).Handle,
        },
        // ... other tools with appropriate permissions
    }, nil
}
```

---

## 📊 Configuration

### Enhanced Config Structure

```yaml
# config.yaml with authentication

transport:
  type: "sse"
  host: "0.0.0.0"
  port: 8080

# Authentication configuration
authentication:
  enabled: true
  jwt:
    secret_key: "${JWT_SECRET_KEY}"  # Load from environment
    issuer: "dokku-mcp-auth"
    audience: "dokku-api"
  
  vault:
    address: "https://vault.example.com:8200"
    mount_path: "secret"
    token: "${VAULT_TOKEN}"  # Load from environment or use Kubernetes auth
  
  # Default permissions for tenant tiers
  tenant_tiers:
    free:
      - "apps:read"
      - "apps:create"
      - "config:read"
    pro:
      - "apps:*"
      - "config:*"
      - "domains:manage"
    enterprise:
      - "*"  # All permissions

# SSH configuration (fallback for non-tenant mode)
ssh:
  host: "localhost"
  port: 3022
  user: "dokku"
  key_path: ""

# Logging with tenant context
log_level: "info"
log_format: "json"
```

---

## 🚀 Deployment Architecture

### Kubernetes Deployment with Vault Integration

```yaml
# k8s/deployment.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: dokku-mcp-sse
  namespace: dokku-mcp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: dokku-mcp
  template:
    metadata:
      labels:
        app: dokku-mcp
      annotations:
        vault.hashicorp.com/agent-inject: "true"
        vault.hashicorp.com/role: "dokku-mcp"
    spec:
      serviceAccountName: dokku-mcp
      
      containers:
      - name: dokku-mcp
        image: dokku-mcp:latest
        
        env:
        # Transport config
        - name: DOKKU_MCP_TRANSPORT_TYPE
          value: "sse"
        - name: DOKKU_MCP_TRANSPORT_HOST
          value: "0.0.0.0"
        - name: DOKKU_MCP_TRANSPORT_PORT
          value: "8080"
        
        # Authentication config
        - name: DOKKU_MCP_AUTHENTICATION_ENABLED
          value: "true"
        - name: DOKKU_MCP_AUTHENTICATION_JWT_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: dokku-mcp-auth
              key: jwt-secret
        - name: DOKKU_MCP_AUTHENTICATION_VAULT_ADDRESS
          value: "http://vault.vault.svc.cluster.local:8200"
        - name: DOKKU_MCP_AUTHENTICATION_VAULT_TOKEN
          valueFrom:
            secretKeyRef:
              name: dokku-mcp-vault
              key: token
        
        ports:
        - containerPort: 8080
          name: sse
        
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10

---
apiVersion: v1
kind: Service
metadata:
  name: dokku-mcp-sse
  namespace: dokku-mcp
spec:
  selector:
    app: dokku-mcp
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
    name: sse
  type: ClusterIP

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: dokku-mcp-ingress
  namespace: dokku-mcp
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - dokku-api.example.com
    secretName: dokku-mcp-tls
  rules:
  - host: dokku-api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: dokku-mcp-sse
            port:
              number: 8080
```

---

## 🎯 Benefits of This MCP-Native DDD Approach

| Aspect | Benefit |
|--------|---------|
| **MCP Protocol Compliance** | Uses built-in `WithSSEContextFunc` - no protocol violations |
| **Clean Architecture** | Authentication as separate bounded context with clear interfaces |
| **Zero External Dependencies** | No separate auth proxy - authentication integrated into dokku-mcp |
| **Testable** | Domain logic separated from infrastructure, easy to mock |
| **Type-Safe** | Leverages Go's type system for entities and value objects |
| **Context Propagation** | Tenant context flows naturally through Go context.Context |
| **Flexible** | Easy to swap Vault for other secret stores (interface-based) |
| **Observable** | Tenant-scoped logging at every layer |

---

## 📈 Implementation Roadmap

### Phase 1: Domain Layer (Week 1-2)
- [ ] Define authentication bounded context entities
- [ ] Implement domain services and interfaces
- [ ] Write comprehensive unit tests for domain logic

### Phase 2: Infrastructure Layer (Week 3-4)
- [ ] Implement SSEContextAuthenticator with `WithSSEContextFunc`
- [ ] Build VaultSecretProvider for tenant credentials
- [ ] Create PostgreSQL TenantRepository implementation

### Phase 3: Application Layer (Week 5)
- [ ] Implement AuthorizedToolHandler wrapper
- [ ] Add authorization checks to all plugin tools
- [ ] Enhance Dokku client with tenant context support

### Phase 4: Testing & Deployment (Week 6-8)
- [ ] Integration tests with mock authentication
- [ ] End-to-end tests with real Vault instance
- [ ] Deploy to staging Kubernetes cluster
- [ ] Load testing and performance optimization

---

## 🔍 Example: End-to-End Flow

```
1. Client requests OAuth token from auth server
   → Receives JWT with tenant_id and permissions

2. Client connects to SSE endpoint with token
   → EventSource('https://api.example.com/sse?token=JWT')

3. mcp-go calls SSEContextFunc (SSEContextAuthenticator.InjectTenantContext)
   → Extracts token from query param
   → Validates JWT signature
   → Retrieves tenant from repository
   → Gets tenant-specific Dokku SSH credentials from Vault
   → Injects TenantIdentity into context

4. Client calls MCP tool (e.g., "deploy_app")
   → Tool handler wrapped by AuthorizedToolHandler
   → Extracts TenantIdentity from context
   → Checks permission (AppsDeploy)
   → If authorized, calls original tool handler

5. Original tool handler executes
   → Dokku client extracts SSH config from context
   → Uses tenant-specific SSH credentials to connect
   → Executes dokku command on tenant's instance
   → Returns result

6. Result flows back through MCP protocol
   → Client receives deployment status
   → All logs tagged with tenant_id for observability
```

---

This architecture maintains the purity of the dokku-mcp core while adding enterprise-grade multi-tenant authentication using MCP-native patterns and clean DDD principles. The `WithSSEContextFunc` option is the key that makes this approach truly integrated rather than bolted-on.

