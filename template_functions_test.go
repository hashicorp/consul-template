package main

import (
	"os"
	"reflect"
	"testing"

	dep "github.com/hashicorp/consul-template/dependency"
)

func TestDatacentersFunc_emptyString(t *testing.T) {
	brain := NewBrain()
	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := datacentersFunc(brain, used, missing)
	result, err := f()
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestDatacentersFunc_hasData(t *testing.T) {
	d, err := dep.ParseDatacenters()
	if err != nil {
		t.Fatal(err)
	}

	dcs := []string{"dc1", "dc2"}

	brain := NewBrain()
	brain.Remember(d, dcs)

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := datacentersFunc(brain, used, missing)
	result, err := f()
	if err != nil {
		t.Fatal(err)
	}

	expected := dcs
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if len(missing) != 0 {
		t.Errorf("expected missing to have 0 elements, but had %d", len(missing))
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}
}

func TestDatacentersFunc_missingData(t *testing.T) {
	d, err := dep.ParseDatacenters()
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := datacentersFunc(brain, used, missing)
	result, err := f()
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}

	if _, ok := missing[d.HashCode()]; !ok {
		t.Errorf("expected dep to be missing")
	}
}

func TestFileFunc_emptyString(t *testing.T) {
	brain := NewBrain()
	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := fileFunc(brain, used, missing)
	result, err := f("")
	if err != nil {
		t.Fatal(err)
	}

	if result != "" {
		t.Errorf("expected %q to be %q", result, "")
	}
}

func TestFileFunc_hasData(t *testing.T) {
	d, err := dep.ParseFile("/existing/file")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()
	brain.Remember(d, "contents")

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := fileFunc(brain, used, missing)
	result, err := f("/existing/file")
	if err != nil {
		t.Fatal(err)
	}

	if result != "contents" {
		t.Errorf("expected %q to be %q", result, "contents")
	}

	if len(missing) != 0 {
		t.Errorf("expected missing to have 0 elements, but had %d", len(missing))
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}
}

func TestFileFunc_missingData(t *testing.T) {
	d, err := dep.ParseFile("/non-existing/file")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := fileFunc(brain, used, missing)
	result, err := f("/non-existing/file")
	if err != nil {
		t.Fatal(err)
	}

	if result != "" {
		t.Errorf("expected %q to be %q", result, "")
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}

	if _, ok := missing[d.HashCode()]; !ok {
		t.Errorf("expected dep to be missing")
	}
}

func TestKeyFunc_emptyString(t *testing.T) {
	brain := NewBrain()
	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := keyFunc(brain, used, missing)
	result, err := f("")
	if err != nil {
		t.Fatal(err)
	}

	if result != "" {
		t.Errorf("expected %q to be %q", result, "")
	}
}

func TestKeyFunc_hasData(t *testing.T) {
	d, err := dep.ParseStoreKey("existing")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()
	brain.Remember(d, "contents")

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := keyFunc(brain, used, missing)
	result, err := f("existing")
	if err != nil {
		t.Fatal(err)
	}

	if result != "contents" {
		t.Errorf("expected %q to be %q", result, "contents")
	}

	if len(missing) != 0 {
		t.Errorf("expected missing to have 0 elements, but had %d", len(missing))
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}
}

func TestKeyFunc_missingData(t *testing.T) {
	d, err := dep.ParseStoreKey("non-existing")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := keyFunc(brain, used, missing)
	result, err := f("non-existing")
	if err != nil {
		t.Fatal(err)
	}

	if result != "" {
		t.Errorf("expected %q to be %q", result, "")
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}

	if _, ok := missing[d.HashCode()]; !ok {
		t.Errorf("expected dep to be missing")
	}
}

func TestLsFunc_emptyString(t *testing.T) {
	brain := NewBrain()
	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := lsFunc(brain, used, missing)
	result, err := f("")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.KeyPair{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestLsFunc_hasData(t *testing.T) {
	d, err := dep.ParseStoreKeyPrefix("existing")
	if err != nil {
		t.Fatal(err)
	}

	data := []*dep.KeyPair{
		&dep.KeyPair{Key: "", Value: ""},
		&dep.KeyPair{Key: "user/sethvargo", Value: "true"},
		&dep.KeyPair{Key: "maxconns", Value: "11"},
		&dep.KeyPair{Key: "minconns", Value: "2"},
	}

	brain := NewBrain()
	brain.Remember(d, data)

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := lsFunc(brain, used, missing)
	result, err := f("existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.KeyPair{
		&dep.KeyPair{Key: "maxconns", Value: "11"},
		&dep.KeyPair{Key: "minconns", Value: "2"},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if len(missing) != 0 {
		t.Errorf("expected missing to have 0 elements, but had %d", len(missing))
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}
}

func TestLsFunc_missingData(t *testing.T) {
	d, err := dep.ParseStoreKeyPrefix("non-existing")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := lsFunc(brain, used, missing)
	result, err := f("non-existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.KeyPair{}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}

	if _, ok := missing[d.HashCode()]; !ok {
		t.Errorf("expected dep to be missing")
	}
}

func TestNodesFunc_noArgs(t *testing.T) {
	brain := NewBrain()
	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := nodesFunc(brain, used, missing)
	result, err := f()
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.Node{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestNodesFunc_hasData(t *testing.T) {
	d, err := dep.ParseCatalogNodes("@existing")
	if err != nil {
		t.Fatal(err)
	}

	data := []*dep.Node{
		&dep.Node{Node: "a"},
		&dep.Node{Node: "b"},
	}

	brain := NewBrain()
	brain.Remember(d, data)

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := nodesFunc(brain, used, missing)
	result, err := f("@existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := data
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if len(missing) != 0 {
		t.Errorf("expected missing to have 0 elements, but had %d", len(missing))
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}
}

func TestNodesFunc_missingData(t *testing.T) {
	d, err := dep.ParseCatalogNodes("@non-existing")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := nodesFunc(brain, used, missing)
	result, err := f("@non-existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.Node{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}

	if _, ok := missing[d.HashCode()]; !ok {
		t.Errorf("expected dep to be missing")
	}
}

func TestServiceFunc_emptyString(t *testing.T) {
	brain := NewBrain()
	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := serviceFunc(brain, used, missing)
	result, err := f("")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.HealthService{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestServiceFunc_hasData(t *testing.T) {
	d, err := dep.ParseHealthServices("existing")
	if err != nil {
		t.Fatal(err)
	}

	data := []*dep.HealthService{
		&dep.HealthService{Node: "a"},
		&dep.HealthService{Node: "b"},
	}

	brain := NewBrain()
	brain.Remember(d, data)

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := serviceFunc(brain, used, missing)
	result, err := f("existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := data
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if len(missing) != 0 {
		t.Errorf("expected missing to have 0 elements, but had %d", len(missing))
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}
}

func TestServiceFunc_missingData(t *testing.T) {
	d, err := dep.ParseHealthServices("non-existing")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := serviceFunc(brain, used, missing)
	result, err := f("non-existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.HealthService{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}

	if _, ok := missing[d.HashCode()]; !ok {
		t.Errorf("expected dep to be missing")
	}
}

func TestServicesFunc_noArgs(t *testing.T) {
	brain := NewBrain()
	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := servicesFunc(brain, used, missing)
	result, err := f()
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.CatalogService{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestServicesFunc_hasData(t *testing.T) {
	d, err := dep.ParseCatalogServices("@existing")
	if err != nil {
		t.Fatal(err)
	}

	data := []*dep.CatalogService{
		&dep.CatalogService{Name: "a"},
		&dep.CatalogService{Name: "b"},
	}

	brain := NewBrain()
	brain.Remember(d, data)

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := servicesFunc(brain, used, missing)
	result, err := f("@existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := data
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if len(missing) != 0 {
		t.Errorf("expected missing to have 0 elements, but had %d", len(missing))
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}
}

func TestServicesFunc_missingData(t *testing.T) {
	d, err := dep.ParseCatalogServices("@non-existing")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := servicesFunc(brain, used, missing)
	result, err := f("@non-existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.CatalogService{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}

	if _, ok := missing[d.HashCode()]; !ok {
		t.Errorf("expected dep to be missing")
	}
}

func TestTreeFunc_emptyString(t *testing.T) {
	brain := NewBrain()
	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := treeFunc(brain, used, missing)
	result, err := f("")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.KeyPair{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestTreeFunc_hasData(t *testing.T) {
	d, err := dep.ParseStoreKeyPrefix("existing")
	if err != nil {
		t.Fatal(err)
	}

	data := []*dep.KeyPair{
		&dep.KeyPair{Key: "", Value: ""},
		&dep.KeyPair{Key: "user/sethvargo", Value: "true"},
		&dep.KeyPair{Key: "maxconns", Value: "11"},
		&dep.KeyPair{Key: "minconns", Value: "2"},
	}

	brain := NewBrain()
	brain.Remember(d, data)

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := treeFunc(brain, used, missing)
	result, err := f("existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.KeyPair{
		&dep.KeyPair{Key: "user/sethvargo", Value: "true"},
		&dep.KeyPair{Key: "maxconns", Value: "11"},
		&dep.KeyPair{Key: "minconns", Value: "2"},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if len(missing) != 0 {
		t.Errorf("expected missing to have 0 elements, but had %d", len(missing))
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}
}

func TestTreeFunc_missingData(t *testing.T) {
	d, err := dep.ParseStoreKeyPrefix("non-existing")
	if err != nil {
		t.Fatal(err)
	}

	brain := NewBrain()

	used := make(map[string]dep.Dependency)
	missing := make(map[string]dep.Dependency)

	f := treeFunc(brain, used, missing)
	result, err := f("non-existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.KeyPair{}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}

	if _, ok := used[d.HashCode()]; !ok {
		t.Errorf("expected dep to be used")
	}

	if _, ok := missing[d.HashCode()]; !ok {
		t.Errorf("expected dep to be missing")
	}
}

func TestByTag_emptyList(t *testing.T) {
	result, err := byTag(nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string][]*dep.HealthService{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestByTag_GroupsList(t *testing.T) {
	result, err := byTag([]*dep.HealthService{
		&dep.HealthService{Name: "web3", Tags: []string{"metric"}},
		&dep.HealthService{Name: "web2", Tags: []string{"search"}},
		&dep.HealthService{Name: "web1", Tags: []string{"auth", "search"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string][]*dep.HealthService{
		"auth": []*dep.HealthService{
			&dep.HealthService{Name: "web1", Tags: []string{"auth", "search"}},
		},
		"metric": []*dep.HealthService{
			&dep.HealthService{Name: "web3", Tags: []string{"metric"}},
		},
		"search": []*dep.HealthService{
			&dep.HealthService{Name: "web2", Tags: []string{"search"}},
			&dep.HealthService{Name: "web1", Tags: []string{"auth", "search"}},
		},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestByKey_emptyList(t *testing.T) {
	result, err := byKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string][]*dep.KeyPair{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v to be %#v", result, expected)
	}
}

func TestByKey_topLevel(t *testing.T) {
	result, err := byKey([]*dep.KeyPair{
		&dep.KeyPair{Key: "elasticsearch/a", Value: "1"},
		&dep.KeyPair{Key: "elasticsearch/b", Value: "2"},
		&dep.KeyPair{Key: "redis/a/b", Value: "3"},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string][]*dep.KeyPair{
		"elasticsearch": []*dep.KeyPair{
			&dep.KeyPair{Key: "a", Value: "1"},
			&dep.KeyPair{Key: "b", Value: "2"},
		},
		"redis": []*dep.KeyPair{
			&dep.KeyPair{Key: "a/b", Value: "3"},
		},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v to be %#v", result, expected)
	}
}

func TestEnv(t *testing.T) {
	if err := os.Setenv("foo", "bar"); err != nil {
		t.Fatal(err)
	}

	result, err := env("foo")
	if err != nil {
		t.Fatal(err)
	}

	if result != "bar" {
		t.Errorf("expected %#v to be %#v", result, "bar")
	}
}

func TestParseJSON(t *testing.T) {
	result, err := parseJSON(`{"foo": "bar"}`)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]interface{}{"foo": "bar"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v to be %#v", result, expected)
	}
}

func TestParseJSON_empty(t *testing.T) {
	result, err := parseJSON("")
	if err != nil {
		t.Fatal(err)
	}

	expected := make([]interface{}, 0)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v to be %#v", result, expected)
	}
}

func TestToLower(t *testing.T) {
	result, err := toLower("FOO")
	if err != nil {
		t.Fatal(err)
	}

	expected := "foo"
	if result != expected {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestToTitle(t *testing.T) {
	result, err := toTitle("foo")
	if err != nil {
		t.Fatal(err)
	}

	expected := "Foo"
	if result != expected {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestToUpper(t *testing.T) {
	result, err := toUpper("foo")
	if err != nil {
		t.Fatal(err)
	}

	expected := "FOO"
	if result != expected {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestReplaceAll(t *testing.T) {
	result, err := replaceAll("bar", "foo", "foobarzipbar")
	if err != nil {
		t.Fatal(err)
	}

	expected := "foofoozipfoo"
	if result != expected {
		t.Errorf("expected %q to be %q", result, expected)
	}
}

func TestRegexReplaceAll(t *testing.T) {
	result, err := regexReplaceAll(`[a-z]`, "x", "foobarzipbar")
	if err != nil {
		t.Fatal(err)
	}

	expected := "xxxxxxxxxxxx"
	if result != expected {
		t.Errorf("expected %q to be %q", result, expected)
	}
}
