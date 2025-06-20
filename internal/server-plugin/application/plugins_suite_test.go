//go:build !integration

package plugins_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServerPlugins(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "[Server Plugins] - Application Layer")
}
