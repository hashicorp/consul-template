package dependency

import (
	"reflect"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	vaultapi "github.com/hashicorp/vault/api"
)

func TestNewClientSet(t *testing.T) {
	clients := NewClientSet()

	if clients.consul != nil {
		t.Errorf("expected %#v to be nil", clients.consul)
	}

	if clients.vault != nil {
		t.Errorf("expected %#v to be nil", clients.vault)
	}
}

func TestAdd_consulClient(t *testing.T) {
	clients := NewClientSet()

	consul := &consulapi.Client{}
	if err := clients.Add(consul); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(clients.consul, consul) {
		t.Errorf("expected %#v to be %#v", clients.consul, consul)
	}
}

func TestAdd_consulClientExists(t *testing.T) {
	clients := &ClientSet{consul: &consulapi.Client{}}

	consul := &consulapi.Client{}
	err := clients.Add(consul)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "a consul client already exists"
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestAdd_vaultClient(t *testing.T) {
	clients := NewClientSet()

	vault := &vaultapi.Client{}
	if err := clients.Add(vault); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(clients.vault, vault) {
		t.Errorf("expected %#v to be %#v", clients.vault, vault)
	}
}

func TestAdd_vaultClientExists(t *testing.T) {
	clients := &ClientSet{vault: &vaultapi.Client{}}

	vault := &vaultapi.Client{}
	err := clients.Add(vault)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "a vault client already exists"
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestConsul_exists(t *testing.T) {
	consul := &consulapi.Client{}
	clients := &ClientSet{consul: consul}

	back, err := clients.Consul()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(back, consul) {
		t.Errorf("expected %#v to be %#v", back, consul)
	}
}

func TestConsul_missing(t *testing.T) {
	clients := NewClientSet()

	_, err := clients.Consul()
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "clientset: missing consul client"
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestVault_exists(t *testing.T) {
	vault := &vaultapi.Client{}
	clients := &ClientSet{vault: vault}

	back, err := clients.Vault()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(back, vault) {
		t.Errorf("expected %#v to be %#v", back, vault)
	}
}

func TestVault_missing(t *testing.T) {
	clients := NewClientSet()

	_, err := clients.Vault()
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "clientset: missing vault client"
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}
