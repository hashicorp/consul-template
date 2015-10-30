package dependency

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// VaultToken is the dependency to Vault for a secret
type VaultToken struct {
	sync.Mutex

	leaseID       string
	leaseDuration int
}

// Fetch queries the Vault API
func (d *VaultToken) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}

	log.Printf("[DEBUG] (%s) renewing vault token", d.Display())

	// If this is not the first query and we have a lease duration, sleep until we
	// try to renew.
	if opts.WaitIndex != 0 && d.leaseDuration != 0 {
		duration := time.Duration(d.leaseDuration/2) * time.Second
		log.Printf("[DEBUG] (%s) sleeping for %q", d.Display(), duration)
		time.Sleep(duration)
	}

	// Grab the vault client
	vault, err := clients.Vault()
	if err != nil {
		return nil, nil, fmt.Errorf("vault_token: %s", err)
	}

	token, err := vault.Auth().Token().RenewSelf(0)
	if err != nil {
		return nil, nil, fmt.Errorf("error renewing vault token: %s", err)
	}

	// Create our cloned secret
	secret := &Secret{
		LeaseID:       token.LeaseID,
		LeaseDuration: token.Auth.LeaseDuration,
		Renewable:     token.Auth.Renewable,
		Data:          token.Data,
	}

	leaseDuration := secret.LeaseDuration
	if leaseDuration == 0 {
		log.Printf("[WARN] (%s) lease duration is 0, setting to 5m", d.Display())
		leaseDuration = 5 * 60
	}

	d.Lock()
	d.leaseID = secret.LeaseID
	d.leaseDuration = secret.LeaseDuration
	d.Unlock()

	log.Printf("[DEBUG] (%s) successfully renewed token", d.Display())

	ts := time.Now().Unix()
	rm := &ResponseMetadata{
		LastContact: 0,
		LastIndex:   uint64(ts),
	}

	return secret, rm, nil
}

// HashCode returns the hash code for this dependency.
func (d *VaultToken) HashCode() string {
	return "VaultToken"
}

// Display returns a string that should be displayed to the user in output (for
// example).
func (d *VaultToken) Display() string {
	return "vault_token"
}
