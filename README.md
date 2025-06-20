# Dokku MCP Server

**Model Context Protocol (MCP)** for **Dokku** written in Go.

## Overview

The Dokku MCP Server bridges Dokku infrastructure management and LLMs by exposing Dokku capabilities through the standardized Model Context Protocol (MCP).

**Current Status**: Early development stage with core application management features implemented. The server provides MCP resources, tools, and prompts for managing Dokku applications and monitoring system status.

⚠️ **Development Warning**: This is in early development with breaking changes expected. Not recommended for production use as it could potentially break your Dokku infrastructure.

## Features

### MCP Resources
- **Applications**: Complete list of Dokku applications with status and summary statistics
- **System Status**: Current Dokku server status including version, configuration, and resource usage
- **Server Information**: Comprehensive server details including plugins, domains, and configuration
- **Plugin List**: All installed Dokku plugins with their status and versions

### MCP Tools
- **Application Management - not tested**: 
  - Create new applications with validation
  - Deploy applications from Git repositories
  - Scale application processes
  - Configure environment variables
  - Get comprehensive application status
- **System Tools**:
  - Get system status and configuration
  - List installed Dokku plugins

### MCP Prompts
- **Application Doctor**: Application health diagnosis and troubleshooting guidance

## Current Development Status

🚧 **Early Development Stage** - Breaking changes are expected.

⚠️ **Not Production Ready** - Could potentially break your Dokku infrastructure.

### What's Implemented
- ✅ Core MCP server with mcp-go
- ✅ Plugin architecture
- ✅ Application management (create, deploy, scale, configure, status)
- ✅ System status and dokku plugin information
- ✅ Basic diagnostic prompts

### Proposed planned Features
- Dokku integration
   - 🔄 **Services**: Database connections, service status, and configuration
   - 🔄 **Deployments**: Deployment introspection
   - 🔄 **Logs**: Application and system logs with filtering  
   - 🔄 **Metrics**: Performance metrics and resource usage data
   - 🔄 **Service Management**: Create, backup, and restore
   - 🔄 **Advanced Monitoring**: Health checks and diagnostics
   - 🔄 **Security Prompts**: Security audits and recommendations
   - 🔄 **Optimization Prompts**: Performance improvement recommendations
- Infrastructure management extras
   - 🔄 **Workflows**: Playbooks for common use-cases available at tool command


## Architecture

### Key Principles
- **Domain-Driven Design** : Clear separation of business logic and infrastructure
- **Strong Typing - target** : Explicit types everywhere, no `interface{}` without justification
- **Error Handling** : Complete error handling with context and logging
- **Security** : Input validation, command sanitization, and audit logging
- **Performance** : Cache, connection pool, and resource optimization
- **Extensibility** : Plugin system for adding new capabilities

## Quick Start

### Installation - No built binaries yet

1. **Clone the repository**
   ```bash
   git clone https://github.com/alex-galey/dokku-mcp.git
   cd dokku-mcp
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Install development tools**
   ```bash
   make install-tools
   ```

4. **Build the server**
   ```bash
   make build
   ```

5. **Run the server**
   ```bash
   ./build/dokku-mcp
   ```

### Configuration

Create a `config.yaml` file from config.yaml.example:

### Environment Variables

All configuration options can be set via environment variables with the `DOKKU_MCP_` prefix:

```bash
export DOKKU_MCP_HOST="0.0.0.0"
export DOKKU_MCP_PORT="8080"
export DOKKU_MCP_LOG_LEVEL="debug"
export DOKKU_MCP_DOKKU_PATH="/usr/bin/dokku"
```

## Usage with mcp-go

This project uses **mcp-go** from [mark3labs](https://github.com/mark3labs/mcp-go) to implement the MCP protocol. Here are the key points:

### MCP Architecture

```go
// Main server with mcp-go
mcpServer := server.NewMCPServer(
    "Dokku MCP Server",
    Version,
    server.WithToolCapabilities(true),
    server.WithResourceCapabilities(true),
    server.WithPromptCapabilities(true),
)
```

### Resource Registration
```go
// Resource for applications
applicationsResource := mcp.NewResource(
    "dokku://applications",
    mcp.WithResourceName("Applications Dokku"),
    mcp.WithResourceDescription("List of all Dokku applications"),
    mcp.WithResourceMimeType("application/json"),
)

mcpServer.AddResource(applicationsResource, handler.handleApplicationsResource)
```

### Resource Registration
```go
// Tool creation
createTool := mcp.NewTool(
    "create_application",
    mcp.WithDescription("Create a new Dokku application"),
    mcp.WithInputSchema(schema),
)

mcpServer.AddTool(createTool, handler.handleCreateApplication)
```

### Resource Registration
```go
// Diagnostic prompt
diagnosticPrompt := mcp.NewPrompt(
    "diagnose_application",
    mcp.WithPromptName("Diagnose an Application"),
    mcp.WithPromptDescription("Analyze potential issues with a Dokku application"),
)

mcpServer.AddPrompt(diagnosticPrompt, handler.handleDiagnosticPrompt)
```

## Development

### Project Structure

```
dokku-mcp/
├── cmd/                    # Entry points for the application
│   └── server/            # Server main command
├── internal/              # Private application code
│   ├── server/            # MCP server and adapter
│   ├── server-plugin/     # Plugin system infrastructure
│   │   ├── domain/        # Plugin interfaces and contracts
│   │   └── application/   # Plugin registry and management
│   ├── server-plugins/    # Actual plugin implementations
│   │   ├── app/          # Application management plugin
│   │   ├── core/         # Core system functionality plugin
│   │   └── deployment/   # Deployment services (in development)
│   ├── dokku-api/        # Dokku CLI client and API
│   ├── shared/           # Shared domain services
│   └── pkg/              # Package utilities
├── docs/                  # Documentation
└── scripts/              # Build and deployment scripts
```

### Development Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Verify code quality**
   ```bash
   make check
   ```

4. **Run integration tests**
   ```bash
   make test-integration
   ```

### Available Make Commands

```bash
make help                # Show all available commands
```

### Adding New Features

#### 1. Add a new MCP plugin

1. **Create plugin directory** in `internal/server-plugins/your-feature/`
2. **Implement the plugin interface** in `plugin.go`:
   ```go
   type YourFeaturePlugin struct {
       // dependencies
   }
   
   func (p *YourFeaturePlugin) ID() string { return "your-feature" }
   func (p *YourFeaturePlugin) Name() string { return "Your Feature" }
   // ... other ServerPlugin methods
   ```
3. **Implement provider interfaces** (ResourceProvider, ToolProvider, PromptProvider)
4. **Register in module system** with dependency injection
5. **Add comprehensive tests** for all functionality

#### 2. Extend existing plugin

1. **Add new resources/tools/prompts** to existing plugin in `internal/server-plugins/`
2. **Implement handlers** with proper validation and error handling
3. **Update domain entities** if needed in plugin's domain layer
4. **Add tests** for new functionality

#### 3. Add domain functionality

1. **Define entities and value objects** in plugin's `domain/` directory
2. **Create repository interfaces** for data access
3. **Implement infrastructure adapters** for Dokku CLI integration
4. **Wire through dependency injection** in plugin module

## Tests

### Unit Tests
```bash
make test
```

### Integration Tests
```bash
make test-integration
```

### Coverage Report
```bash
make test-coverage
# Open coverage.html in browser
```

### Load Tests
```bash
make load-test
```

## Deployment

### Binary Version
```bash
make build
# Binary available as ./build/dokku-mcp
```

### Running the Server
```bash
# Default stdio mode (for MCP clients)
./dokku-mcp

# SSE mode (for web clients)
DOKKU_MCP_TRANSPORT_TYPE=sse DOKKU_MCP_TRANSPORT_HOST=0.0.0.0 DOKKU_MCP_TRANSPORT_PORT=8080 ./dokku-mcp
```

### Docker
```bash
docker build -t dokku-mcp .
docker run dokku-mcp
```

## Integration with MCP Clients

### Claude Desktop

Add to `~/.config/claude_desktop/claude_desktop_config.json` :

```json
{
  "mcpServers": {
    "dokku": {
      "command": "/path/to/dokku-mcp",
      "args": [],
      "env": {
        "DOKKU_MCP_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Custom MCP Clients

The server supports two transport modes:

**stdio (default)**: Standard input/output for direct process communication
```bash
./dokku-mcp
```

**SSE (Server-Sent Events)**: HTTP-based transport for web clients
```bash
DOKKU_MCP_TRANSPORT_TYPE=sse DOKKU_MCP_TRANSPORT_HOST=0.0.0.0 DOKKU_MCP_TRANSPORT_PORT=8080 ./dokku-mcp
```

## Contribution

Contributions are welcome ! Please see our [Development Playbook](docs/playbooks/development.md) for detailed guidance.

### Code Style
- Follow Go conventions (gofmt, golint)
- Use strong typing everywhere
- Document all public functions
- Write complete tests
- Follow DDD principles

### Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Make changes
4. Add tests for new features
5. Ensure all tests pass : `make test-all`
6. Submit a pull request

## Support

- **Documentation** : [docs/](docs/)
- **Issues** : [GitHub Issues](https://github.com/alex-galey/dokku-mcp/issues)
- **Discussions** : [GitHub Discussions](https://github.com/alex-galey/dokku-mcp/discussions)

## Acknowledgments

- [mcp-go](https://github.com/mark3labs/mcp-go) for the Go implementation of the MCP protocol
- [Dokku](https://dokku.com/) for the fantastic PaaS platform
- MCP community for specifications and tools

## License

This project is under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright [Alex Galey]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
