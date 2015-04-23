package dependency

import (
	"fmt"
	"log"
	"sync"
	"time"
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

	secretPath    string
	leaseDuration int
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

	// Attempt to read the secret
	vaultSecret, err := vault.Logical().Read(d.secretPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading from vault: %s", err)
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
	d.leaseDuration = leaseDuration
	d.Unlock()

	log.Printf("[DEBUG] (%s) Consul returned the secret", d.Display())

	ts := time.Now().Unix()
	rm := &ResponseMetadata{
		LastContact: time.Duration(ts),
		LastIndex:   uint64(ts),
	}

	return secret, rm, nil
}

// HashCode returns the hash code for this dependency.
func (d *VaultSecret) HashCode() string {
	return fmt.Sprintf("VaultSecret|%s", d.secretPath)
}

// Display returns a string that should be displayed to the user in output (for
// example).
func (d *VaultSecret) Display() string {
	return fmt.Sprintf(`"vault(%s)"`, d.secretPath)
}

// ParseVaultSecret creates a new datacenter dependency.
func ParseVaultSecret(s string) (*VaultSecret, error) {
	return &VaultSecret{secretPath: s}, nil
}
