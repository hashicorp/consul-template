package dependency

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/vault/builtin/logical/pki"
	"github.com/hashicorp/vault/builtin/logical/transit"
	"github.com/hashicorp/vault/helper/namespace"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/physical/inmem"
	"github.com/hashicorp/vault/vault"

	hclog "github.com/hashicorp/go-hclog"
	logicalKv "github.com/hashicorp/vault-plugin-secrets-kv"
)

var testConsul *testutil.TestServer
var testClients *ClientSet

func TestMain(m *testing.M) {
	consul, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	log.SetOutput(ioutil.Discard)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to start consul server: %v", err))
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

	serviceMetaService := &api.AgentServiceRegistration{
		ID:   "service-meta",
		Name: "service-meta",
		Tags: []string{"tag1"},
		Meta: map[string]string{
			"meta1": "value1",
		},
	}

	if err := testClients.consul.client.Agent().ServiceRegister(serviceMetaService); err != nil {
		panic(err)
	}

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
	_, err := s.core.HandleRequest(namespace.RootContext(context.Background()), req)
	return err
}

// testVaultServer is a helper for creating a Vault server and returning the
// appropriate client to connect to it.
func testVaultServer(t *testing.T) (*ClientSet, *vaultServer) {
	inm, err := inmem.NewInmem(nil, hclog.NewNullLogger())
	if err != nil {
		t.Fatal(err)
	}

	core, err := vault.NewCore(&vault.CoreConfig{
		DisableMlock:    true,
		DisableCache:    true,
		DefaultLeaseTTL: 2 * time.Second,
		MaxLeaseTTL:     3 * time.Second,
		Logger:          hclog.NewNullLogger(),
		Physical:        inm,
		LogicalBackends: map[string]logical.Factory{
			"pki":     pki.Factory,
			"transit": transit.Factory,
			"kv":      logicalKv.Factory,
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

	sealed := core.Sealed()
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
