package dependency

import (
	"fmt"
	"log"
	"sync"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

// Secret is a vault secret.
type Secret struct {
	LeaseID       string
	LeaseDuration int
	Renewable     bool

	// Data is the actual contents of the secret. The format of the data
	// is arbitrary and up to the secret backend.
	Data map[string]interface{}
}

// VaultSecret is the dependency to Vault for a secret
type VaultSecret struct {
	sync.Mutex

	Path          string
	leaseID       string
	leaseDuration int
	renewable     bool
}

// Fetch queries the Vault API
func (d *VaultSecret) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}

	log.Printf("[DEBUG] (%s) querying vault with %+v", d.Display(), opts)

	// If this is not the first query and we have a lease duration, sleep until we
	// try to renew.
	if opts.WaitIndex != 0 && d.leaseDuration != 0 {
		duration := time.Duration(d.leaseDuration/2) * time.Second
		log.Printf("[DEBUG] (%s) pretending to long-poll for %q",
			d.Display(), duration)
		time.Sleep(duration)
	}

	// Grab the vault client
	vault, err := clients.Vault()
	if err != nil {
		return nil, nil, fmt.Errorf("vault secret: %s", err)
	}

	// Attempt to renew the secret
	var vaultSecret *vaultapi.Secret
	if d.renewable && d.leaseID != "" {
		vaultSecret, err = vault.Sys().Renew(d.leaseID, 1)
		if err != nil {
			log.Printf("[WARN] (%s) failed to renew, re-reading", d.Display())
		}
	}

	// If we did not renew, attempt a fresh read
	if vaultSecret == nil {
		vaultSecret, err = vault.Logical().Read(d.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading from vault: %s", err)
		}
	}

	// The secret could be nil (maybe it does not exist yet)
	if vaultSecret == nil {
		return nil, nil, fmt.Errorf("no secret exists at path %q", d.Display())
	}

	// Create our cloned secret
	secret := &Secret{
		LeaseID:       vaultSecret.LeaseID,
		LeaseDuration: vaultSecret.LeaseDuration,
		Renewable:     vaultSecret.Renewable,
		Data:          vaultSecret.Data,
	}

	leaseDuration := secret.LeaseDuration
	if leaseDuration == 0 {
		log.Printf("[WARN] (%s) lease duration is 0, setting to 5m", d.Display())
		leaseDuration = 5 * 60
	}

	d.Lock()
	d.leaseID = secret.LeaseID
	d.leaseDuration = secret.LeaseDuration
	d.renewable = secret.Renewable
	d.Unlock()

	log.Printf("[DEBUG] (%s) vault returned the secret", d.Display())

	ts := time.Now().Unix()
	rm := &ResponseMetadata{
		LastContact: 0,
		LastIndex:   uint64(ts),
	}

	return secret, rm, nil
}

// HashCode returns the hash code for this dependency.
func (d *VaultSecret) HashCode() string {
	return fmt.Sprintf("VaultSecret|%s", d.Path)
}

// Display returns a string that should be displayed to the user in output (for
// example).
func (d *VaultSecret) Display() string {
	return fmt.Sprintf(`"vault(%s)"`, d.Path)
}

// ParseVaultSecret creates a new datacenter dependency.
func ParseVaultSecret(s string) (*VaultSecret, error) {
	return &VaultSecret{Path: s}, nil
}
