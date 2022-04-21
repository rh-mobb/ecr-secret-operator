package ecr_test

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
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
			cfg := &aws.Config{
				Region:      aws.String("us-west-2"),
				Credentials: credentials.NewStaticCredentials("A", "H", ""),
			}
			_, err := r.GetToken(cfg)
			Expect(err).Should(BeNil())
		})
	})

})
