package dependency

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	vapi "github.com/hashicorp/vault/api"
)

const vaultAddr = "http://127.0.0.1:8200"
const vaultToken = "a_token"

var testConsul *testutil.TestServer
var testVault *vaultServer
var testClients *ClientSet

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	runTestVault()
	runTestConsul()
	clients := NewClientSet()
	if err := clients.CreateConsulClient(&CreateConsulClientInput{
		Address: testConsul.HTTPAddr,
	}); err != nil {
		testConsul.Stop()
		Fatalf("failed to create consul client: %v\n", err)
	}
	if err := clients.CreateVaultClient(&CreateVaultClientInput{
		Address: vaultAddr,
		Token:   vaultToken,
	}); err != nil {
		testVault.Stop()
		Fatalf("failed to create vault client: %v\n", err)
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

	consul_agent := testClients.consul.client.Agent()
	if err := consul_agent.ServiceRegister(serviceMetaService); err != nil {
		Fatalf("%v", err)
	}

	exitCh := make(chan int, 1)
	func() {
		defer func() {
			// Attempt to recover from a panic and stop the server. If we don't
			// stop it, the panic will cause the server to remain running in
			// the background. Here we catch the panic and the re-raise it.
			if r := recover(); r != nil {
				testConsul.Stop()
				testVault.Stop()
				panic(r)
			}
		}()

		exitCh <- m.Run()
	}()

	exit := <-exitCh

	testConsul.Stop()
	testVault.Stop()
	os.Exit(exit)
}

func runTestConsul() {
	consul, err := testutil.NewTestServerConfig(
		func(c *testutil.TestServerConfig) {
			c.LogLevel = "warn"
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard
		})
	if err != nil {
		Fatalf("failed to start consul server: %v", err)
	}
	testConsul = consul
}

type vaultServer struct {
	secretsPath string
	cmd         *exec.Cmd
}

func runTestVault() {
	path, err := exec.LookPath("vault")
	if err != nil || path == "" {
		Fatalf("vault not found on $PATH")
	}
	args := []string{"server", "-dev", "-dev-root-token-id", vaultToken}
	cmd := exec.Command("vault", args...)
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard

	if err := cmd.Start(); err != nil {
		Fatalf("vault failed to start: %v", err)
	}
	testVault = &vaultServer{
		cmd: cmd,
	}
}

func (v vaultServer) Stop() error {
	if v.cmd != nil && v.cmd.Process != nil {
		return v.cmd.Process.Signal(os.Interrupt)
	}
	return nil
}

func testVaultServer(t *testing.T, secrets_path, version string,
) (*ClientSet, *vaultServer) {
	vc := testClients.Vault()
	if err := vc.Sys().Mount(secrets_path, &vapi.MountInput{
		Type:        "kv",
		Description: "test mount",
		Options:     map[string]string{"version": version},
	}); err != nil {
		fmt.Println(err)
		t.Fatalf("Error creating secrets engine: %s", err)
	}
	return testClients, &vaultServer{secretsPath: secrets_path}
}

func (v *vaultServer) CreateSecret(path string, data map[string]interface{},
) error {
	q, err := NewVaultWriteQuery(v.secretsPath+"/"+path, data)
	if err != nil {
		return err
	}
	_, err = q.writeSecret(testClients, &QueryOptions{})
	if err != nil {
		fmt.Println(err)
	}
	return err
}

// deleteSecret lets us delete keys as needed for tests
func (v *vaultServer) deleteSecret(path string) error {
	_, err := testClients.Vault().Logical().Delete(v.secretsPath + "/" + path)
	if err != nil {
		fmt.Println(err)
	}
	return err
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

func Fatalf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	runtime.Goexit()
}
