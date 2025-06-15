#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

printf "${GREEN}🐳 Configuration de Dokku local via Docker...${NC}\n"

mkdir -p docker-data/dokku

if ! command -v docker &> /dev/null; then
    printf "${RED}❌ Docker n'est pas installé ou n'est pas accessible${NC}\n"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    printf "${RED}❌ Docker Compose n'est pas installé${NC}\n"
    exit 1
fi

printf "${YELLOW}🚀 Starting Dokku...${NC}\n"
docker-compose up -d

printf "${YELLOW}⏳ Waiting for Dokku to start...${NC}\n"
sleep 10

if ! docker ps | grep -q dokku-mcp-dev; then
    printf "${RED}❌ Dokku container is not running${NC}\n"
    docker-compose logs
    exit 1
fi

printf "${YELLOW}⏳ Dokku starting...${NC}\n"
timeout=60
counter=0
while [ $counter -lt $timeout ]; do
    if docker exec dokku-mcp-dev dokku version &>/dev/null; then
        break
    fi
    sleep 2
    counter=$((counter + 2))
done

if [ $counter -ge $timeout ]; then
    printf "${RED}❌ Timeout: Dokku n'est pas prêt après ${timeout}s${NC}\n"
    exit 1
fi

printf "${YELLOW}🔑 Configuring SSH keys for tests...${NC}\n"

DOKKU_SSH_KEY_PATH="$HOME/.ssh/dokku_mcp_test"
if [ ! -f "${DOKKU_SSH_KEY_PATH}" ]; then
    printf "${YELLOW}⚠️  Generating dedicated SSH key pair for Dokku MCP tests...${NC}\n"
    ssh-keygen -t rsa -b 4096 -f "${DOKKU_SSH_KEY_PATH}" -N "" -C "dokku-mcp-test@$(hostname)"
    printf "${GREEN}✅ SSH key created: ${DOKKU_SSH_KEY_PATH}${NC}\n"
else
    printf "${GREEN}✅ Using existing SSH key: ${DOKKU_SSH_KEY_PATH}${NC}\n"
fi

SSH_KEY=$(cat "${DOKKU_SSH_KEY_PATH}.pub")
docker exec dokku-mcp-dev bash -c "echo '$SSH_KEY' | dokku ssh-keys:add dokku-mcp-test"

printf "${YELLOW}🔧 Configuration SSH client...${NC}\n"
SSH_CONFIG_ENTRY="
Host dokku.local
  HostName 127.0.0.1
  Port 3022
  User dokku
  IdentityFile ${DOKKU_SSH_KEY_PATH}
  IdentitiesOnly yes
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
"

if grep -q "Host dokku.local" ~/.ssh/config 2>/dev/null; then
    printf "${YELLOW}⚠️  Removing existing dokku.local SSH configuration...${NC}\n"
    awk '
    /^Host dokku\.local$/ { skip=1; next }
    /^Host / && skip { skip=0 }
    !skip { print }
    ' ~/.ssh/config > ~/.ssh/config.tmp && mv ~/.ssh/config.tmp ~/.ssh/config
fi

echo "$SSH_CONFIG_ENTRY" >> ~/.ssh/config
chmod 600 ~/.ssh/config
printf "${GREEN}✅ SSH configuration updated${NC}\n"

printf "${YELLOW}🌍 env config...${NC}\n"
cat > .env.dokku-local << EOF
DOKKU_MCP_USE_MOCK=false
DOKKU_HOST=127.0.0.1
DOKKU_PORT=3022
DOKKU_USER=dokku
DOKKU_HOSTNAME=dokku.local
DOKKU_MCP_INTEGRATION_TESTS=1
EOF

printf "${GREEN}✅ Dokku local configured successfully!${NC}\n"
printf "${YELLOW}📝 Important information:${NC}\n"
printf "  • Dokku SSH: 127.0.0.1:3022\n"
printf "  • Dokku HTTP: http://127.0.0.1:8080\n"
printf "  • Dokku HTTPS: https://127.0.0.1:8443\n"
printf "  • Configuration SSH: ~/.ssh/config (Host dokku.local)\n"
printf "  • env: .env.dokku-local\n"
printf "\n"
printf "${YELLOW}🧪 To run integration tests with real Dokku:${NC}\n"
printf "  source .env.dokku-local\n"
printf "  make test-integration\n"
printf "\n"
printf "${YELLOW}🐛 To debug Dokku:${NC}\n"
printf "  docker exec -it dokku-mcp-dev bash\n"
printf "  docker-compose logs -f\n" 