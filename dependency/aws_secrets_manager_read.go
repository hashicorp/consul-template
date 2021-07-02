package dependency

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"net/url"
	"os"
	"strings"
)

var (
	// Ensure implements
	_ Dependency = (*AWSsecretsManagerQuery)(nil)
)

type AWSsecretsManagerQuery struct {
	stopCh chan struct{}

	path string
}

func NewAWSsecretsManagerQuery(s string) (*AWSsecretsManagerQuery, error) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	if s == "" {
		return nil, fmt.Errorf("vault.read: invalid format: %q", s)
	}

	secretURL, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	secretID := os.Getenv("CONSUL_TEMPLATE_SECRET_PREFIX") + secretURL.Path

	return &AWSsecretsManagerQuery{
		stopCh: make(chan struct{}, 1),
		path:   secretID,
	}, nil
}

func (s *AWSsecretsManagerQuery) Fetch(clients *ClientSet, _options *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-s.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	client := clients.SecretsManager()

	output, err := client.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId:     aws.String(s.path),
			VersionStage: aws.String("AWSCURRENT"),
		},
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, s.String())
	}

	return respWithMetadata(
		&SecretsContainer{
			Raw: nil,
			Data: map[string]interface{}{
				"value": aws.StringValue(output.SecretString),
			},
		},
	)
}

func (s *AWSsecretsManagerQuery) CanShare() bool {
	return false
}

func (s *AWSsecretsManagerQuery) String() string {
	return fmt.Sprintf("secrets_manager.read(%s)", s.path)
}

func (s *AWSsecretsManagerQuery) Stop() {
	close(s.stopCh)
}

func (s *AWSsecretsManagerQuery) Type() Type {
	return TypeAWSsecretsManager
}
