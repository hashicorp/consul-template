// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/consul/api"
)

type Tenancy struct {
	Partition string
	Namespace string
}

type TenancyHelper struct {
	once               sync.Once
	isConsulEnterprise bool
	consulClient       *api.Client
}

func NewTenancyHelper(consulClient *api.Client) (*TenancyHelper, error) {
	t := &TenancyHelper{
		consulClient: consulClient,
	}

	err := t.init()
	if err != nil {
		return nil, err
	}

	return t, nil
}

// TestTenancies returns a list of tenancies which represent
// the namespace and partition combinations that can be used in unit tests
func (t *TenancyHelper) TestTenancies() []*Tenancy {
	tenancies := []*Tenancy{
		t.Tenancy("default.default"),
	}

	if t.isConsulEnterprise {
		tenancies = append(tenancies, t.Tenancy("default.bar"), t.Tenancy("foo.default"), t.Tenancy("foo.bar"))
	}

	return tenancies
}

// Tenancy constructs a Tenancy from a concise string representation
// suitable for use in unit tests.
//
// - ""        : partition=""    namespace=""
// - "foo"     : partition="foo" namespace=""
// - "foo.bar" : partition="foo" namespace="bar"
// - <others>  : partition="BAD" namespace="BAD"
func (t *TenancyHelper) Tenancy(s string) *Tenancy {
	parts := strings.Split(s, ".")
	switch len(parts) {
	case 0:
		return &Tenancy{}
	case 1:
		return &Tenancy{
			Partition: parts[0],
		}
	case 2:
		return &Tenancy{
			Partition: parts[0],
			Namespace: parts[1],
		}
	default:
		return &Tenancy{Partition: "BAD", Namespace: "BAD"}
	}
}

func (t *TenancyHelper) init() error {
	var versionErr error

	t.once.Do(func() {
		v, err := t.consulClient.Agent().Version()
		if err != nil {
			versionErr = err
			return
		}

		if version, ok := v["HumanVersion"].(string); ok {
			// if type is string & the key is present
			// then check whether the version contains "ent"
			// example: "1.8.0+ent" is enterprise
			// otherwise it is CE
			t.isConsulEnterprise = strings.Contains(version, "ent")
		}
	})

	return versionErr
}

func (t *TenancyHelper) AppendTenancyInfo(name string, tenancy *Tenancy) string {
	return fmt.Sprintf("%s_%s_Namespace_%s_Partition", name, tenancy.Namespace, tenancy.Partition)
}

func (t *TenancyHelper) RunWithTenancies(testFunc func(tenancy *Tenancy), test *testing.T, testName string) {
	for _, tenancy := range t.TestTenancies() {
		test.Run(t.AppendTenancyInfo(testName, tenancy), func(t *testing.T) {
			testFunc(tenancy)
		})
	}
}

func (t *TenancyHelper) GenerateTenancyTests(generationFunc func(tenancy *Tenancy) []interface{}) []interface{} {
	cases := make([]interface{}, 0)
	for _, tenancy := range t.TestTenancies() {
		cases = append(cases, generationFunc(tenancy)...)
	}
	return cases
}

func (t *TenancyHelper) GenerateNonDefaultTenancyTests(generationFunc func(tenancy *Tenancy) []interface{}) []interface{} {
	cases := make([]interface{}, 0)
	for _, tenancy := range t.TestTenancies() {
		if tenancy.Partition != "default" || tenancy.Namespace != "default" {
			cases = append(cases, generationFunc(tenancy)...)
		}
	}
	return cases
}

func (t *TenancyHelper) GenerateDefaultTenancyTests(generationFunc func(tenancy *Tenancy) []interface{}) []interface{} {
	cases := make([]interface{}, 0)
	for _, tenancy := range t.TestTenancies() {
		if tenancy.Partition == "default" && tenancy.Namespace == "default" {
			cases = append(cases, generationFunc(tenancy)...)
		}
	}
	return cases
}

func (t *TenancyHelper) GetUniquePartitions() map[*api.Partition]interface{} {
	partitions := make(map[*api.Partition]interface{})
	for _, tenancy := range t.TestTenancies() {
		partitions[&api.Partition{
			Name: tenancy.Partition,
		}] = nil
	}
	return partitions
}
