package sm

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sm Suite")
}
