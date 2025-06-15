#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${GREEN}🚀 Setting up Dokku MCP development environment...${NC}"

echo -e "${YELLOW}📝 Configuring Git hooks...${NC}"
git config core.hooksPath .githooks
echo -e "${GREEN}✅ Git hooks configured${NC}"

echo -e "${YELLOW}🔧 Making hooks executable...${NC}"
chmod +x .githooks/*
echo -e "${GREEN}✅ Hooks are executable${NC}"

echo -e "${YELLOW}🛠️  Installing Go development tools...${NC}"
if ! command -v goimports >/dev/null 2>&1; then
    go install golang.org/x/tools/cmd/goimports@latest
    echo -e "${GREEN}✅ goimports installed${NC}"
else
    echo -e "${GREEN}✅ goimports already installed${NC}"
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
    echo -e "${YELLOW}📦 Installing golangci-lint...${NC}"
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    echo -e "${GREEN}✅ golangci-lint installed${NC}"
else
    echo -e "${GREEN}✅ golangci-lint already installed${NC}"
fi

echo -e "${YELLOW}🧪 Testing pre-commit hook...${NC}"
if [ -f .githooks/pre-commit ]; then
    echo -e "${GREEN}✅ Pre-commit hook is ready${NC}"
else
    echo -e "${RED}❌ Pre-commit hook not found${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 Development environment setup complete!${NC}"
echo -e "${YELLOW}💡 You can now commit with autochecks${NC}" 