package dependency

import (
	"fmt"
	"sync"

	consulapi "github.com/hashicorp/consul/api"
	vaultapi "github.com/hashicorp/vault/api"
)

// ClientSet is a collection of clients that dependencies use to communicate
// with remote services like Consul or Vault.
type ClientSet struct {
	sync.RWMutex

	vault  *vaultapi.Client
	consul *consulapi.Client
}

// NewClientSet creates a new client set that is ready to accept clients.
func NewClientSet() *ClientSet {
	return &ClientSet{}
}

// Add stores the given client.
func (c *ClientSet) Add(client interface{}) error {
	c.Lock()
	defer c.Unlock()

	switch typed := client.(type) {
	case *consulapi.Client:
		if c.consul != nil {
			return fmt.Errorf("a consul client already exists")
		}
		c.consul = typed
	case *vaultapi.Client:
		if c.vault != nil {
			return fmt.Errorf("a vault client already exists")
		}
		c.vault = typed
	default:
		return fmt.Errorf("unknown client type %T", client)
	}

	return nil
}

// Consul returns the Consul client for this clientset, or an error if no
// Consul client has been set.
func (c *ClientSet) Consul() (*consulapi.Client, error) {
	c.RLock()
	defer c.RUnlock()

	if c.consul == nil {
		return nil, fmt.Errorf("clientset: missing consul client")
	}
	cp := new(consulapi.Client)
	*cp = *c.consul
	return cp, nil
}

// Vault returns the Vault client for this clientset, or an error if no
// Vault client has been set.
func (c *ClientSet) Vault() (*vaultapi.Client, error) {
	c.RLock()
	defer c.RUnlock()

	if c.vault == nil {
		return nil, fmt.Errorf("clientset: missing vault client")
	}
	cp := new(vaultapi.Client)
	*cp = *c.vault
	return cp, nil
}
