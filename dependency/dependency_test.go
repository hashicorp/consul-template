// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/proto-public/pbresource"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	nomadapi "github.com/hashicorp/nomad/api"
	vapi "github.com/hashicorp/vault/api"
)

const (
	vaultAddr  = "http://127.0.0.1:8200"
	vaultToken = "a_token"
)

var (
	testConsul    *testutil.TestServer
	testVault     *vaultServer
	testNomad     *nomadServer
	testClients   *ClientSet
	tenancyHelper *test.TenancyHelper
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	nomadFuture := runTestNomad()
	runTestVault()
	tb := &test.TestingTB{}
	runTestConsul(tb)
	clients := NewClientSet()
	if err := clients.CreateConsulClient(&CreateConsulClientInput{
		Address: testConsul.HTTPAddr,
	}); err != nil {
		testConsul.Stop()
		testVault.Stop()
		testNomad.Stop()
		Fatalf("failed to create consul client: %v\n", err)
	}

	if t, err := test.NewTenancyHelper(clients.Consul()); err != nil {
		testConsul.Stop()
		testVault.Stop()
		testNomad.Stop()
		Fatalf("failed to create tenancy helper: %v\n", err)
	} else {
		tenancyHelper = t
	}

	if err := clients.CreateVaultClient(&CreateVaultClientInput{
		Address: vaultAddr,
		Token:   vaultToken,
	}); err != nil {
		testConsul.Stop()
		testVault.Stop()
		Fatalf("failed to create vault client: %v\n", err)
	}
	if err := clients.CreateNomadClient(&CreateNomadClientInput{
		Address: "http://127.0.0.1:4646",
	}); err != nil {
		testConsul.Stop()
		testVault.Stop()
		testNomad.Stop()
		Fatalf("failed to create nomad client: %v\n", err)
	}

	testClients = clients

	if err := testClients.createConsulPartitions(); err != nil {
		testConsul.Stop()
		testVault.Stop()
		testNomad.Stop()
		Fatalf("failed to create consul partitions: %v\n", err)
	}

	if err := testClients.createConsulNs(); err != nil {
		testConsul.Stop()
		testVault.Stop()
		testNomad.Stop()
		Fatalf("failed to create consul namespaces: %v\n", err)
	}

	setupVaultPKI(clients)

	if err := testClients.createConsulTestResources(); err != nil {
		testConsul.Stop()
		testVault.Stop()
		testNomad.Stop()
		Fatalf("failed to create consul test resources: %v\n", err)
	}

	// Wait for Nomad initialization to finish
	if err := <-nomadFuture; err != nil {
		testConsul.Stop()
		testNomad.Stop()
		testVault.Stop()
		Fatalf("failed to start Nomad: %v\n", err)
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
				testNomad.Stop()
				panic(r)
			}
		}()

		exitCh <- m.Run()
	}()

	exit := <-exitCh

	tb.DoCleanup()
	testConsul.Stop()
	testVault.Stop()
	testNomad.Stop()
	os.Exit(exit)
}

func (c *ClientSet) createConsulTestResources() error {
	catalog := testClients.Consul().Catalog()

	node, err := testClients.Consul().Agent().NodeName()
	if err != nil {
		return err
	}

	for _, tenancy := range tenancyHelper.TestTenancies() {
		// service with meta data
		serviceMetaService := &api.AgentService{
			ID:      fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
			Service: fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
			Tags:    []string{"tag1"},
			Meta: map[string]string{
				"meta1": "value1",
			},
			Namespace: tenancy.Namespace,
			Partition: tenancy.Partition,
		}
		if _, err := catalog.Register(&api.CatalogRegistration{
			Service:   serviceMetaService,
			Partition: tenancy.Partition,
			Node:      node,
			Address:   "127.0.0.1",
		}, nil); err != nil {
			return err
		}
		// service with serviceTaggedAddresses
		serviceTaggedAddressesService := &api.AgentService{
			ID:      fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
			Service: fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
			TaggedAddresses: map[string]api.ServiceAddress{
				"lan": {
					Address: "192.0.2.1",
					Port:    80,
				},
				"wan": {
					Address: "192.0.2.2",
					Port:    443,
				},
			},
			Namespace: tenancy.Namespace,
			Partition: tenancy.Partition,
		}
		if _, err := catalog.Register(&api.CatalogRegistration{
			Service:   serviceTaggedAddressesService,
			Partition: tenancy.Partition,
			Node:      node,
			Address:   "127.0.0.1",
		}, nil); err != nil {
			return err
		}

		// connect enabled service
		testService := &api.AgentService{
			ID:        fmt.Sprintf("conn-enabled-service-%s-%s", tenancy.Partition, tenancy.Namespace),
			Service:   fmt.Sprintf("conn-enabled-service-%s-%s", tenancy.Partition, tenancy.Namespace),
			Port:      12345,
			Connect:   &api.AgentServiceConnect{},
			Namespace: tenancy.Namespace,
			Partition: tenancy.Partition,
		}
		// this is based on what `consul connect proxy` command does at
		// consul/command/connect/proxy/register.go (register method)
		testConnect := &api.AgentService{
			Kind:    api.ServiceKindConnectProxy,
			ID:      fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
			Service: fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
			Port:    21999,
			Proxy: &api.AgentServiceConnectProxyConfig{
				DestinationServiceName: fmt.Sprintf("conn-enabled-service-%s-%s", tenancy.Partition, tenancy.Namespace),
			},
			Namespace: tenancy.Namespace,
			Partition: tenancy.Partition,
		}

		if _, err := catalog.Register(&api.CatalogRegistration{
			Service:   testService,
			Partition: tenancy.Partition,
			Node:      node,
			Address:   "127.0.0.1",
		}, nil); err != nil {
			return err
		}

		if _, err := catalog.Register(&api.CatalogRegistration{
			Service:   testConnect,
			Partition: tenancy.Partition,
			Node:      node,
			Address:   "127.0.0.1",
		}, nil); err != nil {
			return err
		}

		if err := testClients.createConsulPeerings(tenancy); err != nil {
			return err
		}
	}

	return nil
}

func (c *ClientSet) createConsulPeerings(tenancy *pbresource.Tenancy) error {
	generateReq := api.PeeringGenerateTokenRequest{PeerName: "foo", Partition: tenancy.Partition}
	_, _, err := c.consul.client.Peerings().GenerateToken(context.Background(), generateReq, &api.WriteOptions{})
	if err != nil {
		return err
	}

	generateReq = api.PeeringGenerateTokenRequest{PeerName: "bar", Partition: tenancy.Partition}
	_, _, err = c.consul.client.Peerings().GenerateToken(context.Background(), generateReq, &api.WriteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func runTestConsul(tb testutil.TestingTB) {
	consul, err := testutil.NewTestServerConfigT(tb,
		func(c *testutil.TestServerConfig) {
			c.LogLevel = "warn"
			c.Stdout = io.Discard
			c.Stderr = io.Discard
		})
	if err != nil {
		Fatalf("failed to start consul server: %v", err)
	}
	testConsul = consul
}

// runTestNomad starts a Nomad agent and returns a chan which will block until
// initialization is complete or fails. Stop() is safe to call after the chan
// is returned.
func runTestNomad() <-chan error {
	path, err := exec.LookPath("nomad")
	if err != nil || path == "" {
		Fatalf("nomad not found on $PATH")
	}
	cmd := exec.Command(path, "agent", "-dev",
		"-node=test",
		"-vault-enabled=false",
		"-consul-auto-advertise=false",
		"-consul-client-auto-join=false", "-consul-server-auto-join=false",
		"-network-speed=100",
		"-log-level=error", // We're just discarding it anyway
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		Fatalf("nomad failed to start: %v", err)
	}
	testNomad = &nomadServer{
		cmd: cmd,
	}

	errCh := make(chan error, 1)
	go initTestNomad(errCh)

	return errCh
}

func initTestNomad(errCh chan<- error) {
	defer close(errCh)

	// Load a job with a Nomad service. Use a JSON formatted job to avoid
	// an additional dependency upon Nomad's jobspec package or having to
	// wait for the agent to be up before the job can be parsed.
	fd, err := os.Open("../test/testdata/nomad.json")
	if err != nil {
		errCh <- fmt.Errorf("error opening test job: %w", err)
		return
	}
	var job nomadapi.Job
	if err := json.NewDecoder(fd).Decode(&job); err != nil {
		errCh <- fmt.Errorf("error parsing test job: %w", err)
		return
	}

	config := nomadapi.DefaultConfig()
	client, err := nomadapi.NewClient(config)
	if err != nil {
		errCh <- fmt.Errorf("failed to create nomad client: %w", err)
		return
	}

	// Wait for API to become available
	for e := time.Now().Add(30 * time.Second); time.Now().Before(e); {
		var self *nomadapi.AgentSelf
		self, err = client.Agent().Self()
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		fmt.Printf("Nomad v%s running on %s\n", self.Member.Tags["build"], config.Address)
		break
	}
	if err != nil {
		errCh <- fmt.Errorf("failed to contact nomad agent: %w", err)
		return
	}

	// Register a job
	if _, _, err := client.Jobs().Register(&job, nil); err != nil {
		errCh <- fmt.Errorf("failed registering nomad job: %w", err)
		return
	}

	// Wait for it start
	var allocs []*nomadapi.AllocationListStub
	for e := time.Now().Add(30 * time.Second); time.Now().Before(e); {
		allocs, _, err = client.Jobs().Allocations(*job.ID, true, nil)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if n := len(allocs); n > 1 {
			errCh <- fmt.Errorf("expected 1 nomad alloc but found: %d\n%s\n%s",
				n,
				compileTaskStates(allocs[0]),
				compileTaskStates(allocs[1]),
			)
			return
		} else if n == 0 {
			err = fmt.Errorf("expected 1 nomad alloc but found none")
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if s := allocs[0].ClientStatus; s != "running" {
			err = fmt.Errorf("expected nomad alloc running but found %q\n%s",
				s, compileTaskStates(allocs[0]),
			)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	if err != nil {
		errCh <- fmt.Errorf("failed to start nomad job: %w", err)
		return
	}
	fmt.Printf("Nomad started: %s\n", compileTaskStates(allocs[0]))
}

func compileTaskStates(a *nomadapi.AllocationListStub) string {
	out := ""
	for name, state := range a.TaskStates {
		out += fmt.Sprintf("%s: [", name)
		for i, e := range state.Events {
			out += e.Type
			if i != len(state.Events)-1 {
				out += ", "
			}
		}
		out += "] "
	}
	return out
}

type nomadServer struct {
	cmd *exec.Cmd
}

func (n *nomadServer) Stop() error {
	if n == nil || n.cmd == nil || n.cmd.Process == nil {
		fmt.Println("No Nomad process to stop")
		return nil
	}

	fmt.Println("Signalling Nomad")
	n.cmd.Process.Signal(os.Interrupt)
	return n.cmd.Wait()
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
	args := []string{
		"server", "-dev", "-dev-root-token-id", vaultToken,
		"-dev-no-store-token",
	}
	cmd := exec.Command("vault", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

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
	tags := []string{"hello", "world", "these", "are", "tags", "foo:bar", "baz=qux"}
	expected := []string{"are", "baz=qux", "foo:bar", "hello", "tags", "these", "world"}

	result := deepCopyAndSortTags(tags)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v to be %#v", result, expected)
	}
}

func Fatalf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	runtime.Goexit()
}

func (v *nomadServer) CreateVariable(path string, data map[string]string, opts *nomadapi.WriteOptions) error {
	nVar := nomadapi.NewVariable(path)
	for k, v := range data {
		nVar.Items[k] = v
	}
	_, _, err := testClients.Nomad().Variables().Update(nVar, opts)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (v *nomadServer) CreateNamespace(name string, opts *nomadapi.WriteOptions) error {
	ns := nomadapi.Namespace{Name: name}
	_, err := testClients.Nomad().Namespaces().Register(&ns, opts)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (v *nomadServer) DeleteVariable(path string, opts *nomadapi.WriteOptions) error {
	_, err := testClients.Nomad().Variables().Delete(path, opts)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (c *ClientSet) createConsulPartitions() error {
	for p := range tenancyHelper.GetUniquePartitions() {
		if p.Name != "" && p.Name != "default" {
			partition := &api.Partition{Name: p.Name}
			_, _, err := c.Consul().Partitions().Create(context.Background(), partition, nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *ClientSet) createConsulNs() error {
	for _, tenancy := range tenancyHelper.TestTenancies() {
		if tenancy.Namespace != "" && tenancy.Namespace != "default" {
			ns := &api.Namespace{Name: tenancy.Namespace, Partition: tenancy.Partition}
			_, _, err := c.Consul().Namespaces().Create(ns, nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
