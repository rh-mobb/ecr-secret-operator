package ecr_test

import (
	"github.com/aws/aws-sdk-go/aws"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-mobb/ecr-secret-operator/api/v1alpha1"
	"github.com/rh-mobb/ecr-secret-operator/ecr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakeTokenGenerator struct {
}

func (f *FakeTokenGenerator) GetToken(cfg *aws.Config) (string, error) {
	return "test", nil
}

var _ = Describe("SecretGenerator", func() {
	var (
		tg         *FakeTokenGenerator        = &FakeTokenGenerator{}
		sg         ecr.DefaultSecretGenerator = *ecr.NewDefaultSecretGenerator(tg)
		dockerJson string                     = `{"auths":{"test.dkr.ecr.abc.amazonaws.com":{"username":"AWS","password":"test","auth":"QVdTOnRlc3Q="}}}`
		s          *v1alpha1.Secret           = &v1alpha1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "ecr.mobb.redhat.com/v1alpha1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mysecret",
				Namespace: "test-namespace",
			},
			Spec: v1alpha1.SecretSpec{
				GenerateSecretName: "test-secret",
				ECRRegistry:        "test.dkr.ecr.abc.amazonaws.com",
			},
		}
		iam *v1.Secret = &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mysecret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"aws_secret_access_id":  []byte("aws_access_key"),
				"aws_secret_access_key": []byte("aws_secret_access_key"),
				"region":                []byte("region"),
			},
		}
	)

	Context("", func() {
		It("Should have correct secret configuration", func() {
			secret, err := sg.GenerateSecret(&ecr.Input{
				S:         s,
				IamSecret: iam,
			})
			Expect(err).Should(BeNil())
			Expect(secret.ObjectMeta.Name).Should(Equal("test-secret"))
			Expect(secret.ObjectMeta.Namespace).Should(Equal("test-namespace"))
			Expect(secret.TypeMeta.APIVersion).Should(Equal("v1"))
			Expect(secret.TypeMeta.Kind).Should(Equal("Secret"))
			var tp v1.SecretType = "kubernetes.io/dockerconfigjson"
			Expect(secret.Type).Should(Equal(tp))
			// {
			// 	"auths": {
			// 	  "test.dkr.ecr.abc.amazonaws.com": {
			// 		"username": "AWS",
			// 		"password": "test",
			// 		"auth": "QVdTOnRlc3Q="
			// 	  }
			// 	}
			// }
			Expect(string(secret.Data[".dockerconfigjson"])).Should(Equal(dockerJson))
		})
	})
})
