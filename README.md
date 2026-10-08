# Dokku MCP Server

**Model Context Protocol (MCP)** server for **Dokku**, written in Go.

Version: see [releases](https://github.com/dokku-mcp/dokku-mcp/releases) (`dokku-mcp --version`)

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

### Tools

Tools are grouped by plugin. Plugins tied to a Dokku plugin (for example Let's Encrypt) only appear when that plugin is installed. Every tool carries MCP annotations (title, read-only, destructive, idempotent), and newer tools return structured content with an output schema.

| Area | Tools |
|------|-------|
| Applications | `create_app`, `deploy_app`, `rollback_app`, `scale_app`, `configure_app`, `get_app_status`, `restart_app`, `stop_app`, `start_app` |
| Deployments | `get_deployment_status`, `list_deployments` |
| Logs | `get_runtime_logs`, `follow_runtime_logs` (live, streamed as progress notifications), `get_failed_deploy_logs` |
| Datastores | `list_services`, `create_service`, `get_service_info`, `link_service`, `unlink_service`, `get_service_logs`, `destroy_service` (postgres, mysql, mariadb, redis, mongo, rabbitmq, memcached, clickhouse, elasticsearch) |
| Domains | `get_app_domains`, `add_app_domain`, `remove_app_domain`, `list_global_domains`, `add_global_domain` |
| HTTPS | `get_letsencrypt_status`, `enable_letsencrypt`, `disable_letsencrypt`, `set_letsencrypt_email` |
| Access | `list_ssh_keys`, `add_ssh_key`, `remove_ssh_key`, `get_registry_report`, `registry_login`, `registry_logout` |
| Server | `get_server_logs` (opt-in with `expose_server_logs`) |

Resources include `dokku://onboarding/quickstart` (start here), `dokku://onboarding/capabilities`, `dokku://onboarding/intent-map`, `dokku://core/server/info` and `dokku://app/{name}/logs`. The `app_doctor` prompt walks through diagnosing an application.

Deployments are asynchronous: `deploy_app` returns a `deployment_id` as soon as the code is synced, and `get_deployment_status` reports the build outcome and the tail of the build log.

## Security

The server runs Dokku commands over SSH with the configured key, so treat it like that key.

- **Argument safety.** Dokku's SSH command word-splits arguments, so every argument is checked against a shell-safe character set. Free-form values (environment variables, registry passwords, SSH public keys) are sent base64-encoded or over stdin, never interpolated into the command line.
- **Allowlist / blacklist.** `security.allowlist` restricts the server to matching commands (`"apps:"`, `"config:*"`, `"logs"`, ...); `security.blacklist` blocks commands containing a pattern (`destroy`, `uninstall`, ...). Both can be set with `DOKKU_MCP_SECURITY_ALLOWLIST` / `DOKKU_MCP_SECURITY_BLACKLIST` as comma-separated lists.
- **Secrets in output.** Datastore connection URLs and passwords are masked in tool results.
- **Guard rails.** `destroy_service` requires the service name to be repeated; `add_ssh_key` refuses names containing `admin` (which Dokku lets add further keys) unless `allow_admin` is set.
- **Remote transport.** The SSE transport has no authentication yet. Only expose it on a trusted network or behind an authenticating proxy.

## Roadmap

Submit an issue for re-priorizing proposal

### NEXT

- **Streamable HTTP transport and authentication**: replace SSE, add bearer token / OAuth support and per-token scopes.
- **Confirmation for destructive tools**: use MCP elicitation for destroy/stop/remove operations.
- **Distribution**: Docker image, Homebrew tap and a Dokku plugin that runs the server on the host.

### IDEAS

- **Release history**: keep a durable deployment log so rollbacks can pick from previous releases.
- **More Dokku plugins**: ports, checks, cron, storage mounts.

## Dokku integrations

- **Implemented**: `apps:list`, `apps:info`, `apps:create`, `apps:exists`, `apps:report`, `config:show`, `config:set --encoded`, `ps:scale`, `ps:report`, `ps:rebuild`, `ps:restart`, `ps:stop`, `ps:start`, `git:sync`, `logs` (including `-t` follow), `logs:failed`, `domains:report`, `domains:add`, `domains:remove`, `domains:add-global`, `letsencrypt:*`, `<datastore>:create/info/list/link/unlink/links/logs/destroy`, `ssh-keys:list/add/remove`, `registry:login/logout/report`, `plugin:list`, `version`, `proxy:report`, `scheduler:report`, `git:report`.
- **Missing/partial**: `ports:*`, `checks:*`, `storage:*`, `cron:*`, datastore backups.

## Contribute — report issues or propose features

[Open an issue](https://github.com/dokku-mcp/dokku-mcp/issues) or read [Contributing](CONTRIBUTING.md).

### 🪿 Get Help from Goose AI within issues

Need help with the dokku-mcp project? Goose AI is available to assist you! Simply tag your GitHub issues or pull requests with the `:goose` label, and Goose AI will automatically be notified to help with development, debugging, feature implementation, and other project-related tasks.

[Example of Goose AI in action](https://github.com/dokku-mcp/dokku-mcp/issues/12)
[Another example](https://github.com/dokku-MCP/dokku-mcp/issues/27)


## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Security](#security)
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
# Linux (amd64; also available: arm64, arm)
curl -L -o dokku-mcp https://github.com/dokku-mcp/dokku-mcp/releases/latest/download/dokku-mcp-linux-amd64
chmod +x dokku-mcp
sudo mv dokku-mcp /usr/local/bin/

# macOS (amd64; also available: arm64)
curl -L -o dokku-mcp https://github.com/dokku-mcp/dokku-mcp/releases/latest/download/dokku-mcp-darwin-amd64
chmod +x dokku-mcp
sudo mv dokku-mcp /usr/local/bin/
```

Compressed archives (`dokku-mcp-<os>-<arch>.tar.gz`) are also published with each release.

### MCP Bundle

Each release also ships `dokku-mcp.mcpb`, an [MCP Bundle](https://github.com/modelcontextprotocol/mcpb) for macOS and Linux that MCP clients such as Claude Desktop install in one step, asking for the Dokku host, SSH user, port and key. The server is listed in the [official MCP Registry](https://registry.modelcontextprotocol.io) as `io.github.dokku-MCP/dokku-mcp`.

### Verify Installation

```bash
dokku-mcp --version
```

### Build from Source

If you prefer to build from source:

1. **Prerequisites:**
   - [Go](https://golang.org/doc/install) (version 1.26 or later)

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

All configuration settings can be overridden with environment variables prefixed with `DOKKU_MCP_`. Nested keys use underscores (`ssh.host` → `DOKKU_MCP_SSH_HOST`) and lists are comma-separated. For example:

```bash
export DOKKU_MCP_SSH_HOST="your-dokku-host.com"
export DOKKU_MCP_SSH_USER="dokku"
export DOKKU_MCP_LOG_LEVEL="debug"
export DOKKU_MCP_SECURITY_ALLOWLIST="apps:,config:,ps:,git:sync,logs,version"
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
   - [Go](https://golang.org/doc/install) (version 1.26 or later)
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
