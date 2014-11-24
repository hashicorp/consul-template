package util

import (
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	api "github.com/armon/consul-api"
	"github.com/hashicorp/consul-template/test"
)

func TestServiceDependencyFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
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

func TestServiceStatusFilter_onlyAllowPassing(t *testing.T) {
	var f ServiceStatusFilter
	f = ServiceStatusFilter{}
	if f.onlyAllowPassing() {
		t.Fatal("Expecting and empty status filter to return false for onlyAllowPassing.")
	}
	f = ServiceStatusFilter{HealthPassing}
	if !f.onlyAllowPassing() {
		t.Fatal("Expecting and empty status filter to return true for onlyAllowPassing.")
	}
	f = ServiceStatusFilter{HealthPassing, HealthWarning}
	if f.onlyAllowPassing() {
		t.Fatal("Expecting passing and warning status filter to return false for onlyAllowPassing.")
	}
	f = ServiceStatusFilter{HealthWarning}
	if f.onlyAllowPassing() {
		t.Fatal("Expecting warning status filter to return false for onlyAllowPassing.")
	}
}

func TestServiceStatusFilter_acceptWithEmptyFilterReturnsTrue(t *testing.T) {
	f := &ServiceStatusFilter{}
	c0 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthUnknown,
		},
		&api.HealthCheck{
			Status: HealthPassing,
		},
		&api.HealthCheck{
			Status: HealthWarning,
		},
		&api.HealthCheck{
			Status: HealthCritical,
		},
	}
	c1 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthUnknown,
		},
	}
	c2 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthPassing,
		},
	}
	c3 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthWarning,
		},
	}
	c4 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthCritical,
		},
	}
	if !f.accept(c0) {
		t.Fatal("Expecting empty filter to accept c0.")
	}
	if !f.accept(c1) {
		t.Fatal("Expecting empty filter to accept c1.")
	}
	if !f.accept(c2) {
		t.Fatal("Expecting empty filter to accept c2.")
	}
	if !f.accept(c3) {
		t.Fatal("Expecting empty filter to accept c3.")
	}
	if !f.accept(c4) {
		t.Fatal("Expecting empty filter to accept c4.")
	}
}

func TestServiceStatusFilter_acceptOnlyReturnsTrueForItemsInFilter(t *testing.T) {
	f1 := &ServiceStatusFilter{HealthPassing}
	f2 := &ServiceStatusFilter{HealthPassing, HealthWarning}
	c1 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthPassing,
		},
	}
	c2 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthWarning,
		},
	}
	c3 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthPassing,
		},
		&api.HealthCheck{
			Status: HealthWarning,
		},
	}
	c4 := []*api.HealthCheck{
		&api.HealthCheck{
			Status: HealthCritical,
		},
	}
	if !f1.accept(c1) {
		t.Fatal("Expecting f1 to accept c1.")
	}
	if f1.accept(c2) {
		t.Fatal("Expecting f1 to not accept c2.")
	}
	if f1.accept(c3) {
		t.Fatal("Expecting f1 to not accept c3.")
	}
	if f1.accept(c4) {
		t.Fatal("Expecting f1 to not accept c4.")
	}
	if !f2.accept(c1) {
		t.Fatal("Expecting f2 to accept c1.")
	}
	if !f2.accept(c2) {
		t.Fatal("Expecting f2 to accept c2.")
	}
	if !f2.accept(c3) {
		t.Fatal("Expecting f2 to accept c3.")
	}
	if f2.accept(c4) {
		t.Fatal("Expecting f2 to not accept c4.")
	}
}

func TestFileDependencyFetch(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep := &FileDependency{
		rawKey: inTemplate.Name(),
	}

	read, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if read != data {
		t.Fatalf("expected %q to be %q", read, data)
	}
}

func TestFileDependencyFetch_waits(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep := &FileDependency{
		rawKey: inTemplate.Name(),
	}

	_, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan interface{})
	go func(c chan<- interface{}) {
		data, _, err := dep.Fetch(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		c <- data
	}(dataCh)

	select {
	case <-dataCh:
		t.Fatal("received data, but should not have")
	case <-time.After(1000 * time.Nanosecond):
		return
	}
}

func TestFileDependencyFetch_firesChanges(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep := &FileDependency{
		rawKey: inTemplate.Name(),
	}

	_, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan interface{})
	go func(c chan<- interface{}) {
		data, _, err := dep.Fetch(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		c <- data
	}(dataCh)

	newData := `{"bar": "baz"}`
	ioutil.WriteFile(inTemplate.Name(), []byte(newData), 0644)

	select {
	case d := <-dataCh:
		if d != newData {
			t.Fatalf("expected %q to be %q", d, newData)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("did not receive data from file changes")
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
		rawKey: "webapp [passing]",
		Name:   "webapp",
		Status: ServiceStatusFilter{HealthPassing},
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_nameAndStatus(t *testing.T) {
	sd, err := ParseServiceDependency("webapp", "any")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "webapp [any]",
		Name:   "webapp",
		Status: ServiceStatusFilter{},
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_slashName(t *testing.T) {
	sd, err := ParseServiceDependency("web/app")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "web/app [passing]",
		Name:   "web/app",
		Status: ServiceStatusFilter{HealthPassing},
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_underscoreName(t *testing.T) {
	sd, err := ParseServiceDependency("web_app")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "web_app [passing]",
		Name:   "web_app",
		Status: ServiceStatusFilter{HealthPassing},
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_dotTag(t *testing.T) {
	sd, err := ParseServiceDependency("first.release.webapp")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "first.release.webapp [passing]",
		Name:   "webapp",
		Tag:    "first.release",
		Status: ServiceStatusFilter{HealthPassing},
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_nameTag(t *testing.T) {
	sd, err := ParseServiceDependency("release.webapp")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "release.webapp [passing]",
		Name:   "webapp",
		Tag:    "release",
		Status: ServiceStatusFilter{HealthPassing},
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_nameTagDataCenter(t *testing.T) {
	sd, err := ParseServiceDependency("release.webapp@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey:     "release.webapp@nyc1 [passing]",
		Name:       "webapp",
		Tag:        "release",
		DataCenter: "nyc1",
		Status:     ServiceStatusFilter{HealthPassing},
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_nameTagDataCenterPort(t *testing.T) {
	sd, err := ParseServiceDependency("release.webapp@nyc1:8500")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey:     "release.webapp@nyc1:8500 [passing]",
		Name:       "webapp",
		Tag:        "release",
		DataCenter: "nyc1",
		Port:       8500,
		Status:     ServiceStatusFilter{HealthPassing},
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
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
		rawKey: "webapp:8500 [passing]",
		Name:   "webapp",
		Port:   8500,
		Status: ServiceStatusFilter{HealthPassing},
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_nameAndDataCenter(t *testing.T) {
	sd, err := ParseServiceDependency("webapp@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey:     "webapp@nyc1 [passing]",
		Name:       "webapp",
		DataCenter: "nyc1",
		Status:     ServiceStatusFilter{HealthPassing},
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestKeyDependencyFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
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
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestKeyPrefixDependencyFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
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
		t.Errorf("expected %+v to equal %+v", kpd, expected)
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
		t.Errorf("expected %+v to equal %+v", kpd, expected)
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
		t.Errorf("expected %+v to equal %+v", kpd, expected)
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
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestNodeDependencyFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &NodeDependency{
		rawKey: "global",
	}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.([]*Node)
	if !ok {
		t.Fatal("could not convert result to []*Node")
	}
}

func TestNodeDependencyHashCode_isUnique(t *testing.T) {
	dep1 := &NodeDependency{rawKey: ""}
	dep2 := &NodeDependency{rawKey: "@nyc1"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseNodeDependency_emptyString(t *testing.T) {
	nd, err := ParseNodeDependency("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &NodeDependency{}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}

func TestParseNodeDependency_dataCenter(t *testing.T) {
	nd, err := ParseNodeDependency("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &NodeDependency{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}

func TestServiceTagsContains(t *testing.T) {
	s := &Service{
		Node:    "node",
		Address: "127.0.0.1",
		ID:      "id",
		Name:    "name",
		Tags:    []string{"foo", "baz"},
		Port:    1234,
	}
	if !s.Tags.Contains("foo") {
		t.Error("expected Contains to return true for foo.")
	}
	if s.Tags.Contains("bar") {
		t.Error("expected Contains to return false for bar.")
	}
	if !s.Tags.Contains("baz") {
		t.Error("expected Contains to return true for baz.")
	}
}
