package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rh-mobb/ecr-secret-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	helmRepoURL = "https://test.dkr.ecr.abc.amazonaws.com/repo"
)

type FakeArgoHelmSecretGenerator struct {
}

func (f *FakeArgoHelmSecretGenerator) GenerateSecret(s *v1alpha1.ArgoHelmRepoSecret) (*v1.Secret, error) {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.ObjectMeta.Namespace,
			Name:      s.Spec.GenerateSecretName,
		},
		Data: map[string][]byte{
			"url":      []byte(helmRepoURL),
			"username": []byte("AWS"),
			"password": []byte("test"),
		},
	}, nil
}

var _ = Describe("ArgoHelmRepoSecretController", func() {
	Context("", func() {
		It("Should create a secret", func() {
			ctx := context.Background()
			secret := &v1alpha1.ArgoHelmRepoSecret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "ecr.mobb.redhat.com/v1alpha1",
					Kind:       "ArgoHelmRepoSecret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNameSpace,
				},
				Spec: v1alpha1.ArgoHelmRepoSecretSpec{
					GenerateSecretName: "test-secret",
					URL:                helmRepoURL,
					Frequency:          &metav1.Duration{Duration: duration},
					Region:             "us-east-2",
				},
			}

			//Create CRD object
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			secretLookupKey := types.NamespacedName{Name: secretName, Namespace: secretNameSpace}
			createdSecret := &v1alpha1.ArgoHelmRepoSecret{}

			// Make sure the CRD object is created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Make sure the CRD object is updated
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, secretLookupKey, createdSecret)
				if err != nil {
					return "", err
				}
				return createdSecret.Status.Phase, nil
			}, timeout, interval).Should(Equal("Updated"))

			// Make sure the docker secret object is created
			secretLookupKey = types.NamespacedName{Name: "test-secret", Namespace: secretNameSpace}
			createdArgoHelmRepoSecret := &v1.Secret{}
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, secretLookupKey, createdArgoHelmRepoSecret)
				if err != nil {
					return "", err
				}
				return string(createdArgoHelmRepoSecret.Data["password"]), nil
			}, timeout, interval).Should(Equal("test"))
		})
	})
})
