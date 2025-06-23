# Advanced Development Playbook

This playbook provides concrete patterns for extending the Dokku MCP server with new capabilities. For general contribution guidelines, please see [CONTRIBUTING.md](../../CONTRIBUTING.md).

## Development Patterns

### 1. Adding a New MCP Resource

This pattern shows how to expose a new type of Dokku resource to the MCP server.

#### Step 1: Define Domain Entity
Start by defining the core entity in the appropriate domain.

```go
// internal/server-plugins/app/domain/application.entity.go
type Application struct {
    name     string
    state    ApplicationState
    // ... other fields
}

func NewApplication(name string) *Application {
    // ... constructor logic
}
```

#### Step 2: Create Repository Interface
Define the contract for data access in the domain layer.

```go
// internal/server-plugins/app/domain/application.repository.go
type Repository interface {
    GetAll(ctx context.Context) ([]*Application, error)
    GetByName(ctx context.Context, name string) (*Application, error)
    Save(ctx context.Context, app *Application) error
}
```

#### Step 3: Implement Infrastructure
Implement the repository interface in the infrastructure layer, interacting with the Dokku API.

```go
// internal/server-plugins/app/infrastructure/application_repository.go
type applicationRepository struct {
    client dokkuApi.DokkuClient
}

func (r *applicationRepository) GetAll(ctx context.Context) ([]*Application, error) {
    // Implementation using Dokku client
}
```

#### Step 4: Create MCP Handler in Plugin
In the application layer of your plugin, create the handler that fetches domain entities and transforms them into MCP resources.

```go
// internal/server-plugins/app/application/application_usecase.go
func (uc *ApplicationUsecase) GetAllResources(ctx context.Context) ([]*mcp.Resource, error) {
    apps, err := uc.appRepo.GetAll(ctx)
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

This pattern shows how to add a new tool that can be executed by an LLM.

#### Step 1: Define the Tool Definition
In your plugin's application layer, define the tool's name, description, and input schema.

```go
// From your plugin's tool provider implementation
func (p *AppPlugin) GetTools(uc *ApplicationUsecase) []*mcp.Tool {
	return []*mcp.Tool{
		mcp.NewTool(
			"deploy_application",
			mcp.WithDescription("Deploy a Dokku application from a Git repository"),
			mcp.WithInputSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"app_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the application to deploy",
					},
					"git_ref": map[string]interface{}{
						"type":        "string",
						"description": "Git reference to deploy (e.g., 'main' or a commit SHA)",
						"default":     "main",
					},
				},
				"required": []string{"app_name"},
			}),
			mcp.WithExecute(uc.DeployApplication),
		),
	}
}
```

#### Step 2: Implement Tool Execution Logic
Implement the function that contains the business logic for the tool. This function will be wired to the tool definition.

```go
// internal/server-plugins/app/application/application_usecase.go
func (uc *ApplicationUsecase) DeployApplication(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
    // 1. Validate and extract parameters
    appName, ok := params["app_name"].(string)
    if !ok || appName == "" {
        return nil, errors.New("app_name parameter is required and must be a string")
    }
    gitRef := "main"
    if ref, ok := params["git_ref"].(string); ok {
		gitRef = ref
	}

    // 2. Execute business logic through domain services
    deployment, err := uc.deploymentService.Deploy(ctx, appName, domain.DeployOptions{
        GitRef: gitRef,
    })
    if err != nil {
        return &mcp.ToolResult{
            Content: fmt.Sprintf("Deployment failed: %v", err),
            IsError: true,
        }, nil
    }
    
    // 3. Return a structured, user-friendly result
    return &mcp.ToolResult{
        Content: fmt.Sprintf("✅ Deployment successful for app '%s'. Deployment ID: %s", appName, deployment.ID),
    }, nil
}
```

### 3. Adding a New Plugin

Refer to the `plugin-development-guide.md` for a comprehensive guide on creating new plugins from scratch.

## Debugging and Troubleshooting

### Enable Debug Logs

You can increase log verbosity for debugging by setting environment variables:

```bash
# Enable detailed logs and JSON format for structured logging
export DOKKU_MCP_LOG_LEVEL=debug
export DOKKU_MCP_LOG_FORMAT=json

# Run the server
make start
```

### Debugging with Delve

You can use [Delve](httpss://github.com/go-delve/delve), the Go debugger, for step-by-step debugging.

```bash
# Install Delve
make install-tools

# Debug the main server application
dlv debug ./cmd/server/main.go

# Debug a specific test package
dlv test ./internal/server-plugins/app/application
```

## Deployment and Release

### Release Preparation

The `Makefile` contains helpers for the release process.

```bash
# Update version in the source code
make bump-version VERSION=v1.2.0

# Generate a changelog from git history
make changelog

# Create multi-platform builds
make build-all
```
