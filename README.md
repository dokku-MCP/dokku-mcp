# Dokku MCP Server

**Model Context Protocol (MCP)** for **Dokku** written in Go.

## Overview

The Dokku MCP Server bridges large language models and Dokku infrastructure management by exposing Dokku capabilities through the standardized Model Context Protocol. It follows Domain-Driven Design principles and maintains a clean, extensible architecture.

## Features

### Core Capabilities
- **Application Management**: Create, deploy, scale, and monitor Dokku applications
- **Service Management**: Manage databases, storage, and other Dokku services
- **Infrastructure Monitoring**: Access logs, metrics, and system status
- **Domain and SSL Management**: Configure domains and SSL certificates
- **Environment Configuration**: Manage environment variables and application settings

### MCP Resources
- **Applications**: Real-time application status, configuration, and deployment history
- **Services**: Database connections, service status, and configuration
- **Logs**: Application and system logs with filtering capabilities
- **Metrics**: Performance metrics and resource usage data

### MCP Tools
- **Deployment**: Deploy applications from Git repositories
- **Scaling**: Adjust application process scaling
- **Configuration**: Update environment variables and settings
- **Service Management**: Create, backup, and restore services
- **Monitoring**: Execute health checks and diagnostics

### MCP Prompts
- **Diagnostics**: Analyze application and performance issues
- **Optimization**: Generate performance improvement recommendations
- **Security**: Provide security audits and recommendations
- **Operations**: Assist with common operational tasks

## Architecture

### Layered Design
```
┌─────────────────────────────────────┐
│        Couche Protocole MCP         │
│  (Transport, Sérialisation, Auth)   │
├─────────────────────────────────────┤
│         Couche Application          │
│     (Handlers, Coordinateurs)       │
├─────────────────────────────────────┤
│          Couche Domaine             │
│ (Entités, Services, Repositories)   │
├─────────────────────────────────────┤
│       Couche Infrastructure         │
│   (CLI Dokku, Système de fichiers)  │
└─────────────────────────────────────┘
```

### Key Principles
- **Domain-Driven Design** : Clear separation of business logic and infrastructure
- **Strong Typing** : Explicit types everywhere, no `interface{}` without justification
- **Error Handling** : Complete error handling with context and logging
- **Security** : Input validation, command sanitization, and audit logging
- **Performance** : Cache, connection pool, and resource optimization
- **Extensibility** : Plugin system for adding new capabilities

## Quick Start

### Prerequisites
- Go 1.21 or newer
- Dokku installed and configured
- Git for version control

### Installation

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

Create a `config.yaml` file:

```yaml
# Server configuration
host: "localhost"
port: 8080
log_level: "info"
log_format: "json"
timeout: "30s"

# Dokku configuration
dokku_path: "/usr/bin/dokku"

# Cache configuration
cache_enabled: true
cache_ttl: "5m"

# Security configuration
security:
  allowed_commands:
    - "apps:list"
    - "apps:info"
    - "config:get"
    - "config:set"
    - "ps:scale"
  rate_limit:
    enabled: true
    requests_per_minute: 60
```

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
│   ├── domain/            # Domain layer (business logic)
│   │   └── application/   # Application domain
│   ├── application/       # Application layer
│   │   └── handlers/      # MCP request handlers
│   └── infrastructure/    # Infrastructure layer
│       └── dokku/        # Dokku client implementation
├── docs/                  # Documentation
├── scripts/              # Build and deployment scripts
└── tests/                # Test files
```

### Development Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Follow TDD approach**
   ```bash
   # Write tests first
   make test
   
   # Implement the feature
   # Re-run tests
   make test
   ```

3. **Verify code quality**
   ```bash
   make lint
   make fmt
   make vet
   ```

4. **Run integration tests**
   ```bash
   make test-integration
   ```

### Available Make Commands

```bash
make help                # Show all available commands
make build               # Build the server
make test                # Run unit tests
make test-coverage       # Run tests with coverage
make test-integration    # Run integration tests
make lint                # Analyze code
make fmt                 # Format code
make vet                 # Run static analysis
make clean               # Clean build artifacts
make install-tools       # Install development tools
```

### Adding New Features

#### 1. Add a new MCP resource

1. **Define the domain entity** in `internal/domain/`
2. **Create the repository interface** for data access
3. **Implement the infrastructure** in `internal/infrastructure/`
4. **Create the MCP handler** in `internal/application/handlers/`
5. **Add tests** for all layers

#### 2. Add a new MCP tool

1. **Define the tool structure** in `internal/application/tools/`
2. **Implement the tool logic** with appropriate validation
3. **Add to tool registry**
4. **Write complete tests**

#### 3. Add a plugin

1. **Create the plugin directory** in `internal/plugins/`
2. **Implement the plugin interface**
3. **Register the plugin** in the main server
4. **Add specific tests for the plugin**

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

## Security

### Input Validation
- All user inputs are validated before processing
- Dokku command parameters are sanitized
- Resource access is controlled and audited

### Audit Logging
- All operations are logged with client identification
- Sensitive operations require additional validation
- Complete audit trail for compliance

### Rate Limiting
- Configurable rate limits per client
- Protection against abuse and resource exhaustion
- Graceful degradation under load

## Monitoring

### Health Checks
```bash
curl http://localhost:8080/health
```

### Metrics
- Prometheus metrics endpoint: `/metrics`
- Custom metrics for Dokku operations
- Performance and utilization statistics

### Logging
- Structured JSON logging
- Configurable log levels
- Request/response tracing

## Deployment

### Binary Version
```bash
make build-all
# Binaries available in build/
```

### Docker
```bash
docker build -t dokku-mcp .
docker run -p 8080:8080 dokku-mcp
```

### Systemd Service
```bash
sudo cp scripts/dokku-mcp.service /etc/systemd/system/
sudo systemctl enable dokku-mcp
sudo systemctl start dokku-mcp
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

The server uses stdio by default but can be extended to support HTTP/SSE :

```go
// Example extension for HTTP
mcpServer.ServeHTTP(ctx, ":8080")
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

- [mcp-go](https://github.com/mark3labs/mcp-go) for the excellent Go implementation of the MCP protocol
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
