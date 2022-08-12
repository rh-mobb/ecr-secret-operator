package ecr

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/rh-mobb/ecr-secret-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var secretMeta = metav1.TypeMeta{
	APIVersion: "v1",
	Kind:       "Secret",
}

type Input struct {
	S *v1alpha1.Secret
}

type SecretGenerator interface {
	GenerateSecret(input *Input) (*v1.Secret, error)
}

type DefaultSecretGenerator struct {
	r TokenRetriever
}

func NewDefaultSecretGenerator(r TokenRetriever) *DefaultSecretGenerator {
	return &DefaultSecretGenerator{
		r: r,
	}
}

func (sg *DefaultSecretGenerator) GenerateSecret(input *Input) (*v1.Secret, error) {
	var token string
	var err error
	var dockerAuth []byte
	if token, err = sg.r.GetToken("eu-central-1"); err != nil {
		return nil, err
	}
	if dockerAuth, err = getDockerAuth(input.S.Spec.ECRRegistry, "AWS", token); err != nil {
		return nil, err
	}
	return &v1.Secret{
		TypeMeta: secretMeta,
		ObjectMeta: metav1.ObjectMeta{
			Namespace: input.S.ObjectMeta.Namespace,
			Name:      input.S.Spec.GenerateSecretName,
		},
		Data: map[string][]byte{
			".dockerconfigjson": dockerAuth,
		},
		Type: "kubernetes.io/dockerconfigjson",
	}, nil
}

type DockerAuths struct {
	Auths map[string]Auth `json:"auths"`
}

type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

func getDockerAuth(registry, username, password string) ([]byte, error) {
	auth := DockerAuths{
		Auths: map[string]Auth{
			registry: {
				Username: username,
				Password: password,
				Auth:     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
			},
		},
	}

	return json.Marshal(&auth)
}
