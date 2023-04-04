package ecr

import (
	"github.com/rh-mobb/ecr-secret-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ArgoHelmSecretGenerator interface {
	GenerateSecret(input *v1alpha1.ArgoHelmRepoSecret) (*v1.Secret, error)
}

type DefaulArgoHelmSecretGenerator struct {
	r TokenRetriever
}

func NewArgoHelmSecretGenerator(r TokenRetriever) *DefaulArgoHelmSecretGenerator {
	return &DefaulArgoHelmSecretGenerator{
		r: r,
	}
}

func (sg *DefaulArgoHelmSecretGenerator) GenerateSecret(input *v1alpha1.ArgoHelmRepoSecret) (*v1.Secret, error) {
	var token string
	var err error
	if token, err = sg.r.GetToken(input.Spec.Region); err != nil {
		return nil, err
	}
	return &v1.Secret{
		TypeMeta: secretMeta,
		ObjectMeta: metav1.ObjectMeta{
			Namespace: input.ObjectMeta.Namespace,
			Name:      input.Spec.GenerateSecretName,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Data: map[string][]byte{
			"username": []byte("AWS"),
			"password": []byte(token),
			"url":      []byte(input.Spec.URL),
			"type":     []byte("helm"),
			"name":     []byte(input.Spec.GenerateSecretName),
		},
	}, nil
}
