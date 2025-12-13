# Dokku MCP Dev Container

This Dev Container provides a complete Go development environment for the Dokku MCP project.

## Features

### Base Image
- **Go 1.23** (Debian Bookworm)
- Pre-configured for Go development

### Included Tools
- **Air** - Live reload for Go apps
- **Ginkgo** - BDD testing framework
- **goimports** - Auto-format and organize imports
- **golangci-lint** - Comprehensive Go linter
- **staticcheck** - Advanced static analysis
- **Docker-in-Docker** - For testing Docker-based features
- **GitHub CLI** - For GitHub operations

### VS Code Extensions
- **Go** - Official Go extension with language server
- **GitHub Copilot** - AI pair programming
- **GitLens** - Enhanced Git integration
- **Docker** - Docker container management
- **YAML/TOML** - Configuration file support
- **Markdown** - Documentation editing
- **Test Explorer** - Visual test runner

### Configuration
- **Auto-format on save** with goimports
- **Organize imports** automatically
- **Lint on save** with golangci-lint
- **Test with race detector** enabled
- **SSH keys mounted** from host (read-only)

## Quick Start

### Open in Dev Container

1. **VS Code**: Open Command Palette (Ctrl+Shift+P) → "Dev Containers: Reopen in Container"
2. **Gitpod**: Automatically opens in dev container

### Available Commands

```bash
# Development with live reload
make dev

# Run tests
make test

# Run tests with coverage
make test-coverage

# Lint code
make lint

# Build binary
make build

# Install/update tools
make install-tools
```

## Port Forwarding

- **8080** - MCP Server (SSE transport)

Ports are automatically forwarded and you'll be notified when the server starts.

## Environment Variables

Pre-configured:
- `GOPROXY=https://proxy.golang.org,direct`
- `GOSUMDB=sum.golang.org`
- `CGO_ENABLED=0` (for static binaries)

## SSH Keys

Your host SSH keys are mounted at `/home/vscode/.ssh` (read-only) for Git operations.

## Lifecycle Hooks

- **postCreateCommand**: Installs tools and downloads Go modules
- **postStartCommand**: Configures Git safe directory
- **postAttachCommand**: Shows welcome message with quick commands

## Customization

### Add More Extensions

Edit `.devcontainer/devcontainer.json`:

```json
"customizations": {
  "vscode": {
    "extensions": [
      "your.extension-id"
    ]
  }
}
```

### Add System Packages

Edit `.devcontainer/Dockerfile`:

```dockerfile
RUN apt-get update && apt-get install -y \
    your-package \
    && apt-get clean
```

### Add Go Tools

Edit `.devcontainer/Dockerfile`:

```dockerfile
RUN go install github.com/your/tool@latest
```

## Troubleshooting

### Go modules not found
```bash
go mod download
```

### Tools not available
```bash
make install-tools
```

### Git safe directory warning
```bash
git config --global --add safe.directory /workspaces/dokku-mcp
```

### Rebuild container
VS Code: Command Palette → "Dev Containers: Rebuild Container"

## Performance Tips

- Use **Docker Desktop** on Mac/Windows for better performance
- Enable **VirtioFS** (Mac) or **WSL2** (Windows) for faster file access
- Keep **node_modules** and **vendor** directories in named volumes

## More Information

- [Dev Containers Documentation](https://containers.dev/)
- [VS Code Dev Containers](https://code.visualstudio.com/docs/devcontainers/containers)
- [Gitpod Dev Containers](https://www.gitpod.io/docs/configure/workspaces/devcontainer)
