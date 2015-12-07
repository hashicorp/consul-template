package main

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul/testutil"
)

func testDedupManager(t *testing.T, templ []*Template) (*testutil.TestServer, *DedupManager) {
	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})

	// Setup the configuration
	config := DefaultConfig()
	config.Consul = consul.HTTPAddr

	// Create the clientset
	clients, err := newClientSet(config)
	if err != nil {
		t.Fatalf("runner: %s", err)
	}

	// Setup a brain
	brain := NewBrain()

	// Create the dedup manager
	dedup, err := NewDedupManager(config, clients, brain, templ)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return consul, dedup
}

func testDedupFollower(t *testing.T, leader *DedupManager) *DedupManager {
	// Setup the configuration
	config := DefaultConfig()
	config.Consul = leader.config.Consul

	// Create the clientset
	clients, err := newClientSet(config)
	if err != nil {
		t.Fatalf("runner: %s", err)
	}

	// Setup a brain
	brain := NewBrain()

	// Create the dedup manager
	dedup, err := NewDedupManager(config, clients, brain, leader.templates)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return dedup
}

func TestDedup_StartStop(t *testing.T) {
	t.Parallel()

	consul, dedup := testDedupManager(t, nil)
	defer consul.Stop()

	// Start and stop
	if err := dedup.Start(); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := dedup.Stop(); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestDedup_IsLeader(t *testing.T) {
	t.Parallel()

	// Create a template
	in := test.CreateTempfile([]byte(`
    {{ range service "consul" }}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)
	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	consul, dedup := testDedupManager(t, []*Template{tmpl})
	defer consul.Stop()

	// Start dedup
	if err := dedup.Start(); err != nil {
		t.Fatalf("err: %v", err)
	}
	defer dedup.Stop()

	// Wait until we are leader
	select {
	case <-dedup.UpdateCh():
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout")
	}

	// Check that we are the leader
	if !dedup.IsLeader(tmpl) {
		t.Fatalf("should be leader")
	}
}

func TestDedup_UpdateDeps(t *testing.T) {
	t.Parallel()

	// Create a template
	in := test.CreateTempfile([]byte(`
    {{ range service "consul" }}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)
	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	consul, dedup := testDedupManager(t, []*Template{tmpl})
	defer consul.Stop()

	// Start dedup
	if err := dedup.Start(); err != nil {
		t.Fatalf("err: %v", err)
	}
	defer dedup.Stop()

	// Wait until we are leader
	select {
	case <-dedup.UpdateCh():
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout")
	}

	// Create the dependency
	dep, err := dependency.ParseHealthServices("consul")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Inject data into the brain
	dedup.brain.Remember(dep, 123)

	// Update the dependencies
	err = dedup.UpdateDeps(tmpl, []dependency.Dependency{dep})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestDedup_FollowerUpdate(t *testing.T) {
	t.Parallel()

	// Create a template
	in := test.CreateTempfile([]byte(`
    {{ range service "consul" }}{{.Node}}{{ end }}
  `), t)
	defer test.DeleteTempfile(in, t)
	tmpl, err := NewTemplate(in.Name())
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	consul, dedup1 := testDedupManager(t, []*Template{tmpl})
	defer consul.Stop()

	dedup2 := testDedupFollower(t, dedup1)

	// Start dedups
	if err := dedup1.Start(); err != nil {
		t.Fatalf("err: %v", err)
	}
	defer dedup1.Stop()
	if err := dedup2.Start(); err != nil {
		t.Fatalf("err: %v", err)
	}
	defer dedup2.Stop()

	// Wait until we have a leader
	var leader, follow *DedupManager
	select {
	case <-dedup1.UpdateCh():
		if dedup1.IsLeader(tmpl) {
			leader = dedup1
			follow = dedup2
		}
	case <-dedup2.UpdateCh():
		if dedup2.IsLeader(tmpl) {
			leader = dedup2
			follow = dedup1
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout")
	}

	// Create the dependency
	dep, err := dependency.ParseHealthServices("consul")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Inject data into the brain
	leader.brain.Remember(dep, 123)

	// Update the dependencies
	err = leader.UpdateDeps(tmpl, []dependency.Dependency{dep})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Follower should get an update
	select {
	case <-follow.UpdateCh():
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout")
	}

	// Recall from the brain
	data, ok := follow.brain.Recall(dep)
	if !ok {
		t.Fatalf("missing data")
	}
	if data != 123 {
		t.Fatalf("bad: %v", data)
	}
}
