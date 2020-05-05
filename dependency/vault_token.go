package dependency

import (
	"context"

	"github.com/hashicorp/consul-template/telemetry"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/api/metric"
)

var (
	// Ensure implements
	_ Dependency = (*VaultTokenQuery)(nil)
)

// VaultTokenQuery is the dependency to Vault for a secret
type VaultTokenQuery struct {
	stopCh      chan struct{}
	secret      *Secret
	vaultSecret *api.Secret

	// counterRenew is a counter to monitor the renewal status of the vault token.
	counterRenew metric.Int64Counter
}

// NewVaultTokenQuery creates a new dependency.
func NewVaultTokenQuery(token string) (*VaultTokenQuery, error) {
	vaultSecret := &api.Secret{
		Auth: &api.SecretAuth{
			ClientToken:   token,
			Renewable:     true,
			LeaseDuration: 1,
		},
	}

	meter := telemetry.GlobalMeter()
	counter, err := meter.NewInt64Counter("consul-template.vault.token",
		metric.WithDescription("A counter of vault token renewal statuses"+
			"with label status=(configured|renewed|expired|stopped)"))
	if err != nil {
		return nil, err
	}

	counter.Add(context.Background(), 1, telemetry.NewLabel("status", "configured"))

	return &VaultTokenQuery{
		stopCh:       make(chan struct{}, 1),
		vaultSecret:  vaultSecret,
		secret:       transformSecret(vaultSecret),
		counterRenew: counter,
	}, nil
}

// Fetch queries the Vault API
func (d *VaultTokenQuery) Fetch(clients *ClientSet, opts *QueryOptions,
) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	if vaultSecretRenewable(d.secret) {
		err := renewSecret(clients, d)
		if err != nil {
			return nil, nil, errors.Wrap(err, d.String())
		}
	}

	return nil, nil, ErrLeaseExpired
}

func (d *VaultTokenQuery) stopChan() chan struct{} {
	return d.stopCh
}

func (d *VaultTokenQuery) secrets() (*Secret, *api.Secret) {
	return d.secret, d.vaultSecret
}

// CanShare returns if this dependency is shareable.
func (d *VaultTokenQuery) CanShare() bool {
	return false
}

// Stop halts the dependency's fetch function.
func (d *VaultTokenQuery) Stop() {
	close(d.stopCh)
}

// String returns the human-friendly version of this dependency.
func (d *VaultTokenQuery) String() string {
	return "vault.token"
}

// Type returns the type of this dependency.
func (d *VaultTokenQuery) Type() Type {
	return TypeVault
}

// recordCounter increments a counter for the vault dependency with a
// set of key value label
func (d *VaultTokenQuery) recordCounter(key, value string) {
	ctx := context.Background()
	d.counterRenew.Add(ctx, 1, telemetry.NewLabel(key, value))
}
