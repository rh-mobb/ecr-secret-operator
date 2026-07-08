package ecr

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

type TokenRetriever interface {
	GetToken(region string) (string, error)
}

type DefaultTokenRetriever struct {
}

func NewDefaultTokenRetriever() *DefaultTokenRetriever {
	return &DefaultTokenRetriever{}
}

func (r *DefaultTokenRetriever) GetToken(region string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return "", err
	}
	svc := ecr.NewFromConfig(cfg)
	result, err := svc.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	}
	var tokenBytes []byte
	if tokenBytes, err = base64.StdEncoding.DecodeString(*result.AuthorizationData[0].AuthorizationToken); err != nil {
		return "", err
	}
	token := strings.Split(string(tokenBytes), ":")
	if len(token) != 2 {
		return "", errors.New("token returned from AWS is not valid")
	}
	return token[1], nil
}
