//go:build integration

package dokku

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestDokku est le point d'entrée pour la suite de tests d'intégration Dokku
// Suite optimisée avec les meilleures pratiques Ginkgo/Gomega :
// - Tests unitaires avec mocks simples sans dépendances externes
// - Tests d'intégration avec fixtures réutilisables et cleanup automatique
// - Utilisation de DescribeTable pour les tests data-driven
// - Pattern Builder pour créer des applications de test
// - Gestion d'erreurs avec Gomega matchers
// - Support des fonctionnalités Dokku conditionnelles (events, git:report)
func TestDokku(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dokku Suite - Tests Unitaires et d'Intégration")
}
