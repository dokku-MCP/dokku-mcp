package dokkuApi_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDokkuUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dokku API Unit Suite")
}
