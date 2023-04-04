package ecr_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-mobb/ecr-secret-operator/api/v1alpha1"
	"github.com/rh-mobb/ecr-secret-operator/ecr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SecretGenerator", func() {
	var (
		tg *FakeTokenGenerator          = &FakeTokenGenerator{}
		sg ecr.ArgoHelmSecretGenerator  = ecr.NewArgoHelmSecretGenerator(tg)
		s  *v1alpha1.ArgoHelmRepoSecret = &v1alpha1.ArgoHelmRepoSecret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "ecr.mobb.redhat.com/v1alpha1",
				Kind:       "ArgoHelmRepoSecret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mysecret",
				Namespace: "test-namespace",
			},
			Spec: v1alpha1.ArgoHelmRepoSecretSpec{
				GenerateSecretName: "test-secret",
				URL:                "https://test.dkr.ecr.abc.amazonaws.com/abc",
				Region:             "region_name",
			},
		}
	)

	Context("", func() {
		It("Should have correct secret configuration", func() {
			secret, err := sg.GenerateSecret(s)
			Expect(err).Should(BeNil())
			Expect(secret.ObjectMeta.Name).Should(Equal("test-secret"))
			Expect(secret.ObjectMeta.Namespace).Should(Equal("test-namespace"))
			Expect(secret.ObjectMeta.Labels).Should(Equal(map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			}))
			Expect(secret.TypeMeta.APIVersion).Should(Equal("v1"))
			Expect(secret.TypeMeta.Kind).Should(Equal("Secret"))
			Expect(region).Should(Equal("region_name"))
			Expect(string(secret.Data["url"])).Should(Equal("https://test.dkr.ecr.abc.amazonaws.com/abc"))
			Expect(string(secret.Data["username"])).Should(Equal("AWS"))
			Expect(string(secret.Data["password"])).Should(Equal("test"))
			Expect(string(secret.Data["name"])).Should(Equal("test-secret"))
		})
	})
})
