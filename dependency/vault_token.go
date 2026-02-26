// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

// VaultTokenRefreshCurrent tells to refresh the current client token.
const VaultTokenRefreshCurrent = ""

// Ensure implements
var _ Dependency = (*VaultTokenQuery)(nil)

// VaultTokenQuery is the dependency to Vault for a secret
type VaultTokenQuery struct {
	stopCh      chan struct{}
	secret      *Secret
	vaultSecret *api.Secret

	initialToken string
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
	return &VaultTokenQuery{
		stopCh:       make(chan struct{}, 1),
		vaultSecret:  vaultSecret,
		secret:       transformSecret(vaultSecret),
		initialToken: token,
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

	var currentRenewer renewer = d

	if d.initialToken == VaultTokenRefreshCurrent {
		currentRenewer = newVaultSecretsOverrideRenewer(d, clients.Vault().Token())
	}

	secret, _ := currentRenewer.secrets()

	if vaultSecretRenewable(secret) {
		err := renewSecret(clients, currentRenewer)
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

func newVaultSecretsOverrideRenewer(parent renewer, token string) *vaultSecretsOverrideRenewer {
	vaultSecret := &api.Secret{
		Auth: &api.SecretAuth{
			ClientToken:   token,
			Renewable:     true,
			LeaseDuration: 1,
		},
	}

	return &vaultSecretsOverrideRenewer{
		renewer:     parent,
		vaultSecret: vaultSecret,
		secret:      transformSecret(vaultSecret),
	}
}

type vaultSecretsOverrideRenewer struct {
	renewer

	secret      *Secret
	vaultSecret *api.Secret
}

func (d *vaultSecretsOverrideRenewer) secrets() (*Secret, *api.Secret) {
	return d.secret, d.vaultSecret
}
