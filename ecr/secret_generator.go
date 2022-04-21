package ecr

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pkg/errors"
	"github.com/rh-mobb/ecr-secret-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var secretMeta = metav1.TypeMeta{
	APIVersion: "v1",
	Kind:       "Secret",
}

type Input struct {
	S         *v1alpha1.Secret
	IamSecret *v1.Secret
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
	var cfg *aws.Config
	var dockerAuth []byte
	if cfg, err = getAWSConfig(input.IamSecret); err != nil {
		return nil, err
	}
	if token, err = sg.r.GetToken(cfg); err != nil {
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

func getAWSConfig(iam *v1.Secret) (*aws.Config, error) {
	namespace := iam.ObjectMeta.Namespace
	name := iam.ObjectMeta.Name
	aws_access_key_id, ok := iam.Data["aws_secret_access_id"]
	if !ok {
		return nil, fmt.Errorf("secret invalid, no %#v key in %s/%s", "aws_secret_access_id", namespace, name)
	}

	aws_secret_key, ok := iam.Data["aws_secret_access_key"]
	if !ok {
		return nil, fmt.Errorf("secret invalid, no %#v key in %s/%s", "aws_secret_access_key", namespace, name)
	}

	region, ok := iam.Data["region"]
	if !ok {
		return nil, errors.Errorf("secret invalid, no %#v key in %s/%s", "region", namespace, name)
	}

	return &aws.Config{
		Region:      aws.String(string(region)),
		Credentials: credentials.NewStaticCredentials(string(aws_access_key_id), string(aws_secret_key), ""),
	}, nil
}
