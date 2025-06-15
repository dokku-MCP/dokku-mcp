# Tests d'intégration avec Dokku

Ce document décrit l'utilisation des tests d'intégration pour le serveur MCP Dokku.

## Vue d'ensemble

Les tests d'intégration vérifient que le serveur MCP fonctionne correctement avec une vraie installation Dokku. Ces tests créent de vraies applications Dokku pour valider le comportement de la méthode `GetHistory` et d'autres fonctionnalités.

## Prérequis

### Installation Dokku

Les tests d'intégration nécessitent une installation Dokku fonctionnelle :

```bash
# Sur Ubuntu (exemple)
wget -O- https://raw.githubusercontent.com/dokku/dokku/master/bootstrap.sh | sudo bash

# Vérifier l'installation
dokku version
```

### Permissions

L'utilisateur exécutant les tests doit avoir les permissions pour :
- Exécuter les commandes Dokku
- Créer et supprimer des applications
- Accéder aux logs et événements Dokku

```bash
# Ajouter l'utilisateur au groupe dokku (si nécessaire)
sudo usermod -aG dokku $USER

# Vérifier les permissions
dokku apps:list
```

## Configuration

### Variables d'environnement

```bash
# Chemin vers l'exécutable Dokku (optionnel)
export DOKKU_PATH="/usr/bin/dokku"

# Timeout pour les tests (optionnel)
export TEST_TIMEOUT="10m"

# Mode verbeux (optionnel)
export VERBOSE="true"

# Activer les tests d'intégration
export DOKKU_MCP_INTEGRATION_TESTS=1
```

### Configuration Dokku

Pour tester l'historique des déploiements, il est recommandé d'activer les événements Dokku :

```bash
# Activer les événements
sudo dokku events:on

# Vérifier les événements
dokku events
```

## Exécution des tests

### Méthodes d'exécution

#### 1. Script dédié (recommandé)

```bash
# Tests basiques
./scripts/run-integration-tests.sh

# Mode verbeux
./scripts/run-integration-tests.sh --verbose

# Inclure les benchmarks
./scripts/run-integration-tests.sh --bench

# Nettoyage uniquement
./scripts/run-integration-tests.sh --cleanup-only

# Aide complète
./scripts/run-integration-tests.sh --help
```

#### 2. Via Makefile

```bash
# Tests d'intégration
make test-integration

# Tests verbeux
make test-integration-verbose

# Tests avec benchmarks
make test-integration-bench

# Nettoyage
make test-integration-clean

# Tous les tests (unité + intégration)
make test-all
```

#### 3. Commande Go directe

```bash
# Avec variables d'environnement
DOKKU_MCP_INTEGRATION_TESTS=1 go test -tags=integration -v ./internal/infrastructure/dokku/

# Tests courts exclus
go test -tags=integration -short ./internal/infrastructure/dokku/
```

### Options de contrôle

#### Skip des tests

Les tests d'intégration sont automatiquement ignorés si :
- `testing.Short()` est vrai (`-short` flag)
- Dokku n'est pas disponible à `/usr/bin/dokku`
- `DOKKU_MCP_INTEGRATION_TESTS` n'est pas défini

#### Personnalisation du chemin Dokku

```bash
# Chemin personnalisé
./scripts/run-integration-tests.sh --dokku-path /opt/dokku/bin/dokku

# Variable d'environnement
DOKKU_PATH=/opt/dokku/bin/dokku ./scripts/run-integration-tests.sh
```

## Tests implémentés

### 1. Tests de base du client (`TestDokkuClient_Integration_Basic`)

- **Objectif** : Vérifier les opérations de base du client Dokku
- **Scénarios** :
  - Listage des applications
  - Vérification d'applications inexistantes
- **Durée** : ~5 secondes

### 2. Tests GetHistory (`TestDeploymentService_Integration_GetHistory`)

- **Objectif** : Tester la récupération d'historique avec de vraies applications
- **Scénarios** :
  - Application inexistante → liste vide
  - Application existante → historique disponible
- **Applications créées** : 1 application de test
- **Durée** : ~15 secondes

### 3. Tests de parsing d'événements (`TestDeploymentService_Integration_EventsParsing`)

- **Objectif** : Valider le parsing des événements Dokku réels
- **Conditions** : Skip si les événements ne sont pas activés
- **Durée** : ~5 secondes

### 4. Tests de fallback git:report (`TestDeploymentService_Integration_GitReport`)

- **Objectif** : Tester le mécanisme de fallback
- **Applications créées** : 1 application de test
- **Durée** : ~10 secondes

### 5. Workflow complet (`TestDeploymentService_Integration_FullWorkflow`)

- **Objectif** : Tester un workflow complet de création et historique
- **Étapes** :
  1. Vérification app inexistante
  2. Création de l'application
  3. Récupération de l'historique
- **Applications créées** : 1 application de test
- **Durée** : ~20 secondes

### 6. Benchmarks (`BenchmarkDeploymentService_Integration_GetHistory`)

- **Objectif** : Mesurer les performances de GetHistory
- **Métriques** : Temps d'exécution, allocations mémoire
- **Applications créées** : 1 application de test

## Sécurité et nettoyage

### Noms d'applications de test

Toutes les applications de test utilisent le préfixe `dokku-mcp-test-` suivi d'un timestamp :

```
dokku-mcp-test-1735123456789
```

### Nettoyage automatique

Le système de nettoyage fonctionne à plusieurs niveaux :

1. **Par test** : Chaque test nettoie ses applications avec `t.Cleanup()`
2. **Global** : `TestMain` nettoie les applications orphelines
3. **Script** : Le script nettoie avant et après l'exécution
4. **Manuel** : `make test-integration-clean`

### Nettoyage manuel

Si des applications de test persistent :

```bash
# Via script
./scripts/run-integration-tests.sh --cleanup-only

# Via Makefile
make test-integration-clean

# Manuellement avec Dokku
dokku apps:list | grep dokku-mcp-test | xargs -I {} dokku apps:destroy {} --force
```

## Dépannage

### Dokku non trouvé

```bash
Error: Dokku non trouvé à /usr/bin/dokku
```

**Solutions** :
1. Installer Dokku
2. Spécifier le bon chemin : `--dokku-path /path/to/dokku`
3. Ignorer la vérification : `--skip-checks`

### Permissions insuffisantes

```bash
Warning: Impossible d'exécuter 'dokku apps:list'
```

**Solutions** :
1. Ajouter l'utilisateur au groupe dokku : `sudo usermod -aG dokku $USER`
2. Exécuter avec sudo (non recommandé pour les tests)

### Événements Dokku non activés

```bash
Events command not available or enabled
```

**Solutions** :
1. Activer les événements : `sudo dokku events:on`
2. Le test sera automatiquement ignoré

### Tests qui traînent

Si les tests semblent bloqués :

1. Vérifier les applications de test : `dokku apps:list | grep dokku-mcp-test`
2. Nettoyer manuellement : `make test-integration-clean`
3. Vérifier les processus Dokku : `ps aux | grep dokku`

### Timeouts

Pour des tests plus longs :

```bash
./scripts/run-integration-tests.sh --timeout 20m
```

## Intégration CI/CD

### GitHub Actions (exemple)

```yaml
name: Integration Tests
on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Install Dokku
        run: |
          wget -O- https://raw.githubusercontent.com/dokku/dokku/master/bootstrap.sh | sudo bash
          sudo dokku events:on
          
      - name: Run Integration Tests
        run: |
          export DOKKU_MCP_INTEGRATION_TESTS=1
          make test-integration-verbose
```

### Variables CI

Recommandations pour l'environnement CI :

```bash
# Timeout plus long pour CI
export TEST_TIMEOUT="15m"

# Toujours verbeux en CI
export VERBOSE="true"

# Nettoyer même en cas d'échec
trap "make test-integration-clean" EXIT
```

## Métriques et monitoring

### Métriques collectées

- **Temps d'exécution** : Par test et global
- **Applications créées** : Compteur par test
- **Taux de nettoyage** : Applications supprimées vs créées
- **Taux d'échec** : Tests échoués vs total

### Logs de debug

Mode verbeux activé avec :

```bash
export VERBOSE=true
# ou
./scripts/run-integration-tests.sh --verbose
```

Logs inclus :
- Création/suppression d'applications
- Commandes Dokku exécutées
- Résultats de parsing
- Métriques de performance

## Bonnes pratiques

### Développement

1. **Toujours nettoyer** : Utiliser `t.Cleanup()` systématiquement
2. **Tests isolés** : Chaque test doit être indépendant
3. **Noms uniques** : Utiliser des timestamps pour éviter les collisions
4. **Timeouts appropriés** : Prévoir suffisamment de temps pour Dokku
5. **Vérifications préalables** : Skip si l'environnement n'est pas prêt

### Débogage

1. **Mode verbeux** : Activer pour voir les détails
2. **Tests individuels** : Exécuter un test spécifique
3. **Nettoyage manuel** : Nettoyer entre les tentatives
4. **Logs Dokku** : Consulter `/var/log/dokku/events.log`

### Performance

1. **Parallélisation limitée** : Éviter trop de tests simultanés
2. **Réutilisation** : Partager les applications quand possible
3. **Timeout optimisés** : Ni trop courts, ni trop longs
4. **Benchmarks réguliers** : Monitorer les performances

## Exemple complet

```bash
#!/bin/bash

# Préparation de l'environnement
export DOKKU_MCP_INTEGRATION_TESTS=1
export VERBOSE=true

# Vérifications préalables
if ! command -v dokku &> /dev/null; then
    echo "Dokku non installé"
    exit 1
fi

# Activer les événements si possible
sudo dokku events:on 2>/dev/null || true

# Nettoyer avant de commencer
make test-integration-clean

# Exécuter les tests
make test-integration-verbose

# Benchmarks (optionnel)
make test-integration-bench

# Nettoyage final
make test-integration-clean

echo "Tests d'intégration terminés avec succès!"
```

## Support

Pour les problèmes liés aux tests d'intégration :

1. Vérifier la documentation Dokku officielle
2. Consulter les logs détaillés avec `--verbose`
3. Tester manuellement les commandes Dokku utilisées
4. Vérifier les permissions et l'installation Dokku 