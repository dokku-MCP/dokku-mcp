# Authentication Implementation Guide

## Practical Integration with Existing Dokku-MCP Codebase

This guide shows **exactly how to integrate** the MCP-native authentication into the current dokku-mcp codebase following existing DDD patterns.

---

## 📁 New Directory Structure

```
internal/
├── authentication/              # NEW: Authentication bounded context
│   ├── domain/
│   │   ├── tenant_identity.go
│   │   ├── access_token.go
│   │   ├── permission.go
│   │   ├── authentication_service.go
│   │   ├── repository.go       # Interfaces
│   │   ├── errors.go
│   │   └── authentication_suite_test.go
│   ├── application/
│   │   ├── authenticate_connection_usecase.go
│   │   ├── authorized_tool_handler.go
│   │   └── provision_secrets_usecase.go
│   ├── infrastructure/
│   │   ├── sse_context_authenticator.go  # Key integration point
│   │   ├── vault_secret_provider.go
│   │   ├── postgres_tenant_repository.go
│   │   └── jwt_token_parser.go
│   └── module.go               # Fx module for DI
│
├── dokku-api/                  # EXISTING - enhance with context awareness
│   ├── client.go               # Modify to read tenant context
│   └── ...
│
├── server/                     # EXISTING - enhance with auth
│   ├── server.go               # Modify to inject SSEContextFunc
│   ├── module.go               # Add authentication module
│   └── ...
│
└── server-plugins/             # EXISTING - wrap with authorization
    ├── app/
    │   ├── plugin.go           # Wrap tools with AuthorizedToolHandler
    │   └── ...
    └── ...
```

---

## 🔧 Step-by-Step Implementation

### Step 1: Add Authentication Configuration

```go
// pkg/config/config.go - Add authentication config

type AuthenticationConfig struct {
    Enabled bool                  `mapstructure:"enabled"`
    JWT     JWTConfig            `mapstructure:"jwt"`
    Vault   VaultConfig          `mapstructure:"vault"`
}

type JWTConfig struct {
    SecretKey string `mapstructure:"secret_key"`
    Issuer    string `mapstructure:"issuer"`
    Audience  string `mapstructure:"audience"`
}

type VaultConfig struct {
    Address   string `mapstructure:"address"`
    Token     string `mapstructure:"token"`
    MountPath string `mapstructure:"mount_path"`
}

type ServerConfig struct {
    // ... existing fields ...
    
    Authentication AuthenticationConfig `mapstructure:"authentication"` // NEW
}

func DefaultConfig() *ServerConfig {
    return &ServerConfig{
        // ... existing defaults ...
        
        Authentication: AuthenticationConfig{
            Enabled: false, // Off by default for backward compatibility
            JWT: JWTConfig{
                Issuer:   "dokku-mcp",
                Audience: "dokku-api",
            },
        },
    }
}
```

### Step 2: Create Authentication Domain Module

```go
// internal/authentication/module.go

package authentication

import (
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/application"
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/domain"
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/infrastructure"
    "github.com/dokku-mcp/dokku-mcp/pkg/config"
    "go.uber.org/fx"
)

var Module = fx.Module("authentication",
    // Domain services
    fx.Provide(
        domain.NewAuthenticationService,
    ),
    
    // Infrastructure implementations
    fx.Provide(
        // Annotate to implement interfaces
        fx.Annotate(
            infrastructure.NewVaultSecretProvider,
            fx.As(new(domain.SecretProvider)),
        ),
        fx.Annotate(
            infrastructure.NewPostgresTenantRepository,
            fx.As(new(domain.TenantRepository)),
        ),
        infrastructure.NewJWTTokenParser,
        infrastructure.NewSSEContextAuthenticator,
    ),
    
    // Application use cases
    fx.Provide(
        application.NewAuthenticateConnectionUseCase,
    ),
    
    // Conditional registration based on config
    fx.Invoke(func(cfg *config.ServerConfig) {
        if cfg.Authentication.Enabled {
            // Authentication is enabled
        }
    }),
)
```

### Step 3: Modify Server Initialization

```go
// internal/server/server.go - Enhanced with authentication

package server

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "time"

    authDomain "github.com/dokku-mcp/dokku-mcp/internal/authentication/domain"
    authInfra "github.com/dokku-mcp/dokku-mcp/internal/authentication/infrastructure"
    plugins "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/application"
    "github.com/dokku-mcp/dokku-mcp/pkg/config"
    "github.com/mark3labs/mcp-go/server"
    "go.uber.org/fx"
)

// Enhanced hook registration with optional authentication
func registerServerHooks(
    lc fx.Lifecycle,
    cfg *config.ServerConfig,
    mcpServer *server.MCPServer,
    adapter *MCPAdapter,
    dynamicRegistry *plugins.DynamicServerPluginRegistry,
    logger *slog.Logger,
    // Optional authentication components (only injected if enabled)
    authService *authDomain.AuthenticationService,
    contextAuthenticator *authInfra.SSEContextAuthenticator,
) {
    var httpServer *http.Server

    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            logger.Info("Performing initial plugin synchronization...")

            if err := dynamicRegistry.SyncServerPlugins(ctx); err != nil {
                logger.Error("Initial plugin sync failed", "error", err)
            }

            logger.Info("Registering all server plugins...")
            if err := adapter.RegisterAllServerPlugins(ctx); err != nil {
                return fmt.Errorf("failed to register server plugins: %w", err)
            }

            switch cfg.Transport.Type {
            case "sse":
                logger.Info("Starting MCP server with 'sse' transport",
                    "authentication_enabled", cfg.Authentication.Enabled)
                
                // Build SSE server options
                sseOptions := []server.SSEOption{
                    server.WithSSEEndpoint("/sse"),
                    server.WithMessageEndpoint("/message"),
                    server.WithKeepAlive(true),
                    server.WithKeepAliveInterval(30 * time.Second),
                }
                
                // Add authentication if enabled
                if cfg.Authentication.Enabled && contextAuthenticator != nil {
                    logger.Info("SSE authentication is enabled")
                    sseOptions = append(sseOptions,
                        server.WithSSEContextFunc(contextAuthenticator.InjectTenantContext),
                    )
                } else {
                    logger.Warn("SSE authentication is DISABLED - running in single-tenant mode")
                }
                
                sseServer := server.NewSSEServer(mcpServer, sseOptions...)
                
                go func() {
                    addr := fmt.Sprintf("%s:%d", cfg.Transport.Host, cfg.Transport.Port)
                    logger.Info("SSE server listening", "address", addr)
                    if err := sseServer.Start(addr); err != nil && err != http.ErrServerClosed {
                        logger.Error("SSE server failed", "error", err)
                    }
                }()
                
            case "stdio":
                // stdio mode - no authentication (local use)
                logger.Info("Starting MCP server with 'stdio' transport.")
                go func() {
                    if err := server.ServeStdio(mcpServer); err != nil {
                        logger.Error("Stdio server failed", "error", err)
                    }
                }()
                
            default:
                return fmt.Errorf("unknown transport type: %s", cfg.Transport.Type)
            }
            return nil
        },
        OnStop: func(ctx context.Context) error {
            if httpServer != nil {
                logger.Info("Shutting down SSE server gracefully...")
                shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
                defer cancel()
                return httpServer.Shutdown(shutdownCtx)
            }
            logger.Info("Stdio server shutdown.")
            return nil
        },
    })
}
```

### Step 4: Implement SSE Context Authenticator (Key Integration Point)

```go
// internal/authentication/infrastructure/sse_context_authenticator.go

package infrastructure

import (
    "context"
    "log/slog"
    "net/http"
    
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/domain"
    "github.com/dokku-mcp/dokku-mcp/pkg/config"
)

// SSEContextAuthenticator implements mcp-go's SSEContextFunc
type SSEContextAuthenticator struct {
    authService  *domain.AuthenticationService
    tokenParser  *JWTTokenParser
    logger       *slog.Logger
    cfg          *config.ServerConfig
}

func NewSSEContextAuthenticator(
    authService *domain.AuthenticationService,
    tokenParser *JWTTokenParser,
    cfg *config.ServerConfig,
    logger *slog.Logger,
) *SSEContextAuthenticator {
    return &SSEContextAuthenticator{
        authService: authService,
        tokenParser: tokenParser,
        cfg:         cfg,
        logger:      logger,
    }
}

// InjectTenantContext implements server.SSEContextFunc
// This is called by mcp-go for EVERY SSE connection request
func (a *SSEContextAuthenticator) InjectTenantContext(
    ctx context.Context,
    r *http.Request,
) context.Context {
    // Extract token from query parameter (SSE limitation)
    tokenStr := r.URL.Query().Get("token")
    
    if tokenStr == "" {
        a.logger.Warn("SSE connection without token",
            "remote_addr", r.RemoteAddr,
            "user_agent", r.UserAgent())
        
        // Inject error so handlers can reject
        return contextWithError(ctx, domain.ErrUnauthorized)
    }
    
    // Parse and validate JWT
    accessToken, err := a.tokenParser.Parse(tokenStr)
    if err != nil {
        a.logger.Error("Failed to parse token",
            "error", err,
            "remote_addr", r.RemoteAddr)
        return contextWithError(ctx, err)
    }
    
    // Authenticate and get tenant identity
    identity, err := a.authService.AuthenticateToken(ctx, accessToken)
    if err != nil {
        a.logger.Error("Authentication failed",
            "error", err,
            "tenant_id", accessToken.TenantID(),
            "remote_addr", r.RemoteAddr)
        return contextWithError(ctx, err)
    }
    
    // Successfully authenticated!
    a.logger.Info("SSE connection authenticated",
        "tenant_id", identity.TenantID(),
        "client_id", identity.ClientID(),
        "remote_addr", r.RemoteAddr)
    
    // Inject tenant identity into context
    ctx = contextWithTenantIdentity(ctx, identity)
    
    // Inject Dokku configuration for this tenant
    dokkuConfig := identity.DokkuConfig()
    ctx = contextWithDokkuConfig(ctx, dokkuConfig)
    
    // Create tenant-scoped logger
    tenantLogger := a.logger.With(
        "tenant_id", identity.TenantID(),
        "client_id", identity.ClientID(),
    )
    ctx = contextWithLogger(ctx, tenantLogger)
    
    return ctx
}

// Context helper functions
type contextKey string

const (
    keyAuthError      contextKey = "auth_error"
    keyTenantIdentity contextKey = "tenant_identity"
    keyDokkuConfig    contextKey = "dokku_config"
    keyLogger         contextKey = "logger"
)

func contextWithError(ctx context.Context, err error) context.Context {
    return context.WithValue(ctx, keyAuthError, err)
}

func contextWithTenantIdentity(ctx context.Context, identity *domain.TenantIdentity) context.Context {
    return context.WithValue(ctx, keyTenantIdentity, identity)
}

func contextWithDokkuConfig(ctx context.Context, config domain.DokkuConnectionConfig) context.Context {
    return context.WithValue(ctx, keyDokkuConfig, config)
}

func contextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
    return context.WithValue(ctx, keyLogger, logger)
}

// GetTenantIdentity extracts tenant identity from context
func GetTenantIdentity(ctx context.Context) (*domain.TenantIdentity, error) {
    // Check for authentication error first
    if err, ok := ctx.Value(keyAuthError).(error); ok {
        return nil, err
    }
    
    identity, ok := ctx.Value(keyTenantIdentity).(*domain.TenantIdentity)
    if !ok || identity == nil {
        return nil, domain.ErrUnauthorized
    }
    
    return identity, nil
}

// GetDokkuConfig extracts Dokku config from context
func GetDokkuConfig(ctx context.Context) (domain.DokkuConnectionConfig, bool) {
    config, ok := ctx.Value(keyDokkuConfig).(domain.DokkuConnectionConfig)
    return config, ok
}

// GetLogger extracts tenant-scoped logger from context
func GetLogger(ctx context.Context) *slog.Logger {
    if logger, ok := ctx.Value(keyLogger).(*slog.Logger); ok {
        return logger
    }
    return slog.Default()
}
```

### Step 5: Enhance Dokku Client with Context Awareness

```go
// internal/dokku-api/client.go - Modify Execute method

func (c *SSHDokkuClient) Execute(ctx context.Context, cmd string, args ...string) (string, error) {
    // Check if we have tenant-specific configuration in context
    if config, ok := authInfra.GetDokkuConfig(ctx); ok {
        // Override client SSH config with tenant-specific values
        c.logger.Debug("Using tenant-specific Dokku configuration",
            "ssh_host", config.Host,
            "ssh_user", config.User)
        
        c.config.SSH.Host = config.Host
        c.config.SSH.Port = config.Port
        c.config.SSH.User = config.User
        c.config.SSH.KeyPath = config.SSHKeyPath
    }
    
    // Continue with existing execution logic
    return c.executeSSHCommand(ctx, cmd, args...)
}
```

### Step 6: Wrap Plugin Tools with Authorization

```go
// internal/server-plugins/app/plugin.go - Enhanced with authorization

package app

import (
    "context"
    
    authApp "github.com/dokku-mcp/dokku-mcp/internal/authentication/application"
    authDomain "github.com/dokku-mcp/dokku-mcp/internal/authentication/domain"
    "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/application"
    "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/domain"
    pluginDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
)

type ApplicationPlugin struct {
    // ... existing fields ...
    
    authService *authDomain.AuthenticationService // NEW: Optional auth service
}

// NewApplicationPlugin with optional authentication
func NewApplicationPlugin(
    useCase *application.ApplicationUseCase,
    logger *slog.Logger,
    authService *authDomain.AuthenticationService, // Optional - nil if auth disabled
) *ApplicationPlugin {
    return &ApplicationPlugin{
        useCase:     useCase,
        logger:      logger,
        authService: authService,
    }
}

func (p *ApplicationPlugin) GetTools(ctx context.Context) ([]*pluginDomain.Tool, error) {
    // Define tools
    tools := []*pluginDomain.Tool{
        {
            Name:        "create_app",
            Description: "Create a new Dokku application",
            InputSchema: createAppInputSchema(),
            Handler:     p.handleCreateApp,
        },
        {
            Name:        "deploy_app",
            Description: "Deploy an application from Git repository",
            InputSchema: deployAppInputSchema(),
            Handler:     p.handleDeployApp,
        },
        {
            Name:        "destroy_app",
            Description: "Destroy a Dokku application",
            InputSchema: destroyAppInputSchema(),
            Handler:     p.handleDestroyApp,
        },
        // ... more tools
    }
    
    // If authentication is enabled, wrap tools with authorization
    if p.authService != nil {
        tools = p.wrapToolsWithAuthorization(tools)
    }
    
    return tools, nil
}

// wrapToolsWithAuthorization wraps each tool handler with auth checks
func (p *ApplicationPlugin) wrapToolsWithAuthorization(
    tools []*pluginDomain.Tool,
) []*pluginDomain.Tool {
    // Define required permissions for each tool
    permissions := map[string]authDomain.Permission{
        "create_app":  authDomain.PermissionAppsCreate,
        "deploy_app":  authDomain.PermissionAppsDeploy,
        "destroy_app": authDomain.PermissionAppsDestroy,
        "scale_app":   authDomain.PermissionAppsScale,
        // ... more mappings
    }
    
    for i, tool := range tools {
        requiredPerm, ok := permissions[tool.Name]
        if !ok {
            // Default to apps:read for tools without explicit permission
            requiredPerm = authDomain.PermissionAppsRead
        }
        
        // Wrap original handler
        originalHandler := tool.Handler
        tools[i].Handler = authApp.NewAuthorizedToolHandler(
            p.authService,
            originalHandler,
            requiredPerm,
        ).Handle
    }
    
    return tools
}
```

### Step 7: Update Main Application Module

```go
// pkg/fxapp/app.go - Wire everything together

package fxapp

import (
    "github.com/dokku-mcp/dokku-mcp/internal/authentication"
    "github.com/dokku-mcp/dokku-mcp/internal/server"
    "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app"
    // ... other imports
    "github.com/dokku-mcp/dokku-mcp/pkg/config"
    "github.com/dokku-mcp/dokku-mcp/pkg/logger"
    "go.uber.org/fx"
)

func New() *fx.App {
    return fx.New(
        // Core modules
        config.Module,
        logger.Module,
        
        // Authentication module (NEW)
        // Only registers components if authentication.enabled = true
        authentication.Module,
        
        // Server and plugins
        server.Module,
        app.Module,
        // ... other plugin modules
        
        fx.NopLogger, // Silence fx startup logs
    )
}
```

---

## 🧪 Testing Strategy

### Unit Tests for Authentication Domain

```go
// internal/authentication/domain/authentication_service_test.go

package domain_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/domain"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestAuthenticationDomain(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Authentication Domain Suite")
}

var _ = Describe("AuthenticationService", func() {
    var (
        authService    *domain.AuthenticationService
        mockTenantRepo *mockTenantRepository
        mockSecretProv *mockSecretProvider
    )
    
    BeforeEach(func() {
        mockTenantRepo = newMockTenantRepository()
        mockSecretProv = newMockSecretProvider()
        authService = domain.NewAuthenticationService(mockTenantRepo, mockSecretProv)
    })
    
    Context("AuthenticateToken", func() {
        It("should authenticate valid token and return tenant identity", func() {
            // Given
            validToken := "valid-jwt-token"
            mockTenantRepo.AddTenant("tenant-123", "Test Tenant", true)
            mockSecretProv.AddConfig("tenant-123", domain.DokkuConnectionConfig{
                Host: "dokku.example.com",
                User: "dokku",
            })
            
            // When
            identity, err := authService.AuthenticateToken(context.Background(), validToken)
            
            // Then
            Expect(err).NotTo(HaveOccurred())
            Expect(identity).NotTo(BeNil())
            Expect(identity.TenantID()).To(Equal("tenant-123"))
        })
        
        It("should reject expired token", func() {
            // Given
            expiredToken := "expired-jwt-token"
            
            // When
            identity, err := authService.AuthenticateToken(context.Background(), expiredToken)
            
            // Then
            Expect(err).To(MatchError(domain.ErrTokenExpired))
            Expect(identity).To(BeNil())
        })
        
        It("should reject inactive tenant", func() {
            // Given
            validToken := "valid-jwt-token"
            mockTenantRepo.AddTenant("tenant-123", "Inactive Tenant", false)
            
            // When
            identity, err := authService.AuthenticateToken(context.Background(), validToken)
            
            // Then
            Expect(err).To(MatchError(domain.ErrTenantInactive))
            Expect(identity).To(BeNil())
        })
    })
})
```

### Integration Test with SSE

```go
// internal/authentication/infrastructure/sse_context_authenticator_integration_test.go

package infrastructure_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/dokku-mcp/dokku-mcp/internal/authentication/infrastructure"
    "github.com/mark3labs/mcp-go/server"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("SSEContextAuthenticator Integration", func() {
    var (
        authenticator *infrastructure.SSEContextAuthenticator
        mcpServer     *server.MCPServer
        sseServer     *server.SSEServer
        testServer    *httptest.Server
    )
    
    BeforeEach(func() {
        // Set up test MCP server with authentication
        mcpServer = server.NewMCPServer("test", "1.0.0")
        authenticator = infrastructure.NewSSEContextAuthenticator(
            testAuthService(),
            testTokenParser(),
            testConfig(),
            testLogger(),
        )
        
        sseServer = server.NewSSEServer(
            mcpServer,
            server.WithSSEContextFunc(authenticator.InjectTenantContext),
        )
        
        testServer = httptest.NewServer(sseServer)
    })
    
    AfterEach(func() {
        testServer.Close()
    })
    
    It("should authenticate SSE connection with valid token", func() {
        // Given
        validToken := generateTestJWT("tenant-123")
        
        // When
        resp, err := http.Get(testServer.URL + "/sse?token=" + validToken)
        
        // Then
        Expect(err).NotTo(HaveOccurred())
        Expect(resp.StatusCode).To(Equal(http.StatusOK))
        Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("text/event-stream"))
    })
    
    It("should reject SSE connection without token", func() {
        // When
        resp, err := http.Get(testServer.URL + "/sse")
        
        // Then
        Expect(err).NotTo(HaveOccurred())
        Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
    })
})
```

---

## 🚀 Deployment Configuration

### config.yaml for Multi-Tenant Mode

```yaml
# config.yaml - Production multi-tenant configuration

transport:
  type: "sse"
  host: "0.0.0.0"
  port: 8080

# Enable authentication
authentication:
  enabled: true
  jwt:
    secret_key: "${JWT_SECRET_KEY}"  # From environment variable
    issuer: "dokku-mcp-prod"
    audience: "dokku-api"
  vault:
    address: "https://vault.internal:8200"
    token: "${VAULT_TOKEN}"
    mount_path: "secret"

# Logging with tenant context
log_level: "info"
log_format: "json"

# Disable server logs tool in production
expose_server_logs: false

# Cache configuration
cache_enabled: true
cache_ttl: "5m"
```

### Docker Compose for Local Development

```yaml
# docker-compose.yml - Local multi-tenant development

version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: dokku_mcp
      POSTGRES_USER: dokku_mcp
      POSTGRES_PASSWORD: dev_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
  
  vault:
    image: hashicorp/vault:latest
    cap_add:
      - IPC_LOCK
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: dev_token
      VAULT_DEV_LISTEN_ADDRESS: "0.0.0.0:8200"
    ports:
      - "8200:8200"
  
  dokku-mcp:
    build: .
    ports:
      - "8080:8080"
    environment:
      DOKKU_MCP_TRANSPORT_TYPE: sse
      DOKKU_MCP_AUTHENTICATION_ENABLED: "true"
      DOKKU_MCP_AUTHENTICATION_JWT_SECRET_KEY: dev_secret_key_change_in_prod
      DOKKU_MCP_AUTHENTICATION_VAULT_ADDRESS: http://vault:8200
      DOKKU_MCP_AUTHENTICATION_VAULT_TOKEN: dev_token
      DOKKU_MCP_LOG_LEVEL: debug
    depends_on:
      - postgres
      - vault
    volumes:
      - ./config.yaml:/app/config.yaml

volumes:
  postgres_data:
```

---

## ✅ Migration Path

### Phase 1: Add Authentication (No Breaking Changes)

1. **Add authentication module** - optional, off by default
2. **Test in parallel** - existing stdio mode still works
3. **Gradual rollout** - enable for SSE only

### Phase 2: Enable for New Deployments

1. **New instances** start with authentication enabled
2. **Existing instances** continue without authentication
3. **Documentation** updated with migration guide

### Phase 3: Full Migration (Optional)

1. **Deprecation notice** for non-authenticated SSE
2. **6-month window** for migration
3. **Force enable** authentication in major version bump

---

This implementation guide provides a **practical, step-by-step path** to integrate MCP-native authentication into the existing dokku-mcp codebase while maintaining backward compatibility and following DDD principles.

