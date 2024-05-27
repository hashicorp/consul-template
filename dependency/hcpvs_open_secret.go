package dependency

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vault-secrets/preview/2023-11-28/client/secret_service"
	"github.com/pkg/errors"
)

var _ Dependency = (*HCPVSOpenSecretQuery)(nil)

type HCPVSOpenSecretQuery struct {
	stopCh     chan struct{}
	appName    string
	secretName string
}

func NewHCPVSOpenSecretQuery(app, secret string) (*HCPVSOpenSecretQuery, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return nil, fmt.Errorf("hcpvs.opensecret: invalid format for app: %q", app)
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("hcpvs.opensecret: invalid format for secret: %q", secret)
	}

	return &HCPVSOpenSecretQuery{
		stopCh:     make(chan struct{}, 1),
		appName:    app,
		secretName: secret,
	}, nil
}

func (d *HCPVSOpenSecretQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	opts = opts.Merge(&QueryOptions{})

	// If we got this far, we either didn't have a secret to renew, the secret was
	// not renewable, or the renewal failed, so attempt a fresh list.
	log.Printf("[TRACE] %s", d)

	p := secret_service.NewOpenAppSecretParams()
	p.AppName, p.SecretName = d.appName, d.secretName
	res, err := clients.hcpvs.client.OpenAppSecret(p, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	result := res.GetPayload().Secret.StaticVersion.Value

	log.Printf("[TRACE] %s: returned 1 result", d)

	return respWithMetadata(result)
}

func (d *HCPVSOpenSecretQuery) CanShare() bool {
	return false
}

func (d *HCPVSOpenSecretQuery) String() string {
	return fmt.Sprintf("hcpvs.open(%s, %s)", d.appName, d.secretName)
}

func (d *HCPVSOpenSecretQuery) Stop() {
	close(d.stopCh)
}

func (d *HCPVSOpenSecretQuery) Type() Type {
	return TypeHCPVS
}
