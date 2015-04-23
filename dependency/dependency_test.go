package dependency

import (
	"reflect"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
)

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
	consul := testutil.NewTestServer(t)

	config := consulapi.DefaultConfig()
	config.Address = consul.HTTPAddr
	client, err := consulapi.NewClient(config)
	if err != nil {
		defer consul.Stop()
		t.Fatal(err)
	}

	clients := NewClientSet()
	if err := clients.Add(client); err != nil {
		defer consul.Stop()
		t.Fatal(err)
	}

	return clients, consul
}
