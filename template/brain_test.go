package template

import (
	"reflect"
	"testing"

	dep "github.com/hashicorp/consul-template/dependency"
)

func TestNewBrain(t *testing.T) {
	b := NewBrain()

	if b.data == nil {
		t.Errorf("expected data to not be nil")
	}

	if b.receivedData == nil {
		t.Errorf("expected receivedData to not be nil")
	}
}

func TestRecall(t *testing.T) {
	b := NewBrain()

	d := &dep.CatalogNodes{}
	nodes := []*dep.Node{&dep.Node{Node: "node", Address: "address"}}

	b.Remember(d, nodes)
	data, ok := b.Recall(d)

	if !ok {
		t.Fatal("expected data from brain")
	}

	result := data.([]*dep.Node)
	if !reflect.DeepEqual(result, nodes) {
		t.Errorf("expected %#v to be %#v", result, nodes)
	}
}

func TestForceSet(t *testing.T) {
	b := NewBrain()

	d := &dep.CatalogNodes{}
	nodes := []*dep.Node{&dep.Node{Node: "node", Address: "address"}}

	b.ForceSet(d.HashCode(), nodes)
	data, ok := b.Recall(d)

	if !ok {
		t.Fatal("expected data from brain")
	}

	result := data.([]*dep.Node)
	if !reflect.DeepEqual(result, nodes) {
		t.Errorf("expected %#v to be %#v", result, nodes)
	}
}

func TestForget(t *testing.T) {
	b := NewBrain()

	list := map[dep.Dependency]interface{}{
		&dep.CatalogNodes{}:    []*dep.Node{},
		&dep.CatalogServices{}: []*dep.CatalogService{},
		&dep.File{}:            "",
		&dep.HealthServices{}:  []*dep.HealthService{},
		&dep.StoreKey{}:        "",
		&dep.StoreKeyPrefix{}:  []*dep.KeyPair{},
	}

	for d, data := range list {
		b.Remember(d, data)
	}

	for d, _ := range list {
		b.Forget(d)
		if _, ok := b.Recall(d); ok {
			t.Errorf("expected %#v to not be forgotten", d)
		}
	}
}
