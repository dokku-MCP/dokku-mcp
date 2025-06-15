# Dokku MCP Server Architecture

## Overview

The Dokku MCP server is designed with a layered architecture that follows Domain Driven Design (DDD) principles while maintaining simplicity and extensibility.

## Layered Architecture

```
┌─────────────────────────────────────┐
│           MCP Protocol Layer        │
│  (Transport, Serialization, Auth)   │
├─────────────────────────────────────┤
│          Application Layer          │
│    (Handlers, Coordinators)         │
├─────────────────────────────────────┤
│            Domain Layer             │
│  (Entities, Services, Repositories) │
├─────────────────────────────────────┤
│         Infrastructure Layer        │
│   (Dokku CLI, File System, HTTP)    │
└─────────────────────────────────────┘
```

## Business Domains

### Application Domain
- **Entities**: Application, Deployment, Process
- **Services**: DeploymentService, ScalingService
- **Repository**: ApplicationRepository

### Service Domain  
- **Entities**: Service, Database, Storage
- **Services**: ServiceManager, BackupService
- **Repository**: ServiceRepository

### Infrastructure Domain
- **Entities**: Server, Network, Certificate
- **Services**: ResourceMonitor, CertificateManager
- **Repository**: InfrastructureRepository

## Core Components

### MCP Server Core
```go
type MCPServer struct {
    resourceManager ResourceManager
    toolManager     ToolManager
    promptManager   PromptManager
    dokkuClient     DokkuClient
}
```

### Resource Manager
- Exposes Dokku resources as MCP resources
- Manages caching and synchronization
- Provides structured data views

### Tool Manager
- Implements MCP tools for Dokku actions
- Validates input parameters
- Executes Dokku commands securely

### Dokku Client
- Interface for executing Dokku commands
- Error handling and timeouts
- Command output parsing

## Architectural Patterns

### Repository Pattern
```go
type ApplicationRepository interface {
    GetAll() ([]*domain.Application, error)
    GetByName(name string) (*domain.Application, error)
    Create(app *domain.Application) error
    Update(app *domain.Application) error
    Delete(name string) error
}
```

### Service Pattern
```go
type DeploymentService interface {
    Deploy(appName string, options DeployOptions) (*domain.Deployment, error)
    Rollback(appName string, version string) error
    GetHistory(appName string) ([]*domain.Deployment, error)
}
```

### Factory Pattern
```go
type ToolFactory interface {
    CreateTool(toolType ToolType) (Tool, error)
    ListAvailableTools() []ToolDefinition
}
```

## Error Management

### Error Hierarchy
```go
type DomainError interface {
    error
    Code() string
    Message() string
    Details() map[string]interface{}
}

type ApplicationError struct {
    code    string
    message string
    details map[string]interface{}
}

type ValidationError struct {
    ApplicationError
    field string
    value interface{}
}
```

## Configuration and Security

### Configuration
```go
type Config struct {
    MCP     MCPConfig     `yaml:"mcp"`
    Dokku   DokkuConfig   `yaml:"dokku"`
    Logging LoggingConfig `yaml:"logging"`
    Security SecurityConfig `yaml:"security"`
}
```

### Security
- Strict input validation
- Limited authorized commands
- Audit trail of executed actions
- Rate limiting per client

## Performance and Monitoring

### Caching Strategy
- In-memory cache for frequently accessed data
- Configurable TTL per resource type
- Invalidation on state changes

### Metrics
- Dokku command performance metrics
- MCP resource usage statistics
- Server health monitoring

## Deployment and CI/CD

### Release Structure
```
releases/
├── linux-amd64/
├── linux-arm64/
├── darwin-amd64/
└── darwin-arm64/
```

### Tests
- Unit tests for each layer
- Integration tests with Dokku
- Performance and load tests
- Security tests 