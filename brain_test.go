package main

import (
	"testing"

	dep "github.com/hashicorp/consul-template/dependency"
)

func TestNewBrain(t *testing.T) {
	b := NewBrain()

	if b.catalogNodes == nil {
		t.Errorf("expected catalogNodes to not be nil")
	}

	if b.catalogServices == nil {
		t.Errorf("expected catalogServices to not be nil")
	}

	if b.files == nil {
		t.Errorf("expected files to not be nil")
	}

	if b.healthServices == nil {
		t.Errorf("expected healthServices to not be nil")
	}

	if b.storeKeys == nil {
		t.Errorf("expected storeKeys to not be nil")
	}

	if b.storeKeyPrefixes == nil {
		t.Errorf("expected storeKeyPrefixes to not be nil")
	}

	if b.receivedData == nil {
		t.Errorf("expected receivedData to not be nil")
	}
}

func TestRemember(t *testing.T) {
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
		if !b.Remembered(d) {
			t.Errorf("expected %#v to be remembered", d)
		}
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
		if b.Remembered(d) {
			t.Errorf("expected %#v to not be forgotten", d)
		}
	}
}
