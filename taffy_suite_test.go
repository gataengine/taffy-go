package taffy

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTaffy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Taffy Suite")
}
