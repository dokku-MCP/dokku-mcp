BINARY_NAME=dokku-mcp
VERSION?=v0.1.0
ENTRYPOINT=cmd/server/main.go
BUILD_DIR=build
GO_SRC_DIRS=./cmd/... ./internal/... ./pkg/...
GO_SRC_PATHS=cmd internal pkg
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
BLUE=\033[0;34m
NC=\033[0m

# Add go bin directory to PATH for tools
GO_BIN := $(shell go env GOPATH)/bin
GINKGO_BINARY := $(GO_BIN)/ginkgo
export PATH := $(GO_BIN):$(PATH)

all: help

help: ## Show this help
	@printf "$(GREEN)Dokku MCP Server - Development Commands$(NC)\n"
	@printf "\n"
	@printf "$(YELLOW)Commands:$(NC)\n"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' Makefile
	@printf "\n"

dev: ## Run in development mode with live reload
	@mkdir -p tmp
	DOKKU_MCP_LOG_LEVEL=debug air

setup-dev: ## Setup development environment
	@printf "$(GREEN)🚀 Setting up development environment...$(NC)\n"
	./scripts/setup-dev.sh

setup-dokku: ## Setup local Dokku instance via Docker
	@printf "$(GREEN)🐳 Setting up local Dokku instance...$(NC)\n"
	./scripts/setup-dokku-local.sh

install-tools: ## Install development tools
	@printf "$(GREEN)🔧 Installing development tools...$(NC)\n"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install github.com/mibk/dupl@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/golang/mock/mockgen@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install golang.org/x/tools/cmd/godoc@latest
	go install github.com/air-verse/air@latest

check: ## Run all quality checks
	@printf "$(GREEN)🔍 Run all quality checks...$(NC)\n"
	-@$(MAKE) --no-print-directory fmt
	-@$(MAKE) --no-print-directory vet
	-@$(MAKE) --no-print-directory lint
	-@$(MAKE) --no-print-directory cyclo
	-@$(MAKE) --no-print-directory dupl
	-@$(MAKE) --no-print-directory _check-security
	@printf "$(GREEN)✅ All quality checks completed successfully!$(NC)\n"

start: build
	@printf "$(GREEN)== Starting MCP server ==$(NC)\n"
	./$(BUILD_DIR)/$(BINARY_NAME)

start-docker: build-docker
	@printf "$(GREEN)== Starting MCP server ==$(NC)\n"
	docker run dokku-mcp

inspect: ## Inspect the MCP server
	@printf "$(GREEN)🔍 Inspecting MCP server...$(NC)\n"
	npx @modelcontextprotocol/inspector

build: ## Build the MCP server
	@printf "$(GREEN)📦 Building MCP server...$(NC)\n"
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(ENTRYPOINT)

build-docker:
	@printf "$(GREEN)📦 Building MCP server docker container...$(NC)\n"
	docker build -t dokku-mcp .

build-all: ## Build for all platforms
	@printf "$(GREEN)📦 Multi-platform build...$(NC)\n"
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(ENTRYPOINT)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(ENTRYPOINT)
	GOOS=linux GOARCH=arm go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm $(ENTRYPOINT)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(ENTRYPOINT)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(ENTRYPOINT)

# Testing Commands
test: $(GINKGO_BINARY)
	$(GINKGO_BINARY) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: $(GINKGO_BINARY)
	$(GINKGO_BINARY) -race ./...

# Integration Testing Commands 
test-integration-local: dokku-start ## -experimental- Run integration tests with local Dokku
	@printf "$(GREEN)🧪 Running integration tests with local Dokku...$(NC)\n"
	@if [ -f ".env.dokku-local" ]; then \
		set -a && source .env.dokku-local && set +a && \
		DOKKU_MCP_LOG_LEVEL=error ginkgo -v -tags=integration --timeout=5m --flake-attempts=2 --randomize-all --poll-progress-after=15s internal/dokku-api/ | grep -v "time=.*level="; \
	else \
		printf "$(RED)❌ .env.dokku-local not found. Run 'make dokku-setup' first$(NC)\n"; \
		exit 1; \
	fi

lint: ## Check code style
	@printf "$(GREEN)🔍 Linting code...$(NC)\n"
	golangci-lint run $(GO_SRC_DIRS)

type: ## Check type safety
	@$(MAKE) --no-print-directory _check-type-safety

fmt: ## Format code
	@printf "$(GREEN)✨ Formatting code...$(NC)\n"
	go fmt $(GO_SRC_DIRS)
	goimports -w $(GO_SRC_PATHS)

vet: ## Static code analysis
	@printf "$(GREEN)🔎 Static analysis...$(NC)\n"
	go vet $(GO_SRC_DIRS)

cyclo: ## Check cyclomatic complexity
	@printf "$(GREEN)📊 Cyclomatic complexity...$(NC)\n"
	gocyclo -over 25 $$(find $(GO_SRC_PATHS) -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" 2>/dev/null || true)

dep-graph-dot: ## Generate Go import dependency graph in DOT format
	@printf "$(GREEN)📊 Generating Go import dependency graph (DOT)...$(NC)\n"
	mkdir -p build
	go run cmd/depgraph/main.go > build/dep-graph.dot
	@printf "$(YELLOW)✅ DOT graph: build/dep-graph.dot$(NC)\n"

dep-graph: dep-graph-dot ## Generate PNG + SVG from DOT dependency graph
	dot -Tpng build/dep-graph.dot -o build/dep-graph.png
	dot -Tsvg build/dep-graph.dot -o build/dep-graph.svg
	@printf "$(YELLOW)✅ PNG + SVG graph: build/dep-graph.{png|svg}$(NC)\n"

dupl: ## Detect duplicate code
	@printf "$(GREEN)👯 Duplicate code detection...$(NC)\n"
	dupl -threshold 70 $$(find $(GO_SRC_PATHS) -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" -not -path "./.git/*" -not -path "./build/*" 2>/dev/null || true)

security: ## Run security tests
	@$(MAKE) --no-print-directory _check-security-detailed

# Documentation and Utilities
docs: ## Generate documentation - not human friendly, for llm use
	@printf "$(GREEN)📚 Generating documentation - not human friendly, for llm use...$(NC)\n"
	@printf "$(YELLOW)📖 Documentation available at http://localhost:6060$(NC)\n"
	@godoc -http=:6060

bump-version: ## Update version
	@printf "$(GREEN)🔖 Version: $(VERSION)$(NC)\n"
	@sed -i 's/Version = ".*"/Version = "$(VERSION)"/' internal/version/version.go
	echo "Think about plugins version too"

changelog: ## Generate changelog
	@printf "$(GREEN)📝 Generating changelog...$(NC)\n"
	git log --oneline --decorate --graph > CHANGELOG.md

clean: ## Clean generated files
	@printf "$(GREEN)🧹 Cleaning...$(NC)\n"
	rm -rf $(BUILD_DIR)/
	rm -rf tmp/
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof
	rm -f build-errors.log

generate: ## Generate code
	@printf "$(GREEN)⚙️  Generating code...$(NC)\n"
	go generate $(GO_SRC_DIRS)

dokku-start: ## Start local Dokku instance
	@printf "$(GREEN)🚀 Starting local Dokku instance...$(NC)\n"
	docker compose up -d
	@printf "$(YELLOW)⏳ Waiting for Dokku to be ready...$(NC)\n"
	sleep 15
	@if docker exec dokku-mcp-dev dokku version &>/dev/null; then \
		printf "$(GREEN)✅ Dokku is ready!$(NC)\n"; \
	else \
		printf "$(RED)❌ Dokku failed to start properly$(NC)\n"; \
		docker compose logs; \
	fi

dokku-stop: ## Stop local Dokku instance
	@printf "$(GREEN)🛑 Stopping local Dokku instance...$(NC)\n"
	docker compose down

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
	docker compose logs -f

dokku-shell: ## Access Dokku container shell
	@printf "$(GREEN)🐚 Accessing Dokku shell...$(NC)\n"
	docker exec -it dokku-mcp-dev bash

dokku-clean: ## Clean local Dokku data and containers
	@printf "$(GREEN)🧹 Cleaning local Dokku...$(NC)\n"
	@if [ -f "./scripts/cleanup-test-apps.sh" ]; then \
		./scripts/cleanup-test-apps.sh; \
	fi
	docker compose down -v
	@if [ -d "docker-data" ]; then \
		printf "$(YELLOW)⚠️  Removing docker-data directory...$(NC)\n"; \
		sudo rm -rf docker-data || rm -rf docker-data; \
	fi
	@printf "$(GREEN)✅ Complete cleanup finished$(NC)\n"

.DEFAULT_GOAL := help

.PHONY: all build test clean install-tools lint fmt vet dev debug
.PHONY: test test-verbose test-integration-local test-all test-race
.PHONY: setup-dokku dokku-start dokku-stop dokku-status dokku-logs dokku-shell dokku-clean
.PHONY: security _check-security _check-type-safety _check-security-detailed docs

_check-type-safety:
	@printf "$(GREEN)🚫 Checking forbidden patterns (Strong Typing)...$(NC)\n"
	@VIOLATIONS_FOUND=false; \
	for file in $$(find $(GO_SRC_PATHS) -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" 2>/dev/null || true); do \
		if grep -n "interface{}" "$$file" 2>/dev/null | grep -v "//" | grep -v "// NOTE:" | grep -v "This is a valid exception" >/dev/null 2>&1; then \
			printf "$(RED)❌ Forbidden pattern 'interface{}' found in $$file$(NC)\n"; \
			grep -n "interface{}" "$$file" | grep -v "//" | grep -v "// NOTE:" | grep -v "This is a valid exception" | head -5; \
			VIOLATIONS_FOUND=true; \
		fi; \
		if grep -nE "(map\[[^]]*\]any|\\[\\]any|: any|\\(any\\)|any\\[|\\]any)" "$$file" 2>/dev/null | grep -v "// NOTE:" | grep -v "This is a valid exception" >/dev/null 2>&1; then \
			printf "$(RED)❌ Forbidden Go type 'any' found in $$file$(NC)\n"; \
			grep -nE "(map\[[^]]*\]any|\\[\\]any|: any|\\(any\\)|any\\[|\\]any)" "$$file" | grep -v "// NOTE:" | grep -v "This is a valid exception" | head -5; \
			VIOLATIONS_FOUND=true; \
		fi; \
		if grep -n "reflect\\." "$$file" 2>/dev/null | grep -v "//" >/dev/null 2>&1; then \
			printf "$(RED)❌ Forbidden pattern 'reflect.' found in $$file$(NC)\n"; \
			grep -n "reflect\\." "$$file" | grep -v "//" | head -5; \
			VIOLATIONS_FOUND=true; \
		fi; \
		if grep -n "unsafe\\." "$$file" 2>/dev/null | grep -v "//" >/dev/null 2>&1; then \
			printf "$(RED)❌ Forbidden pattern 'unsafe.' found in $$file$(NC)\n"; \
			grep -n "unsafe\\." "$$file" | grep -v "//" | head -5; \
			VIOLATIONS_FOUND=true; \
		fi; \
	done; \
	if [ "$$VIOLATIONS_FOUND" = true ]; then \
		printf "$(YELLOW)💡 Use strongly typed interfaces according to project rules$(NC)\n"; \
		printf "$(YELLOW)💡 Add '// NOTE: ... This is a valid exception' comment for library constraints$(NC)\n"; \
		exit 1; \
	fi; \
	printf "$(GREEN)  ✓ No forbidden patterns detected$(NC)\n"

_check-security:
	@printf "$(GREEN)🔒 Security analysis...$(NC)\n"
	@if command -v gosec >/dev/null 2>&1; then \
		printf "$(BLUE)  Running gosec security scanner...$(NC)\n"; \
		if gosec -quiet $(GO_SRC_DIRS) >/dev/null 2>&1; then \
			printf "$(GREEN)  ✓ Security analysis passed$(NC)\n"; \
		else \
			printf "$(YELLOW)  ⚠️  Potential security issues detected$(NC)\n"; \
			printf "$(YELLOW)  💡 Run 'make security' for detailed report$(NC)\n"; \
		fi; \
	else \
		printf "$(YELLOW)  ⏭️  gosec not installed, security analysis skipped$(NC)\n"; \
		printf "$(YELLOW)  💡 Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest$(NC)\n"; \
	fi

_check-security-detailed:
	@printf "$(GREEN)🔒 Detailed Security Analysis...$(NC)\n"
	@if command -v gosec >/dev/null 2>&1; then \
		printf "$(BLUE)  Running gosec security scanner with detailed output...$(NC)\n"; \
		echo ""; \
		if gosec -fmt=text -stdout -verbose $(GO_SRC_DIRS); then \
			printf "\n$(GREEN)✅ Security analysis passed - no issues found$(NC)\n"; \
		else \
			printf "\n$(RED)❌ Security issues detected above$(NC)\n"; \
			printf "$(YELLOW)💡 Review and fix the security issues listed above$(NC)\n"; \
			printf "$(YELLOW)💡 Use 'gosec --help' for more scanning options$(NC)\n"; \
			exit 1; \
		fi; \
	else \
		printf "$(RED)❌ gosec not installed$(NC)\n"; \
		printf "$(YELLOW)📦 Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest$(NC)\n"; \
		exit 1; \
	fi
