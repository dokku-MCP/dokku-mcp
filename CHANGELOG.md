# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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