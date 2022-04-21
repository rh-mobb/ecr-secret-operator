package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rh-mobb/ecr-secret-operator/api/v1alpha1"
	"github.com/rh-mobb/ecr-secret-operator/ecr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	secretName          = "test-secret-name"
	secretNameSpace     = "default"
	generatedSecretName = "test-secret"
	ecrRegistryDomain   = "test.dkr.ecr.abc.amazonaws.com"
	awsIamSecretName    = "aws-iam-secret"

	timeout    = time.Second * 10
	duration   = time.Second * 10
	interval   = time.Millisecond * 250
	dockerJson = `{"auths":{"test.dkr.ecr.abc.amazonaws.com":{"username":"AWS","password":"test","auth":"QVdTOnRlc3Q="}}}`
)

type FakeSecretGenerator struct {
}

func (f *FakeSecretGenerator) GenerateSecret(s *ecr.Input) (*v1.Secret, error) {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.S.ObjectMeta.Namespace,
			Name:      s.S.Spec.GenerateSecretName,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(dockerJson),
		},
		Type: "kubernetes.io/dockerconfigjson",
	}, nil
}

var _ = Describe("SecretController", func() {
	Context("", func() {
		It("Should create a secret", func() {
			ctx := context.Background()
			secret := &v1alpha1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "ecr.mobb.redhat.com/v1alpha1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNameSpace,
				},
				Spec: v1alpha1.SecretSpec{
					GenerateSecretName: generatedSecretName,
					ECRRegistry:        ecrRegistryDomain,
					Frequency:          &metav1.Duration{Duration: duration},
					AwsIamSecret: &v1.SecretReference{
						Name: awsIamSecretName,
					},
				},
			}

			iamSecret := &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      awsIamSecretName,
					Namespace: secretNameSpace,
				},
				Data: map[string][]byte{
					"aws_secret_access_id":  []byte("aws_access_key"),
					"aws_secret_access_key": []byte("aws_secret_access_key"),
					"region":                []byte("region"),
				},
			}

			//Create CRD object and IAM secret
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			Expect(k8sClient.Create(ctx, iamSecret)).Should(Succeed())

			secretLookupKey := types.NamespacedName{Name: secretName, Namespace: secretNameSpace}
			createdSecret := &v1alpha1.Secret{}

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
			dockerSecretLookupKey := types.NamespacedName{Name: generatedSecretName, Namespace: secretNameSpace}
			dockerSecret := &v1.Secret{}
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, dockerSecretLookupKey, dockerSecret)
				if err != nil {
					return "", err
				}
				return string(dockerSecret.Data[".dockerconfigjson"]), nil
			}, timeout, interval).Should(Equal(dockerJson))
		})
	})
})
