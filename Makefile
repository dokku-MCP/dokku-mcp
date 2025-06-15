BINARY_NAME=dokku-mcp
VERSION?=v0.1.0
ENTRYPOINT=src/interface/cmd/main.go
BUILD_DIR=build
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m

# Include CI/CD specific targets
include Makefile.ci

all: help ## This help

setup-dev: ## Setup development environment
	@printf "$(GREEN)🚀 Setting up development environment...$(NC)\n"
	./scripts/setup-dev.sh

help: ## Show this help
	@printf "$(GREEN)Dokku MCP Server - Development Commands$(NC)\n"
	@printf "\n"
	@printf "$(YELLOW)Development Commands:$(NC)\n"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' Makefile
	@printf "\n"
	@printf "$(YELLOW)CI/CD Commands:$(NC)\n"
	@printf "  $(GREEN)make ci-help$(NC)      Show CI/CD specific commands\n"
	@printf "  $(GREEN)make test-pr$(NC)      Run Pull Request test suite\n"
	@printf "  $(GREEN)make test-main$(NC)    Run main branch test suite\n"
	@printf "  $(GREEN)make test-release$(NC)  Run release test suite\n"

check: ## Run all quality checks
	@printf "$(GREEN)🔍 Run all quality checks...$(NC)\n"
	-@$(MAKE) --no-print-directory fmt
	-@$(MAKE) --no-print-directory vet
	-@$(MAKE) --no-print-directory lint
	-@$(MAKE) --no-print-directory cyclo
	-@$(MAKE) --no-print-directory dupl
	-@$(MAKE) --no-print-directory security-test
	@printf "$(GREEN)✅ All quality checks completed successfully!$(NC)\n"

build: ## Build the MCP server
	@printf "$(GREEN)📦 Building MCP server...$(NC)\n"
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(ENTRYPOINT)

build-all: ## Build for all platforms
	@printf "$(GREEN)📦 Multi-platform build...$(NC)\n"
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(ENTRYPOINT)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(ENTRYPOINT)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(ENTRYPOINT)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(ENTRYPOINT)

install-ginkgo: ## Install Ginkgo and Gomega
	@printf "$(GREEN)🔧 Installing Ginkgo and Gomega...$(NC)\n"
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	go mod tidy

# Development Testing Commands
test: install-ginkgo ## Run unit tests with Ginkgo
	@printf "$(GREEN)🧪 Running unit tests with Ginkgo...$(NC)\n"
	ginkgo -p -r --timeout=2m --flake-attempts=2 --randomize-all --poll-progress-after=10s src/domain/ src/application/ src/interface/

test-verbose: install-ginkgo ## Run unit tests with verbose output
	@printf "$(GREEN)🧪 Running unit tests (verbose)...$(NC)\n"
	ginkgo -v -r --timeout=2m --flake-attempts=2 --randomize-all --poll-progress-after=10s src/domain/ src/application/ src/interface/

test-focus: install-ginkgo ## Run focused unit tests
	@printf "$(GREEN)🧪 Running focused unit tests...$(NC)\n"
	ginkgo -focus="$(FOCUS)" -r --timeout=2m --flake-attempts=2 --randomize-all --poll-progress-after=10s src/domain/ src/application/ src/interface/

test-watch: install-ginkgo ## Run unit tests with watch mode
	@printf "$(GREEN)👀 Watching unit tests...$(NC)\n"
	ginkgo watch -r --timeout=2m --flake-attempts=2 --randomize-all --poll-progress-after=10s src/domain/ src/application/ src/interface/

test-coverage: install-ginkgo ## Generate test coverage report with Ginkgo
	@printf "$(GREEN)📊 Tests with coverage (Ginkgo)...$(NC)\n"
	ginkgo -p -r --coverprofile=coverage.out --timeout=2m --flake-attempts=2 --randomize-all --poll-progress-after=10s src/domain/ src/application/ src/interface/
	go tool cover -html=coverage.out -o coverage.html
	@printf "$(YELLOW)📄 Coverage report: coverage.html$(NC)\n"

test-infrastructure-unit: install-ginkgo ## Run infrastructure unit tests (non-integration)
	@printf "$(GREEN)🧪 Running infrastructure unit tests...$(NC)\n"
	ginkgo -p -r --timeout=2m --flake-attempts=2 --randomize-all --poll-progress-after=10s --skip-file="*_ginkgo_test.go" src/infrastructure/

# Integration Testing Commands  
test-integration: install-ginkgo ## Run integration tests with Ginkgo
	@printf "$(GREEN)🧪 Running integration tests with Ginkgo...$(NC)\n"
	DOKKU_MCP_INTEGRATION_TESTS=1 ginkgo -p -tags=integration --timeout=5m --flake-attempts=2 --randomize-all --poll-progress-after=15s ./src/infrastructure/dokku/

test-integration-verbose: install-ginkgo ## Run integration tests with verbose output
	@printf "$(GREEN)🧪 Running integration tests (verbose) with Ginkgo...$(NC)\n"
	DOKKU_MCP_INTEGRATION_TESTS=1 ginkgo -v -tags=integration --timeout=5m --flake-attempts=2 --randomize-all --poll-progress-after=15s ./src/infrastructure/dokku/

test-integration-focus: install-ginkgo ## Run integration tests with focus
	@printf "$(GREEN)🧪 Running integration tests with focus...$(NC)\n"
	DOKKU_MCP_INTEGRATION_TESTS=1 ginkgo -focus="$(FOCUS)" -tags=integration --timeout=5m --flake-attempts=2 --randomize-all --poll-progress-after=15s ./src/infrastructure/dokku/

test-integration-watch: install-ginkgo ## Run integration tests with watch
	@printf "$(GREEN)👀 Watching integration tests...$(NC)\n"
	DOKKU_MCP_INTEGRATION_TESTS=1 ginkgo watch -tags=integration --timeout=5m --flake-attempts=2 --randomize-all --poll-progress-after=15s ./src/infrastructure/dokku/

test-integration-clean: ## Clean up integration test applications
	@printf "$(GREEN)🧹 Cleaning up integration test applications...$(NC)\n"
	@./scripts/cleanup-test-apps.sh

test-integration-bench: install-ginkgo ## Run integration tests with benchmarks
	@printf "$(GREEN)⚡ Running integration benchmarks...$(NC)\n"
	DOKKU_MCP_INTEGRATION_TESTS=1 ginkgo -focus="Performance" -tags=integration --timeout=10m --flake-attempts=1 --randomize-all --poll-progress-after=30s ./src/infrastructure/dokku/

test-race: install-ginkgo ## Run tests with race detector
	@printf "$(GREEN)🏁 Tests with race detector...$(NC)\n"
	ginkgo -race -r --timeout=3m --flake-attempts=2 --randomize-all --poll-progress-after=10s src/domain/ src/application/ src/interface/

test-all-unit: test test-infrastructure-unit ## Run all unit tests (domain + application + interface + infrastructure unit)
	@printf "$(GREEN)✅ All unit tests completed!$(NC)\n"

test-all: test-all-unit test-integration ## Run all tests (unit + integration)
	@printf "$(GREEN)✅ All tests completed!$(NC)\n"

test-regression: ## Run all regression tests
	@printf "$(GREEN)🔍 Regression tests...$(NC)\n"
	make test
	make test-integration
	make lint
	make security-test

test-resources: ## Test MCP resources
	@printf "$(GREEN)🔌 Testing MCP resources...$(NC)\n"
	go run cmd/test-client/main.go --list-resources

lint: ## Check code style
	@printf "$(GREEN)🔍 Linting code...$(NC)\n"
	golangci-lint run ./...

fmt: ## Format code
	@printf "$(GREEN)✨ Formatting code...$(NC)\n"
	go fmt ./...
	goimports -w .

vet: ## Static code analysis
	@printf "$(GREEN)🔎 Static analysis...$(NC)\n"
	go vet ./...

cyclo: ## Check cyclomatic complexity
	@printf "$(GREEN)📊 Cyclomatic complexity...$(NC)\n"
	gocyclo -over 20 $$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*")

dupl: ## Detect duplicate code
	@printf "$(GREEN)👯 Duplicate code detection...$(NC)\n"
	dupl -threshold 50 $$(find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" -not -path "./.git/*" -not -path "./build/*")

security-test: ## Run security tests
	@printf "$(GREEN)🔒 Security tests...$(NC)\n"
	gosec ./...

install-tools: install-ginkgo ## Install development tools
	@printf "$(GREEN)🔧 Installing development tools...$(NC)\n"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install github.com/mibk/dupl@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/golang/mock/mockgen@latest

# Documentation and Utilities
docs: ## Generate documentation
	@printf "$(GREEN)📚 Generating documentation...$(NC)\n"
	go doc -all ./... > docs/api.txt
	godoc -http=:6060 &
	@printf "$(YELLOW)📖 Documentation available at http://localhost:6060$(NC)\n"

profile: ## Profile performance
	@printf "$(GREEN)📊 Profiling...$(NC)\n"
	go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
	@printf "$(YELLOW)📊 Profiles: cpu.prof, mem.prof$(NC)\n"

debug: ## Run in debug mode
	@printf "$(GREEN)🐛 Debug mode...$(NC)\n"
	DOKKU_MCP_LOG_LEVEL=debug go run $(ENTRYPOINT)

bump-version: ## Update version
	@printf "$(GREEN)🔖 Version: $(VERSION)$(NC)\n"
	@sed -i 's/Version = ".*"/Version = "$(VERSION)"/' internal/version/version.go

changelog: ## Generate changelog
	@printf "$(GREEN)📝 Generating changelog...$(NC)\n"
	git log --oneline --decorate --graph > CHANGELOG.md

clean: ## Clean generated files
	@printf "$(GREEN)🧹 Cleaning...$(NC)\n"
	rm -rf $(BUILD_DIR)/
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof

generate: ## Generate code
	@printf "$(GREEN)⚙️  Generating code...$(NC)\n"
	go generate ./...

dokku-setup: ## Setup local Dokku instance via Docker
	@printf "$(GREEN)🐳 Setting up local Dokku instance...$(NC)\n"
	./scripts/setup-dokku-local.sh

dokku-start: ## Start local Dokku instance
	@printf "$(GREEN)🚀 Starting local Dokku instance...$(NC)\n"
	docker-compose up -d
	@printf "$(YELLOW)⏳ Waiting for Dokku to be ready...$(NC)\n"
	sleep 15
	@if docker exec dokku-mcp-dev dokku version &>/dev/null; then \
		printf "$(GREEN)✅ Dokku is ready!$(NC)\n"; \
	else \
		printf "$(RED)❌ Dokku failed to start properly$(NC)\n"; \
		docker-compose logs; \
	fi

dokku-stop: ## Stop local Dokku instance
	@printf "$(GREEN)🛑 Stopping local Dokku instance...$(NC)\n"
	docker-compose down

dokku-status: ## Check local Dokku instance status
	@printf "$(GREEN)📊 Dokku instance status...$(NC)\n"
	@if docker ps | grep -q dokku-mcp-dev; then \
		printf "$(GREEN)✅ Dokku container is running$(NC)\n"; \
		printf "$(YELLOW)Version:$(NC) "; \
		docker exec dokku-mcp-dev dokku version 2>/dev/null || echo "N/A"; \
		printf "$(YELLOW)Applications:$(NC)\n"; \
		docker exec dokku-mcp-dev dokku apps:list 2>/dev/null || echo "  No applications"; \
	else \
		printf "$(RED)❌ Dokku container is not running$(NC)\n"; \
	fi

dokku-logs: ## View local Dokku logs
	@printf "$(GREEN)📄 Dokku logs...$(NC)\n"
	docker-compose logs -f

dokku-shell: ## Access Dokku container shell
	@printf "$(GREEN)🐚 Accessing Dokku shell...$(NC)\n"
	docker exec -it dokku-mcp-dev bash

dokku-clean: ## Clean local Dokku data and containers
	@printf "$(GREEN)🧹 Cleaning local Dokku...$(NC)\n"
	@if [ -f "./scripts/cleanup-test-apps.sh" ]; then \
		./scripts/cleanup-test-apps.sh; \
	fi
	docker-compose down -v
	@if [ -d "docker-data" ]; then \
		printf "$(YELLOW)⚠️  Removing docker-data directory...$(NC)\n"; \
		sudo rm -rf docker-data || rm -rf docker-data; \
	fi
	@printf "$(GREEN)✅ Complete cleanup finished$(NC)\n"

test-integration-local: dokku-start ## Run integration tests with local Dokku
	@printf "$(GREEN)🧪 Running integration tests with local Dokku...$(NC)\n"
	@if [ -f ".env.dokku-local" ]; then \
		set -a && source .env.dokku-local && set +a && \
		ginkgo -v -tags=integration --timeout=5m --flake-attempts=2 --randomize-all --poll-progress-after=15s ./src/infrastructure/dokku/; \
	else \
		printf "$(RED)❌ .env.dokku-local not found. Run 'make dokku-setup' first$(NC)\n"; \
		exit 1; \
	fi

test-local-env: ## Test with local Dokku environment
	@printf "$(GREEN)🧪 Testing with local environment...$(NC)\n"
	@if [ -f ".env.dokku-local" ]; then \
		set -a && source .env.dokku-local && set +a && \
		make test-integration; \
	else \
		printf "$(YELLOW)⚠️  Using mock environment$(NC)\n"; \
		DOKKU_MCP_USE_MOCK=true make test-integration; \
	fi

inspect: build ## Inspect the MCP server
	@printf "$(GREEN)🔍 Inspecting MCP server...$(NC)\n"
	npx @modelcontextprotocol/inspector ./build/dokku-mcp

.DEFAULT_GOAL := help

.PHONY: all build test clean install-tools setup-hooks lint fmt vet install-ginkgo
.PHONY: test-integration test-integration-verbose test-integration-focus test-integration-watch test-integration-clean test-integration-bench test-all
.PHONY: test-verbose test-focus test-watch test-race test-coverage test-regression
.PHONY: dokku-setup dokku-start dokku-stop dokku-status dokku-logs dokku-shell dokku-clean test-integration-local test-local-env
