package sbproxy

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSbproxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sbproxy Suite")
}
