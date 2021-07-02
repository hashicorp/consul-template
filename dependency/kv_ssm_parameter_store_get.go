package dependency

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"os"
	"strings"
)

var (
	// Ensure implements
	_ Dependency = (*SSMParameterStoreQuery)(nil)
)

type SSMParameterStoreQuery struct {
	stopCh chan struct{}

	key   string
	block bool
}

func NewSSMParameterStoreQuery(s string) (*SSMParameterStoreQuery, error) {
	if s != "" && !KVGetQueryRe.MatchString(s) {
		return nil, fmt.Errorf("kv.get: invalid format: %q", s)
	}

	pathPrefix := os.Getenv("CONSUL_TEMPLATE_KEY_PREFIX")
	m := regexpMatch(KVGetQueryRe, s)
	key := pathPrefix + m["key"]

	if len(key) > 0 && key[0] != '/' && strings.Contains(key, "/") {
		key = fmt.Sprintf("/%s", key)
	}

	return &SSMParameterStoreQuery{
		stopCh: make(chan struct{}, 1),
		key:    key,
	}, nil
}

func (d *SSMParameterStoreQuery) Fetch(clients *ClientSet, _options *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	client := clients.SSM()

	output, err := client.GetParameter(
		context.TODO(),
		&ssm.GetParameterInput{
			// HACK: Make it check if there's a slash and add leading one due to SSM parameter store quirk
			Name:           aws.String(d.key),
			WithDecryption: false,
		},
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	value := aws.StringValue(output.Parameter.Value)
	return respWithMetadata(value)
}

func (d *SSMParameterStoreQuery) CanShare() bool {
	return true
}

func (d *SSMParameterStoreQuery) String() string {
	if d.block {
		return fmt.Sprintf("kv_ssm_parameter_store.block(%s)", d.key)
	}
	return fmt.Sprintf("kv_ssm_parameter_store.get(%s)", d.key)
}

func (d *SSMParameterStoreQuery) Stop() {
	close(d.stopCh)
}

func (d *SSMParameterStoreQuery) Type() Type {
	return TypeAWSssmParameterStore
}

func (d *SSMParameterStoreQuery) EnableBlocking() {
	d.block = true
}
