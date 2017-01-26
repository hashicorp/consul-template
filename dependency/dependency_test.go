package dependency

import (
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/vault/builtin/logical/pki"
	"github.com/hashicorp/vault/builtin/logical/transit"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/physical"
	"github.com/hashicorp/vault/vault"

	logxi "github.com/mgutz/logxi/v1"
)

func TestCanShare(t *testing.T) {
	t.Parallel()

	deps := []Dependency{
		&CatalogNodeQuery{},
		&FileQuery{},
		&VaultListQuery{},
		&VaultReadQuery{},
		&VaultTokenQuery{},
		&VaultWriteQuery{},
	}

	for _, d := range deps {
		if d.CanShare() {
			t.Errorf("should not share %s", d)
		}
	}
}

func TestDeepCopyAndSortTags(t *testing.T) {
	t.Parallel()

	tags := []string{"hello", "world", "these", "are", "tags"}
	expected := []string{"are", "hello", "tags", "these", "world"}

	result := deepCopyAndSortTags(tags)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v to be %#v", result, expected)
	}
}

// testConsulServer is a helper for creating a Consul server and returning the
// appropriate configuration to connect to it.
func testConsulServer(t *testing.T) (*ClientSet, *testutil.TestServer) {
	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})

	clients := NewClientSet()
	if err := clients.CreateConsulClient(&CreateConsulClientInput{
		Address: consul.HTTPAddr,
	}); err != nil {
		consul.Stop()
		t.Fatalf("clientset: %s", err)
	}

	return clients, consul
}

type vaultServer struct {
	Address string
	Token   string

	core *vault.Core
	ln   net.Listener
}

func (s *vaultServer) Stop() {
	s.ln.Close()
}

func (s *vaultServer) CreateSecret(path string, data map[string]interface{}) error {
	req := &logical.Request{
		Operation:   logical.UpdateOperation,
		Path:        fmt.Sprintf("secret/%s", path),
		Data:        data,
		ClientToken: s.Token,
	}
	_, err := s.core.HandleRequest(req)
	return err
}

// testVaultServer is a helper for creating a Vault server and returning the
// appropriate client to connect to it.
func testVaultServer(t *testing.T) (*ClientSet, *vaultServer) {
	core, err := vault.NewCore(&vault.CoreConfig{
		DisableMlock:    true,
		DisableCache:    true,
		DefaultLeaseTTL: 2 * time.Second,
		MaxLeaseTTL:     3 * time.Second,
		Logger:          logxi.NullLog,
		Physical:        physical.NewInmem(logxi.NullLog),
		LogicalBackends: map[string]logical.Factory{
			"pki":     pki.Factory,
			"transit": transit.Factory,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	keys, token := vault.TestCoreInit(t, core)

	for _, key := range keys {
		if _, err := vault.TestCoreUnseal(core, vault.TestKeyCopy(key)); err != nil {
			t.Fatal(err)
		}
	}

	sealed, err := core.Sealed()
	if err != nil {
		t.Fatal(err)
	}
	if sealed {
		t.Fatal("vault should not be sealed")
	}

	ln, addr := http.TestServer(t, core)
	clients := NewClientSet()
	if err := clients.CreateVaultClient(&CreateVaultClientInput{
		Address: addr,
		Token:   token,
	}); err != nil {
		ln.Close()
		t.Fatal(err)
	}

	server := &vaultServer{
		Address: addr,
		Token:   token,
		core:    core,
		ln:      ln,
	}

	return clients, server
}
