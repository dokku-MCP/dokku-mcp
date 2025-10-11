# Plugin Development Guide

This guide shows developers how to create powerful plugins for the Dokku MCP Server. The server supports a **unified plugin architecture** with ServerPlugin (custom) as the primary interface.

## Plugin Architecture

- **Interface Segregation**: Plugins only implement the capabilities they provide
- **Better Separation of Concerns**: Plugin metadata separate from registration logic  
- **Domain-Driven Types**: Uses domain types instead of MCP-specific types
- **Dynamic Lifecycle Management**: Handled by registry and adapters

## Creating a New Plugin (ServerPlugin Architecture)

### Step 1: Plugin Structure

Create a new plugin by implementing the ServerPlugin interface:

```go
package myfeature

import (
    "context"
    "log/slog"
    
    "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
    "github.com/mark3labs/mcp-go/mcp"
)

type MyFeatureServerPlugin struct {
    logger *slog.Logger
    // Add your dependencies here
}

func NewMyFeatureServerPlugin(logger *slog.Logger) domain.ServerPlugin {
    return &MyFeatureServerPlugin{
        logger: logger,
    }
}

// ServerPlugin interface implementation
func (p *MyFeatureServerPlugin) ID() string {
    return "my-feature"
}

func (p *MyFeatureServerPlugin) Name() string {
    return "My Feature"
}

func (p *MyFeatureServerPlugin) Description() string {
    return "Comprehensive feature management for Dokku"
}

func (p *MyFeatureServerPlugin) Version() string {
    return "0.1.0"
}

func (p *MyFeatureServerPlugin) DokkuPluginName() string {
    return "my-dokku-plugin" // The corresponding Dokku plugin name, or "" for core functionality
}
```

### Step 2: Implement Capability Interfaces

Implement only the capability interfaces your plugin needs:

#### ResourceProvider (Optional)
```go
func (p *MyFeatureServerPlugin) GetResources(ctx context.Context) ([]domain.Resource, error) {
    return []domain.Resource{
        {
            URI:         "dokku://my-feature/data",
            Name:        "My Feature Data",
            Description: "Data provided by my feature",
            MIMEType:    "application/json",
            Handler:     p.handleDataResource,
        },
        {
            URI:         "dokku://my-feature/logs",
            Name:        "My Feature Logs", 
            Description: "Logs from my feature",
            MIMEType:    "text/plain",
            Handler:     p.handleLogsResource,
        },
    }, nil
}

func (p *MyFeatureServerPlugin) handleDataResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
    // Implement resource logic
    data := map[string]interface{}{
        "status": "active",
        "count":  42,
    }
    
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("failed to serialize data: %w", err)
    }
    
    return []mcp.ResourceContents{
        mcp.TextResourceContents{
            URI:      req.Params.URI,
            MIMEType: "application/json",
            Text:     string(jsonData),
        },
    }, nil
}
```

#### ToolProvider (Optional)
```go
func (p *MyFeatureServerPlugin) GetTools(ctx context.Context) ([]domain.Tool, error) {
    return []domain.Tool{
        {
            Name:        "create_something",
            Description: "Create something in my feature",
            Builder:     p.buildCreateSomethingTool,
            Handler:     p.handleCreateSomething,
        },
        {
            Name:        "configure_feature",
            Description: "Configure my feature settings",
            Builder:     p.buildConfigureFeatureTool,
            Handler:     p.handleConfigureFeature,
        },
    }, nil
}

func (p *MyFeatureServerPlugin) buildCreateSomethingTool() mcp.Tool {
    return mcp.NewTool(
        "create_something",
        mcp.WithDescription("Create something new"),
        mcp.WithString("name",
            mcp.Required(),
            mcp.Description("Name of the thing to create"),
            mcp.Pattern("^[a-z0-9-]+$"),
        ),
        mcp.WithString("type",
            mcp.Description("Type of thing to create"),
        ),
        mcp.WithBoolean("enable_feature",
            mcp.Description("Enable special feature"),
        ),
    )
}

func (p *MyFeatureServerPlugin) handleCreateSomething(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    name, err := req.RequireString("name")
    if err != nil {
        return mcp.NewToolResultError("Name parameter is required"), nil
    }
    
    // Validate input
    if len(name) < 3 {
        return mcp.NewToolResultError("Name must be at least 3 characters"), nil
    }
    
    // Perform the action
    // ... your business logic here ...
    
    return mcp.NewToolResultText(fmt.Sprintf("✅ Successfully created '%s'", name)), nil
}
```

#### PromptProvider (Optional)
```go
func (p *MyFeatureServerPlugin) GetPrompts(ctx context.Context) ([]domain.Prompt, error) {
    return []domain.Prompt{
        {
            Name:        "feature_advisor",
            Description: "Get advice on using my feature",
            Builder:     p.buildFeatureAdvisorPrompt,
            Handler:     p.handleFeatureAdvisorPrompt,
        },
        {
            Name:        "troubleshoot_feature", 
            Description: "Help troubleshoot feature issues",
            Builder:     p.buildTroubleshootPrompt,
            Handler:     p.handleTroubleshootPrompt,
        },
    }, nil
}

func (p *MyFeatureServerPlugin) buildFeatureAdvisorPrompt() mcp.Prompt {
    return mcp.NewPrompt(
        "feature_advisor",
        mcp.WithPromptDescription("Get personalized advice for using my feature"),
        mcp.WithArgument("use_case",
            mcp.RequiredArgument(),
            mcp.ArgumentDescription("Describe your use case"),
        ),
        mcp.WithArgument("experience_level",
            mcp.ArgumentDescription("Your experience level: beginner, intermediate, advanced"),
        ),
    )
}

func (p *MyFeatureServerPlugin) handleFeatureAdvisorPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    // Extract required argument from the request
    useCase, ok := req.Params.Arguments["use_case"]
    if !ok || useCase == "" {
        return &mcp.GetPromptResult{
            Description: "use_case parameter is required",
        }, fmt.Errorf("use_case parameter is required")
    }
    
    // Extract optional argument
    experienceLevel := ""
    if level, ok := req.Params.Arguments["experience_level"]; ok {
        experienceLevel = level
    }
    
    promptText := fmt.Sprintf("I need advice on using my feature for: %s", useCase)
    if experienceLevel != "" {
        promptText += fmt.Sprintf("\nMy experience level: %s", experienceLevel)
    }
    
    promptText += `

Please provide:
1. Best practices for this use case
2. Step-by-step implementation guide
3. Common pitfalls to avoid
4. Performance optimization tips
5. Security considerations

Current system data will be analyzed to provide personalized recommendations.`

    return &mcp.GetPromptResult{
        Description: "Personalized feature advice",
        Messages: []mcp.PromptMessage{
            {
                Role: "user",
                Content: mcp.TextContent{
                    Type: "text",
                    Text: promptText,
                },
            },
        },
    }, nil
}
```

## Plugin Registration

### Modern Registration (Automatic)

The unified architecture handles registration automatically through the DynamicServerPluginRegistry:

```go
// In your module file (e.g., internal/myfeature/fx_module.go)
package myfeature

import (
    "go.uber.org/fx"
    "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
)

var Module = fx.Module("myfeature",
    fx.Provide(
        fx.Annotate(
            NewMyFeatureServerPlugin,
            fx.As(new(domain.ServerPlugin)),
            fx.ResultTags(`group:"server_plugins"`),
        ),
        // Add other dependencies
    ),
)
```

The system will:
1. **Automatically discover** your plugin
2. **Register it** with the plugin registry
3. **Activate it** when the corresponding Dokku plugin is enabled (if DokkuPluginName() is not empty)
4. **Manage its lifecycle** dynamically

## Best Practices

### 1. Interface Segregation
Only implement the capability interfaces you need:
- **ResourceProvider**: For read-only data access
- **ToolProvider**: For actions that modify state  
- **PromptProvider**: For prompts generation

### 2. Domain-Driven Design
Structure your plugin following DDD principles:
```
internal/myfeature/
├── domain/               # Core business logic
├── application/          # Use case orchestration
├── infrastructure/       # External integrations
└── server-plugin.go      # ServerPlugin implementation
```

### 3. Error Handling
Provide clear, actionable error messages:
```go
func (p *MyFeatureServerPlugin) handleCreateSomething(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    name, err := req.RequireString("name")
    if err != nil {
        return mcp.NewToolResultError("Name parameter is required"), nil
    }
    
    // Business validation
    if exists := p.checkIfExists(ctx, name); exists {
        return mcp.NewToolResultError(fmt.Sprintf("'%s' already exists", name)), nil
    }
    
    // ... implementation
}
```

### 4. Security Validation
Always validate and sanitize inputs:
```go
func (p *MyFeatureServerPlugin) handleConfigureFeature(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Validate configuration
    args := req.GetArguments()
    configParam, ok := args["config"]
    if !ok {
        return mcp.NewToolResultError("Configuration object is required"), nil
    }
    
    configMap, ok := configParam.(map[string]interface{})
    if !ok {
        return mcp.NewToolResultError("Configuration must be an object"), nil
    }
    
    // Validate each key
    validatedConfig := make(map[string]string)
    for key, value := range configMap {
        if !isValidConfigKey(key) {
            return mcp.NewToolResultError(fmt.Sprintf("Invalid configuration key: %s", key)), nil
        }
        
        strValue, ok := value.(string)
        if !ok {
            return mcp.NewToolResultError(fmt.Sprintf("Configuration value for '%s' must be a string", key)), nil
        }
        
        validatedConfig[key] = sanitizeConfigValue(strValue)
    }
    
    // Apply configuration
    if err := p.applyConfig(ctx, validatedConfig); err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to apply configuration: %v", err)), nil
    }
    
    return mcp.NewToolResultText(fmt.Sprintf("✅ Configuration applied (%d settings)", len(validatedConfig))), nil
}
```

## Testing

### Unit Testing
```go
func TestMyFeatureServerPlugin_CreateSomething(t *testing.T) {
    tests := []struct {
        name        string
        request     mcp.CallToolRequest
        expectError bool
        expectMsg   string
    }{
        {
            name: "valid creation",
            request: mockCallToolRequest(map[string]interface{}{
                "name": "test-item",
                "type": "standard",
            }),
            expectError: false,
            expectMsg:   "✅ Successfully created 'test-item'",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            plugin := NewMyFeatureServerPlugin(slog.Default())
            
            // Cast to ToolProvider
            toolProvider, ok := plugin.(domain.ToolProvider)
            require.True(t, ok)
            
            tools, err := toolProvider.GetTools(context.Background())
            require.NoError(t, err)
            
            // Find the create tool
            var createTool domain.Tool
            for _, tool := range tools {
                if tool.Name == "create_something" {
                    createTool = tool
                    break
                }
            }
            
            result, err := createTool.Handler(context.Background(), tt.request)
            
            if tt.expectError {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Contains(t, result.Content[0].(map[string]interface{})["text"], tt.expectMsg)
        })
    }
}
```
