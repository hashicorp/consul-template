package dependency

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

// VaultSecret is the dependency to Vault for a secret
type VaultSecret struct {
	sync.Mutex
	rawKey        string
	leaseDuration int
}

// Fetch queries the Vault API
func (d *VaultSecret) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	log.Printf("[DEBUG] (%s) querying vault with %+v", d.Display(), options)

	// If this is not the first query and we have a lease duration, sleep until we
	// try to renew.
	if options.WaitIndex != 0 && d.leaseDuration != 0 {
		log.Printf("[DEBUG] (%s) pretending to long-poll", d.Display())
		time.Sleep(d.leaseDuration / 2)
	}

	// TODO - this is not the correct "client" - we need a vault client here, not
	// a consul client - we are going to need to update the dependency format a
	// bit to make this work...
	secret, err := client.Logical().Read(path)
	if err != nil {
		return nil, nil, err
	}

	if !secret.Renewable {
		// TODO: Return an error if the secret is not renewable?
	}

	if secret.Auth == nil {
		// TODO: Return an error if there is no auth (we can't check the lease duration)
	}

	d.Lock()
	d.leaseDuration = secret.Auth.LeaseDuration
	d.Unlock()

	log.Printf("[DEBUG] (%s) Consul returned the secret", d.Display())

	qm := &api.QueryMeta{LastIndex: uint64(time.Now().Unix())}

	return secret, qm, nil
}

// HashCode returns the hash code for this dependency.
func (d *VaultSecret) HashCode() string {
	return fmt.Sprintf("VaultSecret|%s", d.rawKey)
}

// Display returns a string that should be displayed to the user in output (for
// example).
func (d *VaultSecret) Display() string {
	if d.rawKey == "" {
		return fmt.Sprintf(`"vault"`)
	}

	return fmt.Sprintf(`"vault(%s)"`, d.rawKey)
}

// ParseVaultSecret creates a new datacenter dependency.
func ParseVaultSecret(s string) (*VaultSecret, error) {
	return &VaultSecret{rawKey: ""}, nil
}
