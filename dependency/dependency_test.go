package dependency

import (
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/vault"
)

func TestCanShare(t *testing.T) {
	vs := &VaultSecret{}
	vt := &VaultToken{}
	file := &File{}
	service := &HealthServices{}

	if CanShare(vs) {
		t.Fatalf("should not share vault")
	}
	if CanShare(vt) {
		t.Fatalf("should not share vault")
	}
	if CanShare(file) {
		t.Fatalf("should not share file")
	}
	if !CanShare(service) {
		t.Fatalf("should share service")
	}
}

func TestDeepCopyAndSortTags(t *testing.T) {
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
	t.Parallel()

	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})

	config := consulapi.DefaultConfig()
	config.Address = consul.HTTPAddr
	client, err := consulapi.NewClient(config)
	if err != nil {
		consul.Stop()
		t.Fatalf("consul api client err: %s", err)
	}

	clients := NewClientSet()
	if err := clients.Add(client); err != nil {
		consul.Stop()
		t.Fatalf("clientset err: %s", err)
	}

	return clients, consul
}

type vaultServer struct {
	Token string

	core *vault.Core
	ln   net.Listener
}

func (s *vaultServer) Stop() {
	s.ln.Close()
}

func (s *vaultServer) CreateSecret(path string, data map[string]interface{}) error {
	req := &logical.Request{
		Operation:   logical.WriteOperation,
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
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := http.TestServer(t, core)

	config := vaultapi.DefaultConfig()
	config.Address = addr
	client, err := vaultapi.NewClient(config)
	if err != nil {
		defer ln.Close()
		t.Fatal(err)
	}

	client.SetToken(token)

	clients := NewClientSet()
	if err := clients.Add(client); err != nil {
		defer ln.Close()
		t.Fatal(err)
	}

	return clients, &vaultServer{Token: token, core: core, ln: ln}
}
