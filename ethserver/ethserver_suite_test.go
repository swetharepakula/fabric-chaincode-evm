package ethserver_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestEthserver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ethserver Suite")
}
