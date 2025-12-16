# Dokku MCP Server

**Model Context Protocol (MCP)** server for **Dokku**, written in Go.

Version: v0.3.0

This server exposes Dokku's management capabilities through the standardized Model Context Protocol (MCP), allowing Large Language Models (LLMs) to interact with and manage a Dokku instance.

⚠️ **Early Development**: This project is in its early stages. Breaking changes are expected, and it is not recommended for production use.


[![MCP Badge](https://lobehub.com/badge/mcp-full/dokku-mcp-dokku-mcp)](https://lobehub.com/mcp/dokku-mcp-dokku-mcp)

## Try it now — turn your Dokku instance into an AI-manageable PaaS

[Follow Installation](#installation) or grab a [prebuilt binary](https://github.com/dokku-mcp/dokku-mcp/releases) to get started in minutes with cursor, claude-code, goose and all agentic tools which support mcp.

Or ```make start``` if you already have go locally.

### MCP Inspector Playground

For a quick tour of the server without wiring up a full MCP client, use the dedicated make target:

```bash
make inspect
```

It builds the binary (if needed), launches the MCP Inspector CLI, and connects it to the server in stdio mode. Inspector prints a local URL—open it in your browser to browse resources, prompts, and tools, or to issue ad-hoc calls. This is a great way to validate your setup before wiring Dokku MCP into Cursor, Claude, other IDE or internal tools.

### Connecting MCP Clients

The server can be used with any MCP-compatible client.

Add the following to your zed, `.cursor/mcp.json`, windsurf, `claude_desktop_config.json` or `.mcp.json`:

```json
{
  "mcpServers": {
    "dokku": {
      "command": "/home/user/dokku-mcp/build/dokku-mcp",
      "args": [],
      "env": {
        "DOKKU_MCP_SECURITY_BLACKLIST": "destroy,uninstall,remove",
        "DOKKU_MCP_SSH_HOST": "127.0.0.1",
        "DOKKU_MCP_SSH_KEY_PATH": "/home/user/.ssh/id_rsa",
        "DOKKU_MCP_SSH_PORT": "22",
        "DOKKU_MCP_SSH_USER": "dokku"
      }
    }
  }
}
```
*Remember to replace the `command` path with the absolute path to the binary.*

### Highlights

- **Core**: server info and plugin list resources; optional server logs tool.
- **Apps**: create, deploy (Git URL + ref), scale, env config, status; app list resource; troubleshooting prompt.
- **Deployments**: async deploys with IDs and background status.

## Roadmap

Submit an issue for re-priorizing proposal

### NEXT

- **Build-level log**: expose build output for deploy invocations.
- **App-level log**: expose runtime logs.
- **Inter-plugin communication**: allow inter-plugin events management, Eventbus maybe Watermill.
- **User management**: ssh-keys:add, boundaries and prompting.
- **Service plugins**: service templating, postgresql first implementation.

### IDEAS

- **SSH transactions / long-running connections**: streaming logs and interactive operations.
- **Enforce allow-list**: enforce least-principles with explicit allow-list over block-list.

## Dokku integrations

- **Implemented**: `apps:list`, `apps:info`, `apps:create`, `apps:destroy`, `apps:exists`, `apps:report`, `config:show`, `config:set`, `ps:scale`, `ps:report`, `logs`, `plugin:list`, `plugin:install`, `plugin:uninstall`, `plugin:enable`, `plugin:disable`, `plugin:update`, `version`, `proxy:report`, `proxy:set`, `scheduler:report`, `scheduler:set`, `git:report`, `git:set`, `ssh-keys:list`, `ssh-keys:remove`, `registry:logout`, `logs:set`.
- **Missing/partial**: `ssh-keys:add`, `registry:login`/registry listing, configuration key enumeration, Services plugin, SSL plugin, streaming/attach sessions.

## Contribute — report issues or propose features

[Open an issue](https://github.com/dokku-mcp/dokku-mcp/issues) or read [Contributing](CONTRIBUTING.md).

### 🪿 Get Help from Goose AI within issues

Need help with the dokku-mcp project? Goose AI is available to assist you! Simply tag your GitHub issues or pull requests with the `:goose` label, and Goose AI will automatically be notified to help with development, debugging, feature implementation, and other project-related tasks.

[Example of Goose AI in action](https://github.com/dokku-mcp/dokku-mcp/issues/12)
[Another example](https://github.com/dokku-MCP/dokku-mcp/issues/27)


## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)  
- [Local Dokku Development](#local-dokku-development)
- [Connecting MCP Clients](#connecting-mcp-clients)
- [Development](#development)
  - [Development Setup](#development-setup)
  - [Makefile Commands](#makefile-commands)
  - [Testing](#testing)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Pre-built Binaries

Download the latest release for your platform:

```bash
# Linux (amd64 / arm64 / arm)
curl -L -o dokku-mcp https://github.com/dokku-mcp/dokku-mcp/releases/download/v0.3.0/dokku-mcp-linux-amd64
chmod +x dokku-mcp
sudo mv dokku-mcp /usr/local/bin/

# macOS (amd64 / arm64)  
curl -L -o dokku-mcp https://github.com/dokku-mcp/dokku-mcp/releases/download/v0.3.0/dokku-mcp-darwin-amd64
chmod +x dokku-mcp
sudo mv dokku-mcp /usr/local/bin/
```

### Verify Installation

```bash
dokku-mcp --version
```

### Build from Source

If you prefer to build from source:

1. **Prerequisites:**
   - [Go](https://golang.org/doc/install) (version 1.25 or later)

2. **Clone and build:**
   ```bash
   git clone https://github.com/dokku-mcp/dokku-mcp.git
   cd dokku-mcp
   make build
   ```

## Configuration

The server can be configured in two ways: using a `config.yaml` file or via environment variables.

### Configuration File

Create a configuration file at one of the following locations:

- **System-wide**: `/etc/dokku-mcp/config.yaml`
- **User-specific**: `~/.dokku-mcp/config.yaml`
- **Local**: `config.yaml` in the same directory as the binary.

Here is a minimal `config.yaml` example:
```yaml
ssh:
  host: "your-dokku-host.com"
  user: "dokku"
  # key_path: "/path/to/your/ssh/private/key" # Optional, uses ssh-agent if empty

log_level: "info"
```

For a full list of available options, please refer to the [config.yaml.example](./config.yaml.example) file.

### Environment Variables

All configuration settings can be overridden with environment variables prefixed with `DOKKU_MCP_`. For example:

```bash
export DOKKU_MCP_SSH_HOST="your-dokku-host.com"
export DOKKU_MCP_SSH_USER="dokku"
export DOKKU_MCP_LOG_LEVEL="debug"
```

### Running the Server

Once configured, you can run the server:
```bash
dokku-mcp
```
The server will start and be ready to accept connections from an MCP client.

## Local Dokku Development

For development and testing without needing a remote Dokku instance, you can run a local Dokku server using Docker.

**Prerequisites:**
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [Make](https://www.gnu.org/software/make/)

1. **Set up the local Dokku container:**

   This command will download the necessary Docker images and configure the local Dokku instance. It only needs to be run once.
   ```bash
   make setup-dokku

   make dokku-start
   ```

2. **Stop the local Dokku container:**

   ```bash
   make dokku-stop
   ```

When the local Dokku container is running, the MCP server (with default config) should be able to connect to it. You can run integration tests against this local instance.

## Transport Modes
The server supports two transport modes for clients:
- **`stdio` (default):** Standard input/output for direct process communication.
- **`sse` (Server-Sent Events):** HTTP-based transport for web clients.
  ```bash
  DOKKU_MCP_TRANSPORT_TYPE=sse dokku-mcp
  ```

### CORS Configuration

For SSE transport, CORS is handled by default with `Access-Control-Allow-Origin: *`. For remote deployments, you can enable custom CORS middleware:

```yaml
transport:
  type: sse
  cors:
    enabled: true
    allowed_origins:
      - "https://app.example.com"
      - "*.example.com"
```

See [docs/CORS.md](docs/CORS.md) for detailed configuration options and security best practices.

## Development

This section is for developers who want to contribute to the project or modify the source code.

### Development Setup

1. **Prerequisites:**
   - [Go](https://golang.org/doc/install) (version 1.25 or later)
   - [Docker](https://docs.docker.com/get-docker/) and Docker Compose
   - [Make](https://www.gnu.org/software/make/)

2. **Clone the repository:**

   ```bash
   git clone https://github.com/dokku-mcp/dokku-mcp.git
   cd dokku-mcp
   ```

3. **Install development tools:**

   This command installs all the necessary Go tools for development, linting, and testing.

   ```bash
   make install-tools
   ```

4. **Set up the development environment:**

   This command sets up Git hooks to ensure code quality before commits.

   ```bash
   make setup-dev
   ```

5. **Build and run from source:**

   ```bash
   # This command builds the binary and starts the server.
   make start
   ```

### Makefile Commands

The project uses a `Makefile` to automate common tasks.

- `make help`: Show all available commands.
- `make check`: Run all code quality checks (linting, formatting, complexity).
- `make build`: Build the server binary.
- `make clean`: Clean up build artifacts.

### Testing

- **Run all tests:**
  ```bash
  make test
  ```
  This runs all unit and integration tests and generates an HTML coverage report at `coverage.html`.

- **Run integration tests against local Dokku:**
  Make sure your local Dokku container is running (`make dokku-start`).
  ```bash
  make test-integration-local
  ```

## Architecture

The server follows **Domain-Driven Design (DDD)** principles, with a clear separation between:
- **Domain Layer (`internal/domain`):** Core business logic, entities, and repository interfaces.
- **Application Layer (`internal/application`):** Use case orchestration and coordination.
- **Infrastructure Layer (`internal/infrastructure`):** Implementations of interfaces, such as the Dokku client, databases, and external services.

It features a **plugin-based architecture** located in `internal/server-plugins`, where each plugin encapsulates a specific set of Dokku features (e.g., `app`, `core`, `deployment`).

For more details, please refer to the documentation in the `docs/` directory.

## Project Structure

```
dokku-mcp/
├── cmd/                    # Entry points for the application
│   └── server/             # Server main command
├── internal/               # Private application code
│   ├── server/             # MCP server and adapter
│   ├── server-plugin/      # Plugin system infrastructure
│   ├── server-plugins/     # Actual plugin implementations (app, core, etc.)
│   ├── dokku-api/          # Dokku CLI client and API
│   └── shared/             # Shared domain types and services
├── pkg/                    # Reusable packages (config, logger, etc.)
├── docs/                   # Project documentation
└── scripts/                # Helper scripts
```

## Contributing

Contributions are welcome! Please see [Contributing](CONTRIBUTING.md) for detailed guidance on how to contribute.

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
