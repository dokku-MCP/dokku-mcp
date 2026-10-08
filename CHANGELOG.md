# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Deployments**: `deploy_app` returns a `deployment_id`; new `get_deployment_status` and `list_deployments` tools report the real build outcome and the tail of the build log. `rollback_app` redeploys an earlier ref from the last deployed repository.
- **Lifecycle and diagnostics**: `restart_app`, `stop_app`, `start_app`, `get_failed_deploy_logs`, and `follow_runtime_logs` (live logs streamed as MCP progress notifications).
- **Datastores**: `list_services`, `create_service`, `get_service_info`, `link_service`, `unlink_service`, `get_service_logs`, `destroy_service` for the official Dokku datastore plugins. Credentials are masked in output.
- **Domains and HTTPS**: `get_app_domains`, `add_app_domain`, `remove_app_domain`; Let's Encrypt tools `get_letsencrypt_status`, `enable_letsencrypt`, `disable_letsencrypt`, `set_letsencrypt_email` (active when dokku-letsencrypt is installed).
- **Access**: `list_ssh_keys`, `add_ssh_key`, `remove_ssh_key`, `get_registry_report`, `registry_login` (password over stdin), `registry_logout`.
- **Security**: `security.allowlist` to restrict commands; every Dokku argument is validated against a shell-safe character set.
- **MCP**: tool annotations (title, read-only, destructive, idempotent hints) on every tool, structured results with output schemas, server instructions, input-schema validation and panic recovery.

### Fixed
- `configure_app` never applied anything (the repository's environment extraction was a stub). Values are now sent with `config:set --encoded`, so spaces, quotes and shell characters survive, and `restart` can be disabled.
- Saving an application no longer re-sends its whole configuration (scaling used to rewrite every variable).
- Write commands (`git:sync`, `ps:rebuild`, `apps:create`, ...) were cached for up to five minutes, so repeated calls silently did nothing; only read-only queries are cached now and writes invalidate the cache.
- Deployments were reported successful while the new build was still running; the `ps:rebuild` exit status now decides, with polling only as a fallback.
- `get_runtime_logs` and `dokku://app/{name}/logs` returned placeholder text; they now return real logs, using `--num` instead of `--tail` (which follows the stream until timeout).
- Nested environment variables such as `DOKKU_MCP_SSH_HOST` and `DOKKU_MCP_SECURITY_BLACKLIST` were ignored.
- CORS wildcard `*.example.com` also matched `evilexample.com`.
- `ssh-keys:list` reported fingerprints as key names.
- Dokku error messages are included in tool errors instead of only `exit status 1`.
- Data race between deployment status reads and background updates.

### Changed
- Go 1.26 (toolchain go1.26.8); mcp-go v0.43.2 → v1.1.1 (supports the stateless 2026-07-28 protocol revision alongside earlier ones); all dependencies and GitHub Actions updated.
- `ssh.key_path` defaults to empty (ssh-agent / `~/.ssh` fallback) instead of `dokku_mcp_test`.
- `create_app` no longer advertises the unused `buildpack` and `no_vhost` parameters; `deploy_app` no longer advertises `force`.

## [v0.4.3] - 2026-07-02

### Fixed
- Release pipeline fixes for v0.4.2 and v0.4.3: MCP registry namespace and description, goreleaser-action v7, `golang.org/x/net` v0.55.0.

## [v0.4.1] - 2026-07-02

### Changed
- Release pipeline migrated to GoReleaser; releases are gated on Trivy scans and published to the MCP registry.

### Security
- Go toolchain 1.25.11 and patched dependencies (including `golang.org/x/net`); fixed the CORS `Access-Control-Max-Age` header.

## [v0.3.0] - 2025-12-16

### Added
- Deployment build log resources and the log streaming design (see `docs/decisions/001-log-streaming-strategy.md`).

### Changed
- Compatibility test matrix extended through Dokku v0.38.x.

## [v0.2.2] - 2025-12-13

### Added
- **CORS Configuration**: Optional CORS middleware for SSE transport
  - Configurable allowed origins (supports wildcards like `*.example.com`)
  - Configurable allowed methods and headers
  - Preflight request handling
  - Disabled by default (uses mcp-go's `Access-Control-Allow-Origin: *`)
  - See `docs/CORS.md` for configuration details and security best practices

### Changed
- **mcp-go**: Updated from v0.43.0 to v0.43.2
  - Improved SSE server stability
  - Enhanced streaming support

### Documentation
- Added `docs/CORS.md` with security analysis and configuration guide
- Updated `config.yaml.example` with CORS configuration examples
- Updated README.md with CORS section under Transport Modes

## [v0.2.1] - 2025-11-06

### Changed
- **Go Runtime Upgrade**: Updated from Go 1.24.6 to Go 1.25.4
- **Dependencies**: Upgraded all Go dependencies to latest versions
  - `github.com/mark3labs/mcp-go` v0.41.1 → v0.43.0
  - `github.com/spf13/viper` v1.20.1 → v1.21.0
  - `go.uber.org/zap` v1.26.0 → v1.27.0
  - `github.com/mailru/easyjson` v0.7.7 → v0.9.1
  - Plus several other minor/patch updates

### Updated
- **CI/CD**: Updated all GitHub Actions workflows to use Go 1.25
- **Docker**: Updated Dockerfile base image to `golang:1.25-alpine`
- **Documentation**: Updated README.md to require Go 1.25 or later
- **Scripts**: Updated development scripts to reference Go 1.25

### Technical
- All tests continue to pass with 22.0% coverage
- Project builds successfully with Go 1.25.4
- Development tools updated to support Go 1.25

## [v0.2.0] - Previous Release

For changes in v0.2.0 and earlier, please see the git commit history.