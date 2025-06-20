//go:build integration

package dokkuApi_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDokku(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dokku Suite")
}
