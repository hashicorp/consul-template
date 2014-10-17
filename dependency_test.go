package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestServiceDependencyFetch(t *testing.T) {
	client, options := demoConsulClient(t)
	dep := &ServiceDependency{
		rawKey: "consul",
		Name:   "consul",
	}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.([]*Service)
	if !ok {
		t.Fatal("could not convert result to []*Service")
	}
}

func TestServiceDependencyHashCode_isUnique(t *testing.T) {
	dep1 := &ServiceDependency{rawKey: "redis@nyc1"}
	dep2 := &ServiceDependency{rawKey: "redis@nyc2"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseServiceDependency_emptyString(t *testing.T) {
	_, err := ParseServiceDependency("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "cannot specify empty service dependency"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseServiceDependency_name(t *testing.T) {
	sd, err := ParseServiceDependency("webapp")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "webapp",
		Name:   "webapp",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseServiceDependency_nameTag(t *testing.T) {
	sd, err := ParseServiceDependency("release.webapp")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "release.webapp",
		Name:   "webapp",
		Tag:    "release",
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseServiceDependency_nameTagDataCenter(t *testing.T) {
	sd, err := ParseServiceDependency("release.webapp@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey:     "release.webapp@nyc1",
		Name:       "webapp",
		Tag:        "release",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseServiceDependency_nameTagDataCenterPort(t *testing.T) {
	sd, err := ParseServiceDependency("release.webapp@nyc1:8500")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey:     "release.webapp@nyc1:8500",
		Name:       "webapp",
		Tag:        "release",
		DataCenter: "nyc1",
		Port:       8500,
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseServiceDependency_dataCenterOnly(t *testing.T) {
	_, err := ParseServiceDependency("@nyc1")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "invalid service dependency format"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseServiceDependency_nameAndPort(t *testing.T) {
	sd, err := ParseServiceDependency("webapp:8500")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "webapp:8500",
		Name:   "webapp",
		Port:   8500,
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseServiceDependency_nameAndDataCenter(t *testing.T) {
	sd, err := ParseServiceDependency("webapp@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey:     "webapp@nyc1",
		Name:       "webapp",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestKeyDependencyFetch(t *testing.T) {
	client, options := demoConsulClient(t)
	dep := &KeyDependency{
		rawKey: "global/time",
		Path:   "global/time",
	}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.(string)
	if !ok {
		t.Fatal("could not convert result to string")
	}
}

func TestKeyDependencyHashCode_isUnique(t *testing.T) {
	dep1 := &KeyDependency{rawKey: "config/redis/maxconns"}
	dep2 := &KeyDependency{rawKey: "config/redis/minconns"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseKeyDependency_emptyString(t *testing.T) {
	_, err := ParseKeyDependency("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "cannot specify empty key dependency"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseKeyDependency_name(t *testing.T) {
	sd, err := ParseKeyDependency("config/redis/maxconns")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyDependency{
		rawKey: "config/redis/maxconns",
		Path:   "config/redis/maxconns",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseKeyDependency_nameTagDataCenter(t *testing.T) {
	sd, err := ParseKeyDependency("config/redis/maxconns@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyDependency{
		rawKey:     "config/redis/maxconns@nyc1",
		Path:       "config/redis/maxconns",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestKeyPrefixDependencyFetch(t *testing.T) {
	client, options := demoConsulClient(t)
	dep := &KeyPrefixDependency{
		rawKey: "global",
		Prefix: "global",
	}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.([]*KeyPair)
	if !ok {
		t.Fatal("could not convert result to []*KeyPair")
	}
}

func TestKeyPrefixDependencyHashCode_isUnique(t *testing.T) {
	dep1 := &KeyPrefixDependency{rawKey: "config/redis"}
	dep2 := &KeyPrefixDependency{rawKey: "config/consul"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseKeyPrefixDependency_emptyString(t *testing.T) {
	kpd, err := ParseKeyPrefixDependency("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %#v to equal %#v", kpd, expected)
	}
}

func TestParseKeyPrefixDependency_name(t *testing.T) {
	kpd, err := ParseKeyPrefixDependency("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{
		rawKey: "config/redis",
		Prefix: "config/redis",
	}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %#v to equal %#v", kpd, expected)
	}
}

func TestParseKeyPrefixDependency_nameTagDataCenter(t *testing.T) {
	kpd, err := ParseKeyPrefixDependency("config/redis@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{
		rawKey:     "config/redis@nyc1",
		Prefix:     "config/redis",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %#v to equal %#v", kpd, expected)
	}
}

func TestParseKeyPrefixDependency_dataCenter(t *testing.T) {
	kpd, err := ParseKeyPrefixDependency("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %#v to equal %#v", kpd, expected)
	}
}
