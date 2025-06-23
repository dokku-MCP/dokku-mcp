# Dokku MCP Server

**Model Context Protocol (MCP)** server for **Dokku**, written in Go.

Version: v0.1.0-alpha

This server exposes Dokku's management capabilities through the standardized Model Context Protocol (MCP), allowing Large Language Models (LLMs) to interact with and manage a Dokku instance.

⚠️ **Early Development**: This project is in its early stages. Breaking changes are expected, and it is not recommended for production use.

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
curl -L -o dokku-mcp https://github.com/alex-galey/dokku-mcp/releases/download/v0.1.0-alpha/dokku-mcp-linux-amd64
chmod +x dokku-mcp
sudo mv dokku-mcp /usr/local/bin/

# macOS (amd64 / arm64)  
curl -L -o dokku-mcp https://github.com/alex-galey/dokku-mcp/releases/download/v0.1.0-alpha/dokku-mcp-darwin-amd64
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
   - [Go](https://golang.org/doc/install) (version 1.24 or later)

2. **Clone and build:**
   ```bash
   git clone https://github.com/alex-galey/dokku-mcp.git
   cd dokku-mcp
   make build
   ```

## Configuration

The server is configured via a `config.yaml` file or environment variables.

1. **Create a configuration file:**

   Copy the example configuration file:
   ```bash
   cp config.yaml.example config.yaml
   ```

2. **Edit `config.yaml`:**

   Adjust the settings in `config.yaml` to match your environment. At a minimum, you'll need to configure the SSH settings to connect to your Dokku host.

   | Setting | Description |
   |---|---|
   | `host` | Server host to bind to. |
   | `port` | Server port to listen on. |
   | `log_level` | Logging level (`debug`, `info`, `warn`, `error`). |
   | `ssh.host` | Dokku host server. |
   | `ssh.user` | Dokku SSH user (usually `dokku`). |
   | `ssh.key_path`| Path to the SSH private key for Dokku access (optional, uses ssh-agent if empty). |

Alternatively, all settings can be controlled via environment variables with the `DOKKU_MCP_` prefix (e.g., `DOKKU_MCP_SSH_HOST`).

3. **Run the server:**

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
   ```
   
2. **Start the local Dokku container:**

   ```bash
   make dokku-start
   ```

3. **Stop the local Dokku container:**

   ```bash
   make dokku-stop
   ```

When the local Dokku container is running, the MCP server (with default config) should be able to connect to it. You can run integration tests against this local instance.

## Connecting MCP Clients

The server can be used with any MCP-compatible client.

### Claude Desktop

Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "dokku": {
      "command": "/path/to/your/dokku-mcp",
      "args": [],
      "env": {
        "DOKKU_MCP_LOG_LEVEL": "info"
      }
    }
  }
}
```
*Remember to replace the `command` path with the absolute path to the binary.*

### Transport Modes
The server supports two transport modes for clients:
- **`stdio` (default):** Standard input/output for direct process communication.
- **`sse` (Server-Sent Events):** HTTP-based transport for web clients.
  ```bash
  DOKKU_MCP_TRANSPORT_TYPE=sse dokku-mcp
  ```

## Development

This section is for developers who want to contribute to the project or modify the source code.

### Development Setup

1. **Prerequisites:**
   - [Go](https://golang.org/doc/install) (version 1.24 or later)
   - [Docker](https://docs.docker.com/get-docker/) and Docker Compose
   - [Make](https://www.gnu.org/software/make/)

2. **Clone the repository:**

   ```bash
   git clone https://github.com/alex-galey/dokku-mcp.git
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
