#!/bin/bash

# Script d'exécution des tests d'intégration Dokku MCP
# Ce script configure l'environnement et exécute les tests d'intégration

set -e

# Couleurs pour l'affichage
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOKKU_PATH="${DOKKU_PATH:-/usr/bin/dokku}"
TEST_TIMEOUT="${TEST_TIMEOUT:-10m}"
VERBOSE="${VERBOSE:-false}"

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

# Fonction d'aide
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Script d'exécution des tests d'intégration pour le serveur MCP Dokku"
    echo ""
    echo "Options:"
    echo "  -h, --help              Afficher cette aide"
    echo "  -v, --verbose           Mode verbeux"
    echo "  -t, --timeout DURATION  Timeout pour les tests (défaut: 10m)"
    echo "  --dokku-path PATH       Chemin vers l'exécutable Dokku (défaut: /usr/bin/dokku)"
    echo "  --skip-checks           Ignorer les vérifications préalables"
    echo "  --cleanup-only          Nettoyer uniquement les applications de test"
    echo "  --bench                 Exécuter les benchmarks également"
    echo ""
    echo "Variables d'environnement:"
    echo "  DOKKU_PATH              Chemin vers Dokku"
    echo "  TEST_TIMEOUT            Timeout pour les tests"
    echo "  VERBOSE                 Mode verbeux (true/false)"
    echo ""
    echo "Exemples:"
    echo "  $0                      Exécuter les tests normalement"
    echo "  $0 --verbose            Exécuter avec logs détaillés"
    echo "  $0 --cleanup-only       Nettoyer uniquement"
    echo "  $0 --bench              Inclure les benchmarks"
}

# Vérifier les prérequis
check_prerequisites() {
    log "Vérification des prérequis..."
    
    # Vérifier Go
    if ! command -v go &> /dev/null; then
        error "Go n'est pas installé ou non disponible dans le PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    log "Go version détectée: $GO_VERSION"
    
    # Vérifier Dokku
    if [ ! -f "$DOKKU_PATH" ]; then
        error "Dokku non trouvé à $DOKKU_PATH"
        echo "Options:"
        echo "  1. Installer Dokku"
        echo "  2. Spécifier le bon chemin avec --dokku-path"
        echo "  3. Utiliser --skip-checks pour ignorer cette vérification"
        exit 1
    fi
    
    DOKKU_VERSION=$($DOKKU_PATH version 2>/dev/null | head -n1 || echo "Version inconnue")
    success "Dokku détecté: $DOKKU_VERSION"
    
    # Vérifier les permissions
    if [ ! -x "$DOKKU_PATH" ]; then
        error "Dokku n'est pas exécutable à $DOKKU_PATH"
        exit 1
    fi
    
    # Vérifier si l'utilisateur peut exécuter des commandes Dokku
    if ! $DOKKU_PATH apps:list &>/dev/null; then
        warn "Impossible d'exécuter 'dokku apps:list' - vérifiez les permissions"
        warn "Les tests pourraient échouer si vous n'avez pas les permissions appropriées"
    fi
}

# Nettoyer les applications de test
cleanup_test_apps() {
    log "Nettoyage des applications de test..."
    
    # Récupérer la liste des applications
    if ! APPS=$($DOKKU_PATH apps:list 2>/dev/null); then
        warn "Impossible de récupérer la liste des applications"
        return 0
    fi
    
    # Nettoyer les applications de test
    local cleaned=0
    while IFS= read -r app; do
        # Ignorer les lignes vides et les en-têtes
        if [[ "$app" =~ ^dokku-mcp-test ]]; then
            log "Suppression de l'application de test: $app"
            if $DOKKU_PATH apps:destroy "$app" --force &>/dev/null; then
                ((cleaned++))
            else
                warn "Échec de suppression de $app"
            fi
        fi
    done <<< "$APPS"
    
    if [ $cleaned -gt 0 ]; then
        success "Nettoyé $cleaned application(s) de test"
    else
        log "Aucune application de test à nettoyer"
    fi
}

# Exécuter les tests d'intégration
run_integration_tests() {
    log "Exécution des tests d'intégration..."
    
    # Préparer les arguments de test
    local test_args=()
    test_args+=("-tags=integration")
    test_args+=("-timeout=$TEST_TIMEOUT")
    
    if [ "$VERBOSE" = "true" ]; then
        test_args+=("-v")
    fi
    
    # Ajouter le package de test
    test_args+=("./internal/infrastructure/dokku/")
    
    # Variables d'environnement pour les tests
    export DOKKU_MCP_INTEGRATION_TESTS=1
    export DOKKU_PATH="$DOKKU_PATH"
    
    log "Commande de test: go test ${test_args[*]}"
    
    # Exécuter les tests
    if go test "${test_args[@]}"; then
        success "Tests d'intégration réussis!"
        return 0
    else
        error "Échec des tests d'intégration"
        return 1
    fi
}

# Exécuter les benchmarks
run_benchmarks() {
    log "Exécution des benchmarks d'intégration..."
    
    export DOKKU_MCP_INTEGRATION_TESTS=1
    export DOKKU_PATH="$DOKKU_PATH"
    
    local bench_args=()
    bench_args+=("-tags=integration")
    bench_args+=("-bench=.")
    bench_args+=("-benchmem")
    bench_args+=("-timeout=$TEST_TIMEOUT")
    
    if [ "$VERBOSE" = "true" ]; then
        bench_args+=("-v")
    fi
    
    bench_args+=("./internal/infrastructure/dokku/")
    
    log "Commande de benchmark: go test ${bench_args[*]}"
    
    if go test "${bench_args[@]}"; then
        success "Benchmarks terminés!"
        return 0
    else
        error "Échec des benchmarks"
        return 1
    fi
}

# Variables par défaut
SKIP_CHECKS=false
CLEANUP_ONLY=false
RUN_BENCHMARKS=false

# Parse des arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -t|--timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        --dokku-path)
            DOKKU_PATH="$2"
            shift 2
            ;;
        --skip-checks)
            SKIP_CHECKS=true
            shift
            ;;
        --cleanup-only)
            CLEANUP_ONLY=true
            shift
            ;;
        --bench)
            RUN_BENCHMARKS=true
            shift
            ;;
        *)
            error "Option inconnue: $1"
            echo "Utilisez --help pour voir les options disponibles"
            exit 1
            ;;
    esac
done

# Affichage de l'en-tête
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}   Tests d'intégration Dokku MCP Server   ${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Configuration affichée
log "Configuration:"
echo "  Dokku Path: $DOKKU_PATH"
echo "  Test Timeout: $TEST_TIMEOUT"
echo "  Verbose: $VERBOSE"
echo "  Skip Checks: $SKIP_CHECKS"
echo "  Cleanup Only: $CLEANUP_ONLY"
echo "  Run Benchmarks: $RUN_BENCHMARKS"
echo ""

# Nettoyage si demandé
if [ "$CLEANUP_ONLY" = "true" ]; then
    cleanup_test_apps
    success "Nettoyage terminé"
    exit 0
fi

# Vérifications préalables
if [ "$SKIP_CHECKS" = "false" ]; then
    check_prerequisites
fi

# Nettoyage préalable
cleanup_test_apps

# Exécution des tests
log "Démarrage des tests d'intégration..."
echo ""

if run_integration_tests; then
    TEST_SUCCESS=true
else
    TEST_SUCCESS=false
fi

# Exécution des benchmarks si demandé
if [ "$RUN_BENCHMARKS" = "true" ]; then
    echo ""
    if run_benchmarks; then
        BENCH_SUCCESS=true
    else
        BENCH_SUCCESS=false
    fi
fi

# Nettoyage final
echo ""
cleanup_test_apps

# Résultats finaux
echo ""
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}              RÉSULTATS FINAUX             ${NC}"
echo -e "${BLUE}============================================${NC}"

if [ "$TEST_SUCCESS" = "true" ]; then
    success "✅ Tests d'intégration: SUCCÈS"
else
    error "❌ Tests d'intégration: ÉCHEC"
fi

if [ "$RUN_BENCHMARKS" = "true" ]; then
    if [ "$BENCH_SUCCESS" = "true" ]; then
        success "✅ Benchmarks: SUCCÈS"
    else
        error "❌ Benchmarks: ÉCHEC"
    fi
fi

# Code de sortie
if [ "$TEST_SUCCESS" = "true" ] && { [ "$RUN_BENCHMARKS" = "false" ] || [ "$BENCH_SUCCESS" = "true" ]; }; then
    success "Tous les tests ont réussi!"
    exit 0
else
    error "Certains tests ont échoué"
    exit 1
fi 