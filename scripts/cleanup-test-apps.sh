#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

printf "${GREEN}🧹 Cleaning test apps and configurations...${NC}\n"

if command -v dokku &> /dev/null; then
    printf "${YELLOW}🔍 Searching for test apps...${NC}\n"
    
    if dokku apps:list 2>/dev/null | grep -q "dokku-mcp-test"; then
        printf "${YELLOW}🗑️  Deleting test apps...${NC}\n"
        dokku apps:list | grep "dokku-mcp-test" | while read -r app; do
            printf "  Deleting app: $app\n"
            dokku apps:destroy "$app" --force 2>/dev/null || true
        done
    else
        printf "${GREEN}✅ No test apps found${NC}\n"
    fi
    
    printf "${YELLOW}🔑 Cleaning test SSH keys...${NC}\n"
    if dokku ssh-keys:list 2>/dev/null | grep -q "dokku-mcp-test"; then
        dokku ssh-keys:remove dokku-mcp-test 2>/dev/null || true
        printf "${GREEN}✅ Test SSH key removed from Dokku${NC}\n"
    fi
    
elif docker ps | grep -q dokku-mcp-dev; then
    printf "${YELLOW}🐳 Cleaning via Docker Dokku...${NC}\n"
    
    if docker exec dokku-mcp-dev dokku apps:list 2>/dev/null | grep -q "dokku-mcp-test"; then
        printf "${YELLOW}🗑️  Deleting test apps...${NC}\n"
        docker exec dokku-mcp-dev bash -c '
            dokku apps:list | grep "dokku-mcp-test" | while read -r app; do
                echo "  Deleting app: $app"
                dokku apps:destroy "$app" --force 2>/dev/null || true
            done
        '
    fi
    
    printf "${YELLOW}🔑 Cleaning test SSH keys in Dokku...${NC}\n"
    docker exec dokku-mcp-dev bash -c '
        if dokku ssh-keys:list 2>/dev/null | grep -q "dokku-mcp-test"; then
            dokku ssh-keys:remove dokku-mcp-test 2>/dev/null || true
            echo "✅ Test SSH key removed from Dokku"
        fi
    ' || true
    
else
    printf "${YELLOW}⚠️  Dokku is not available, cleaning partially...${NC}\n"
fi

printf "${YELLOW}🔧 Cleaning local SSH configuration...${NC}\n"
if [ -f ~/.ssh/config ] && grep -q "Host dokku.local" ~/.ssh/config; then
    printf "${YELLOW}📝 Deleting local SSH config dokku.local...${NC}\n"
    awk '
    /^Host dokku\.local$/ { skip=1; next }
    /^Host / && skip { skip=0 }
    !skip { print }
    ' ~/.ssh/config > ~/.ssh/config.tmp && mv ~/.ssh/config.tmp ~/.ssh/config
fi

DOKKU_SSH_KEY_PATH="$HOME/.ssh/dokku_mcp_test"
if [ -f "${DOKKU_SSH_KEY_PATH}" ] || [ -f "${DOKKU_SSH_KEY_PATH}.pub" ]; then
    printf "${YELLOW}🔑 Deleting test SSH keys...${NC}\n"
    rm -f "${DOKKU_SSH_KEY_PATH}" "${DOKKU_SSH_KEY_PATH}.pub" 2>/dev/null || true
    printf "${GREEN}✅ Local SSH keys deleted${NC}\n"
fi

if [ -f ".env.dokku-local" ]; then
    printf "${YELLOW}🌍 Deleting local env file...${NC}\n"
    rm -f .env.dokku-local
fi

printf "${GREEN}✅ Cleaning done!${NC}\n"
printf "${YELLOW}📝 Cleaning summary:${NC}\n"
printf "  • Test Dokku apps deleted\n"
printf "  • Test SSH keys deleted (Dokku and local)\n"
printf "  • Local SSH config deleted\n"
printf "  • Local env file deleted\n" 