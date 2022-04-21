package ecr_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEcr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ecr Suite")
}
