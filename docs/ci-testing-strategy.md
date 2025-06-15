# CI Testing Strategy for Dokku MCP Server

## Overview

The CI testing strategy is organized into **3 main workflows** with different test levels depending on context:

### 1. Main Workflow (`.github/workflows/ci.yml`)
- **Trigger**: Pull Requests, Push to `main`/`develop`
- **Focus**: Fast validation and code quality

### 2. Release Workflow (`.github/workflows/release.yml`)
- **Triggers**: Tags `v*.*.*`, Releases
- **Focus**: Complete tests and production validation

### 3. Scheduled Workflow (`.github/workflows/schedule.yml`)
- **Triggers**: Daily (2h UTC), Manual
- **Focus**: Maintenance and continuous monitoring

## Strategy by Test Type

### Unit Tests 🧪

**What**: Validation of individual components without external dependencies

**When**: All workflows (high priority)

**Command**: `make test-coverage`

**Targets**:
- Business logic in `internal/domain/`
- Application handlers in `internal/application/`
- Data parsing and validation
- MCP transformation algorithms

**Success Criteria**:
- ✅ Minimum coverage: 75%
- ✅ All tests pass
- ✅ Cyclomatic complexity < 20

```bash
# Local example
make test-coverage
# Generates coverage.html with detailed report
```

### Integration Tests 🔌

**Two modes depending on context**:

#### Mock Mode (Pull Requests)
- **Objective**: Validate interactions without real infrastructure
- **Advantages**: Fast, deterministic, no dependencies
- **Configuration**: `DOKKU_MCP_USE_MOCK=true`

```bash
# Integration tests with mocks
export DOKKU_MCP_USE_MOCK=true
export DOKKU_MCP_INTEGRATION_TESTS=1
make test-integration-verbose
```

#### Real Mode (Main/Release)
- **Objective**: Validation with complete Dokku infrastructure
- **Configuration**: Automatic Dokku installation in CI
- **Cleanup**: Automatic via `make test-integration-clean`

```bash
# Integration tests with real Dokku
export DOKKU_MCP_USE_MOCK=false
export DOKKU_MCP_INTEGRATION_TESTS=1
export DOKKU_HOST=localhost
make test-integration-verbose
```

### Quality Tests 📊

**Code and security validation**:

```bash
# Formatting and style
make fmt && git diff --exit-code

# Static analysis
make vet

# Advanced linting
make lint

# Cyclomatic complexity (threshold: 20)
make cyclo

# Duplicate code detection
make dupl

# Security analysis
make security-test
```

**Forbidden patterns** (via pre-commit hooks):
- `interface{}` - Use strong types
- `any` - Use specific types  
- `reflect.` - Avoid reflection
- `unsafe.` - Forbid unsafe code

### Performance Tests ⚡

**Integration benchmarks**:
- Load tests with multiple applications
- Response time measurement
- Memory and CPU profiling
- Performance threshold validation

```bash
# Local benchmarks
make test-integration-bench

# With profiling
make profile
```

**Monitored metrics**:
- Response time < 5s for `GetHistory()`
- Stable memory usage
- No goroutine leaks

### Security Tests 🔒

**Automated analysis**:
- **gosec**: Vulnerability detection
- **nancy**: Vulnerable dependency scanning
- **staticcheck**: Deep static analysis

```bash
# Complete security audit
gosec -fmt sarif -out security-report.sarif ./...
go list -json -deps ./... | nancy sleuth
```

**Validations**:
- MCP input sanitization
- Authorized Dokku command validation
- Secure error handling (no internal exposure)

## Test Configuration by Environment

### CI Environment Variables

```bash
# Integration tests
DOKKU_MCP_INTEGRATION_TESTS=1
DOKKU_MCP_USE_MOCK=true/false

# Dokku configuration
DOKKU_HOST=localhost
DOKKU_USER=dokku
DOKKU_PATH=/usr/bin/dokku

# CI-adjusted timeouts
COMMAND_TIMEOUT=60s
APP_CREATE_TIMEOUT=5m
CLEANUP_TIMEOUT=10m

# Test prefixes
DOKKU_MCP_TEST_PREFIX=dokku-mcp-test
MAX_TEST_APPS=20
```

### CI Optimizations

**Parallelization**:
- Unit tests: `ginkgo -p` (parallel by default)
- Independent CI jobs when possible
- Go modules and build caching

**Timeout management**:
- Unit tests: 5 minutes max
- Integration tests: 15 minutes max
- Dokku installation: 10 minutes max

**Automatic cleanup**:
- Test applications deleted even on failure (`if: always()`)
- Use of `t.Cleanup()` in Ginkgo tests
- Manual cleanup script: `make test-integration-clean`

## Failure Management and Debug

### Debug Logs

**CI with detailed logs**:
```bash
export DOKKU_MCP_LOG_LEVEL=debug
make test-integration-verbose
```

**Local tests with focus**:
```bash
# Test specific suite
export FOCUS="DeploymentService"
make test-integration-focus
```

### Preserved Artifacts

**Automatic result uploads**:
- Coverage reports → Codecov
- Performance profiles → GitHub Artifacts
- Security reports → GitHub Security tab
- Failure logs → Artifacts for debug

### Actions on Failure

**Pull Request**:
- ❌ Block merge if critical tests fail
- ⚠️ Warnings for high complexity/duplication
- 💬 Automatic comment with failure details

**Main/Release**:
- 🚨 Automatic issue creation on regression
- 📧 Team notification (if configured)
- 🔄 Possible automatic rollback

## Maintenance and Monitoring

### Daily Scheduled Tests

**Multi-version Dokku compatibility**:
- Tests with Dokku versions: v0.32.0, v0.33.0, v0.34.0, master  
- Maintained compatibility matrix
- Continue even if master version fails (`continue-on-error`)

**Dependency monitoring**:
- Available update detection
- New vulnerability scanning
- Automatic cache maintenance

### Metrics and Trends

**Automatic tracking**:
- Test execution time trends
- Success rate by test type
- Code coverage evolution
- Historical performance benchmarks

## Useful Commands

### Local Development

```bash
# Complete local suite (like CI)
make test-regression

# Fast tests during development  
make test-watch

# Debug specific test
export FOCUS="should deploy application"
make test-integration-focus

# Manual cleanup if needed
make test-integration-clean
```

### Manual CI

```bash
# Manually trigger scheduled workflow
gh workflow run schedule.yml

# View CI run logs
gh run view --log

# Re-run failed job
gh run rerun --failed-only
```

### CI Failure Debug

```bash
# Reproduce CI environment locally
export CI=true
export GITHUB_ACTIONS=true
export DOKKU_MCP_INTEGRATION_TESTS=1
make test-regression
```

## Recommendations for New Tests

### Adding Unit Tests
1. **Placement**: Same package as tested code with `_test.go` suffix
2. **Structure**: Use Ginkgo suites with `Describe`/`Context`/`It`
3. **Mocks**: Generate with `go generate` and mockgen
4. **Names**: Descriptive names following project rules

### Adding Integration Tests  
1. **Build tag**: `//go:build integration`
2. **Configuration**: Use `dokkutesting.TestConfig`
3. **Cleanup**: Always implement `t.Cleanup()`
4. **Conditions**: Skip if environment not suitable

### Adding Benchmarks
1. **Function**: `Benchmark` prefix + `*testing.B`
2. **Focus**: `ginkgo -focus="Performance"`
3. **Metrics**: `b.ReportAllocs()` for memory
4. **Parallel**: `b.RunParallel()` if applicable

This strategy ensures **complete coverage** while **optimizing execution times** and maintaining code **quality** and **security**. 