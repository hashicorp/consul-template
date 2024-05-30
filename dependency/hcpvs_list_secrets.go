package dependency

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vault-secrets/preview/2023-11-28/client/secret_service"
	"github.com/pkg/errors"
)

var _ Dependency = (*HCPVSListQuery)(nil)

type HCPVSListQuery struct {
	stopCh  chan struct{}
	appName string
}

func NewHCPVSListQuery(s string) (*HCPVSListQuery, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("hcpvs.list: invalid format: %q", s)
	}

	return &HCPVSListQuery{
		stopCh:  make(chan struct{}, 1),
		appName: s,
	}, nil
}

func (d *HCPVSListQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	opts = opts.Merge(&QueryOptions{})

	// If we got this far, we either didn't have a secret to renew, the secret was
	// not renewable, or the renewal failed, so attempt a fresh list.
	log.Printf("[TRACE] HCPVS: LIST %q", d.appName)

	p := secret_service.NewListAppSecretsParams()
	p.AppName = d.appName
	p.OrganizationID = clients.hcpvs.orgID
	p.ProjectID = clients.hcpvs.projID
	res, err := clients.hcpvs.client.ListAppSecrets(p, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	var result []string
	for _, v := range res.GetPayload().Secrets {
		result = append(result, v.Name)
	}

	log.Printf("[TRACE] %s: returned %d results", d, len(result))

	return respWithMetadata(result)
}

func (d *HCPVSListQuery) CanShare() bool {
	return false
}

func (d *HCPVSListQuery) String() string {
	return fmt.Sprintf("hcpvs.list(%s)", d.appName)
}

func (d *HCPVSListQuery) Stop() {
	close(d.stopCh)
}

func (d *HCPVSListQuery) Type() Type {
	return TypeHCPVS
}
