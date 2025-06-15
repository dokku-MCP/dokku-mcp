# Makefile Structure for Dokku MCP Server

## Overview

The project uses a **separated Makefile structure** to clearly distinguish between development and CI/CD commands:

- **`Makefile`** - Development-focused commands for local work
- **`Makefile.ci`** - CI/CD-specific commands with enhanced validation

## Architecture

```
Makefile (Development)
├── include Makefile.ci
├── Development Testing
├── Code Quality
├── Build Commands
└── Documentation

Makefile.ci (CI/CD)
├── CI Environment Validation
├── Strict Test Suites
├── Comprehensive Security
├── Performance Benchmarks
└── Debug & Maintenance
```

## Main Makefile (Development Commands)

### Purpose
- **Local development** workflow
- **Quick testing** and validation
- **Code quality** checks
- **Documentation** generation

### Key Commands

#### Development Testing
```bash
make test                 # Unit tests only
make test-verbose         # Verbose unit tests
make test-watch          # Watch mode for TDD
make test-coverage       # Coverage report
make test-integration    # Basic integration tests
```

#### Code Quality
```bash
make fmt                 # Format code
make lint                # Lint code
make vet                 # Static analysis
make cyclo              # Complexity check
make dupl               # Duplicate detection
```

#### Build & Tools
```bash
make build              # Single platform build
make build-all          # Multi-platform build
make install-tools      # Install dev dependencies
make clean              # Clean artifacts
```

## CI Makefile (CI/CD Commands)

### Purpose
- **Strict validation** for CI environments
- **Comprehensive testing** with real infrastructure
- **Security scanning** and compliance
- **Performance benchmarking**

### Key Commands

#### CI Test Suites
```bash
make -f Makefile.ci test-pr         # Pull Request suite
make -f Makefile.ci test-main       # Main branch suite  
make -f Makefile.ci test-release    # Release suite
```

#### Individual CI Tests
```bash
make -f Makefile.ci test-ci-unit                # Unit tests with strict coverage
make -f Makefile.ci test-ci-integration-mock    # Integration with mocks
make -f Makefile.ci test-ci-integration-real    # Integration with real Dokku
make -f Makefile.ci test-ci-quality             # Quality with strict validation
make -f Makefile.ci test-ci-security            # Comprehensive security
make -f Makefile.ci test-ci-performance         # Performance benchmarks
```

#### CI Environment Management
```bash
make -f Makefile.ci validate-ci-environment     # Check CI setup
make -f Makefile.ci debug-ci                    # Debug CI issues
make -f Makefile.ci cleanup-ci                  # Clean CI artifacts
make -f Makefile.ci ci-status                   # Environment status
make -f Makefile.ci ci-metrics                  # Collect metrics
```

#### Help & Discovery
```bash
make ci-help            # Show all CI commands
make -f Makefile.ci ci-help    # Alternative syntax
```

## Usage Patterns

### Local Development Workflow

```bash
# Start development
make setup-dev          # Initial setup
make install-tools      # Install dependencies

# Development cycle
make test-watch         # Start TDD cycle
make fmt                # Format before commit
make test-coverage      # Check coverage
make test-integration   # Validate integration

# Pre-commit validation
make lint               # Code quality
make vet                # Static analysis
make test               # Final test run
```

### CI/CD Workflow

#### Pull Request Pipeline
```bash
make -f Makefile.ci validate-ci-environment
make -f Makefile.ci test-pr
# Runs: unit tests, integration (mock), quality checks
```

#### Main Branch Pipeline
```bash
make -f Makefile.ci test-main
# Runs: unit tests, integration (real), quality, security
```

#### Release Pipeline
```bash
make -f Makefile.ci test-release
# Runs: regression tests, performance benchmarks
```

### GitHub Actions Integration

The workflows automatically use the appropriate Makefile:

```yaml
# CI Workflow
- name: Unit tests with coverage
  run: make -f Makefile.ci test-ci-unit

- name: Integration tests with mocks  
  run: make -f Makefile.ci test-ci-integration-mock
```

## Key Differences

### Development vs CI Testing

| Aspect | Development (`make test`) | CI (`make -f Makefile.ci test-ci-unit`) |
|--------|---------------------------|------------------------------------------|
| **Coverage** | Basic reporting | Strict 75% threshold enforcement |
| **Speed** | Fast feedback | Comprehensive validation |
| **Infrastructure** | Local only | Mock + Real Dokku options |
| **Error Handling** | Warnings | Hard failures |
| **Reporting** | Console output | SARIF, JSON, HTML reports |

### Quality Checks

| Check | Development | CI |
|-------|-------------|-----|
| **Formatting** | `make fmt` | Strict validation + diff check |
| **Linting** | Basic rules | All rules enforced |
| **Security** | Quick scan | SARIF reports + vulnerabilities |
| **Complexity** | Warnings | Hard limits enforced |

## Environment Variables

### Development
```bash
# Optional for development
FOCUS="TestName"        # Focus specific tests
VERBOSE=true           # Verbose output
```

### CI/CD
```bash
# Required for CI
DOKKU_MCP_INTEGRATION_TESTS=1
DOKKU_MCP_USE_MOCK=true/false
DOKKU_HOST=localhost
DOKKU_USER=dokku

# Performance tuning
MAX_TEST_APPS=20
CONCURRENT_TESTS=5
COMMAND_TIMEOUT=60s
```

## Benefits of Separation

### 🎯 **Clarity**
- **Developers** see only relevant commands
- **CI/CD** has specialized tooling
- **Purpose-driven** command grouping

### ⚡ **Performance**
- **Development** optimized for speed
- **CI/CD** optimized for thoroughness
- **Parallel execution** where possible

### 🔒 **Security**
- **CI-specific** security validation
- **Environment-aware** configurations
- **Artifact management** for compliance

### 🛠️ **Maintainability**
- **Logical separation** of concerns
- **Easy to extend** either workflow
- **Clear dependencies** and relationships

## Quick Reference

### Most Common Commands

**Development:**
```bash
make test              # Quick unit tests
make test-coverage     # Coverage report  
make fmt && make lint  # Code quality
```
