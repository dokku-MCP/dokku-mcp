# Development Playbook - Dokku MCP Server

## Development Workflow

### 1. Environment Setup
```bash
git clone <repo-url>
cd dokku-mcp
go mod download
make install-tools
make setup-hooks
```

### 2. Development Cycle

#### Create a New Feature
```bash
git checkout -b feature/feature-name
```

#### Testing and Validation
```bash
make lint
make fmt

# Run all tests
make test

make test-coverage
make test-integration
```

#### Documentation
```bash
# Generate documentation
make docs
make test-examples
```

## Development Patterns

### 1. Adding a New MCP Resource

#### Step 1: Define Domain Entity
```go
// internal/domain/application/entity.go
type Application struct {
    name     string
    state    ApplicationState
    config   *ApplicationConfig
    services []*Service
}

func NewApplication(name string) *Application {
    return &Application{
        name:  name,
        state: StateCreated,
    }
}
```

#### Step 2: Create Repository
```go
// internal/domain/application/repository.go
type Repository interface {
    GetAll(ctx context.Context) ([]*Application, error)
    GetByName(ctx context.Context, name string) (*Application, error)
    Save(ctx context.Context, app *Application) error
}
```

#### Step 3: Implement Infrastructure
```go
// internal/infrastructure/dokku/application_repository.go
type applicationRepository struct {
    client DokkuClient
}

func (r *applicationRepository) GetAll(ctx context.Context) ([]*Application, error) {
    // Implementation using Dokku client
}
```

#### Step 4: Create MCP Handler
```go
// internal/application/handlers/resource_handler.go
func (h *ResourceHandler) HandleApplications(ctx context.Context) ([]*mcp.Resource, error) {
    apps, err := h.appRepo.GetAll(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve applications: %w", err)
    }
    
    resources := make([]*mcp.Resource, len(apps))
    for i, app := range apps {
        resources[i] = &mcp.Resource{
            URI:         fmt.Sprintf("dokku://app/%s", app.Name()),
            Name:        app.Name(),
            Description: fmt.Sprintf("Dokku Application: %s", app.Name()),
            MimeType:    "application/json",
        }
    }
    
    return resources, nil
}
```

### 2. Adding a New MCP Tool

#### Step 1: Define the Tool
```go
// internal/application/tools/deploy_tool.go
type DeployTool struct {
    deployService domain.DeploymentService
}

func (t *DeployTool) Definition() *mcp.ToolDefinition {
    return &mcp.ToolDefinition{
        Name:        "deploy_application",
        Description: "Deploy a Dokku application",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "app_name": map[string]interface{}{
                    "type":        "string",
                    "description": "Name of the application to deploy",
                },
                "git_ref": map[string]interface{}{
                    "type":        "string",
                    "description": "Git reference to deploy (optional)",
                },
            },
            "required": []string{"app_name"},
        },
    }
}
```

#### Step 2: Implement Execution
```go
func (t *DeployTool) Execute(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
    // Parameter validation
    appName, ok := params["app_name"].(string)
    if !ok || appName == "" {
        return nil, errors.New("app_name parameter required")
    }
    
    // Execute deployment
    deployment, err := t.deployService.Deploy(ctx, appName, domain.DeployOptions{
        GitRef: getStringParam(params, "git_ref"),
    })
    if err != nil {
        return &mcp.ToolResult{
            Content: []map[string]interface{}{
                {
                    "type": "text",
                    "text": fmt.Sprintf("Deployment error: %v", err),
                },
            },
            IsError: true,
        }, nil
    }
    
    return &mcp.ToolResult{
        Content: []map[string]interface{}{
            {
                "type": "text",
                "text": fmt.Sprintf("Deployment successful: %s", deployment.ID),
            },
        },
    }, nil
}
```

### 3. Adding a Plugin

#### Plugin Structure
```go
// internal/plugins/database/plugin.go
type DatabasePlugin struct {
    config PluginConfig
    client DokkuClient
}

func (p *DatabasePlugin) Name() string {
    return "database"
}

func (p *DatabasePlugin) GetResources() []ResourceDefinition {
    return []ResourceDefinition{
        {
            Pattern:     "dokku://database/*",
            Handler:     p.handleDatabaseResource,
            Description: "Dokku database resources",
        },
    }
}

func (p *DatabasePlugin) GetTools() []ToolDefinition {
    return []ToolDefinition{
        {
            Name:    "create_database",
            Handler: p.createDatabase,
        },
        {
            Name:    "backup_database", 
            Handler: p.backupDatabase,
        },
    }
}
```

## Debugging and Troubleshooting

### Debug Logs
```bash
# Enable detailed logs
export DOKKU_MCP_LOG_LEVEL=debug
export DOKKU_MCP_LOG_FORMAT=json

# Run server in debug mode
./dokku-mcp --debug
```

### Diagnostic Tools
```bash
make test-mcp-resources
make profile
```

### Debugging with Delve
```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug server
dlv debug cmd/server/main.go

# Debug tests
dlv test ./internal/application/handlers
```

## Code Review and Quality

### Review Checklist
- [ ] Unit tests added/modified
- [ ] Documentation updated
- [ ] Proper error handling
- [ ] Input validation
- [ ] Logs added for important operations
- [ ] Performance verified (no N+1, timeouts)
- [ ] Security verified (validation, sanitization)

### Quality Metrics
```bash
# Cyclomatic complexity
gocyclo -over 20 ./...

# Duplicate code detection
dupl -threshold 50 ./...

# Security analysis
gosec ./...
```

## Deployment and Release

### Release Preparation
```bash
# Update version
make bump-version VERSION=v1.2.0

# Generate changelog
make changelog

# Create multi-platform builds
make build-all

# Complete regression tests
make test-regression
```

### Pre-Production Validation
```bash
# Integration tests on staging environment
make test-integration-staging

# Load tests
make load-test

# Security tests
make security-test
``` 