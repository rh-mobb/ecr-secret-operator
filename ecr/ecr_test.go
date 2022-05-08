package ecr_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rh-mobb/ecr-secret-operator/ecr"
)

var _ = Describe("Ecr", func() {

	var (
		r ecr.TokenRetriever = ecr.NewDefaultTokenRetriever()
	)

	Context("", func() {
		It("Should get a token", func() {
			Skip("")
			_, err := r.GetToken("us-east-2")
			Expect(err).Should(BeNil())
		})
	})

})
