package dependency

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
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

var testConsul *testutil.TestServer
var testClients *ClientSet

func TestMain(m *testing.M) {
	consul, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	if err != nil {
		log.Fatal("failed to start consul server")
	}
	testConsul = consul

	clients := NewClientSet()
	if err := clients.CreateConsulClient(&CreateConsulClientInput{
		Address: testConsul.HTTPAddr,
	}); err != nil {
		testConsul.Stop()
		log.Fatal(err)
	}
	testClients = clients

	exitCh := make(chan int, 1)
	func() {
		defer func() {
			// Attempt to recover from a panic and stop the server. If we don't stop
			// it, the panic will cause the server to remain running in the
			// background. Here we catch the panic and the re-raise it.
			if r := recover(); r != nil {
				testConsul.Stop()
				panic(r)
			}
		}()

		exitCh <- m.Run()
	}()

	exit := <-exitCh

	testConsul.Stop()
	os.Exit(exit)
}

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
