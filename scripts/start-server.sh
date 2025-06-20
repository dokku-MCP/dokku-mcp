#!/bin/bash

# Script de démarrage pour le serveur MCP Dokku
# Ce script configure l'environnement et démarre le serveur

set -e

# Couleurs pour l'affichage
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration par défaut
DOKKU_MCP_HOST="${DOKKU_MCP_HOST:-localhost}"
DOKKU_MCP_PORT="${DOKKU_MCP_PORT:-8080}"
DOKKU_MCP_LOG_LEVEL="${DOKKU_MCP_LOG_LEVEL:-info}"
DOKKU_MCP_LOG_FORMAT="${DOKKU_MCP_LOG_FORMAT:-json}"
DOKKU_MCP_DOKKU_PATH="${DOKKU_MCP_DOKKU_PATH:-/usr/bin/dokku}"

# Fonction d'affichage
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERREUR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCÈS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[AVERTISSEMENT]${NC} $1"
}

# Vérifier si Go est installé
check_go() {
    if ! command -v go &> /dev/null; then
        error "Go n'est pas installé. Veuillez installer Go 1.24 ou plus récent."
        exit 1
    fi
    
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    MINOR=$(echo $GO_VERSION | cut -d. -f2)
    
    if [ "$MAJOR" -lt 1 ] || ([ "$MAJOR" -eq 1 ] && [ "$MINOR" -lt 21 ]); then
        error "Go 1.24 ou plus récent est requis. Version installée: $GO_VERSION"
        exit 1
    fi
    
    success "Go $GO_VERSION détecté"
}

# Vérifier si Dokku est installé (optionnel)
check_dokku() {
    if [ -f "$DOKKU_MCP_DOKKU_PATH" ]; then
        DOKKU_VERSION=$($DOKKU_MCP_DOKKU_PATH version 2>/dev/null | head -n1 || echo "inconnu")
        success "Dokku détecté: $DOKKU_VERSION"
    else
        warn "Dokku non trouvé à $DOKKU_MCP_DOKKU_PATH (fonctionnera en mode simulation)"
    fi
}

# Construire le serveur si nécessaire
build_server() {
    if [ ! -f "./build/dokku-mcp" ] || [ "$(find . -name '*.go' -newer ./build/dokku-mcp 2>/dev/null)" ]; then
        log "Construction du serveur..."
        make build
        success "Serveur construit avec succès"
    else
        log "Already built and up-to-date server"
    fi
}

# Créer le fichier de configuration s'il n'existe pas
create_config() {
    if [ ! -f "config.yaml" ]; then
        log "Création du fichier de configuration par défaut..."
        cat > config.yaml << EOF
# Configuration du serveur MCP Dokku
host: "$DOKKU_MCP_HOST"
port: $DOKKU_MCP_PORT
log_level: "$DOKKU_MCP_LOG_LEVEL"
log_format: "$DOKKU_MCP_LOG_FORMAT"
timeout: "30s"

# Configuration Dokku
dokku_path: "$DOKKU_MCP_DOKKU_PATH"

# Configuration du cache
cache_enabled: true
cache_ttl: "5m"

# Configuration de sécurité
security:
  allowed_commands:
    - "apps:list"
    - "apps:info"
    - "apps:create"
    - "apps:exists"
    - "config:get"
    - "config:set"
    - "config:show"
    - "ps:scale"
    - "logs"
    - "buildpacks:set"
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
  audit:
    enabled: true
    log_file: "/tmp/dokku-mcp-audit.log"
EOF
        success "Fichier de configuration créé: config.yaml"
    else
        log "Fichier de configuration existant trouvé"
    fi
}

# Afficher les informations de démarrage
show_startup_info() {
    log "Configuration du serveur MCP Dokku:"
    echo "  Hôte: $DOKKU_MCP_HOST"
    echo "  Port: $DOKKU_MCP_PORT"
    echo "  Niveau de log: $DOKKU_MCP_LOG_LEVEL"
    echo "  Format de log: $DOKKU_MCP_LOG_FORMAT"
    echo "  Chemin Dokku: $DOKKU_MCP_DOKKU_PATH"
    echo ""
}

# Démarrer le serveur
start_server() {
    log "Démarrage du serveur MCP Dokku..."
    
    # Exporter les variables d'environnement
    export DOKKU_MCP_HOST
    export DOKKU_MCP_PORT
    export DOKKU_MCP_LOG_LEVEL
    export DOKKU_MCP_LOG_FORMAT
    export DOKKU_MCP_DOKKU_PATH
    
    # Démarrer le serveur
    exec ./build/dokku-mcp "$@"
}

# Gestion des signaux pour arrêt propre
cleanup() {
    log "Arrêt du serveur..."
    exit 0
}

trap cleanup SIGINT SIGTERM

# Fonction d'aide
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Script de démarrage pour le serveur MCP Dokku"
    echo ""
    echo "Variables d'environnement:"
    echo "  DOKKU_MCP_HOST         Hôte du serveur (défaut: localhost)"
    echo "  DOKKU_MCP_PORT         Port du serveur (défaut: 8080)"
    echo "  DOKKU_MCP_LOG_LEVEL    Niveau de log (défaut: info)"
    echo "  DOKKU_MCP_LOG_FORMAT   Format de log (défaut: json)"
    echo "  DOKKU_MCP_DOKKU_PATH   Chemin vers Dokku (défaut: /usr/bin/dokku)"
    echo ""
    echo "Options:"
    echo "  -h, --help             Afficher cette aide"
    echo "  --no-build             Ne pas reconstruire si nécessaire"
    echo "  --debug                Activer le mode debug"
    echo ""
    echo "Exemples:"
    echo "  $0                     Démarrer avec la configuration par défaut"
    echo "  $0 --debug             Démarrer en mode debug"
    echo "  DOKKU_MCP_PORT=9090 $0 Démarrer sur le port 9090"
}

# Parse des arguments
NO_BUILD=false
DEBUG=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --no-build)
            NO_BUILD=true
            shift
            ;;
        --debug)
            DEBUG=true
            DOKKU_MCP_LOG_LEVEL="debug"
            shift
            ;;
        *)
            # Passer les arguments restants au serveur
            break
            ;;
    esac
done

# Affichage de l'en-tête
echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║              Serveur MCP Dokku                 ║${NC}"
echo -e "${GREEN}║              Powered by mcp-go                 ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
echo ""

# Vérifications
check_go
check_dokku

# Construction si nécessaire
if [ "$NO_BUILD" = false ]; then
    build_server
fi

# Création de la configuration
create_config

# Affichage des informations
show_startup_info

# Démarrage du serveur
start_server "$@" 