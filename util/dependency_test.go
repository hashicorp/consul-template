package util

import (
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

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
		rawKey: "webapp",
		Name:   "webapp",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseServiceDependency_slashName(t *testing.T) {
	sd, err := ParseServiceDependency("web/app")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "web/app",
		Name:   "web/app",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseServiceDependency_underscoreName(t *testing.T) {
	sd, err := ParseServiceDependency("web_app")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "web_app",
		Name:   "web_app",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %#v to equal %#v", sd, expected)
	}
}

func TestParseServiceDependency_dotTag(t *testing.T) {
	sd, err := ParseServiceDependency("first.release.webapp")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "first.release.webapp",
		Name:   "webapp",
		Tag:    "first.release",
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
		t.Errorf("expected %#v to equal %#v", sd, expected)
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
