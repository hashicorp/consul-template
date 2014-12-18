package util

import (
	"reflect"
	"sort"
	"strings"
	"testing"

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

func TestServiceList_sorts(t *testing.T) {
	a := ServiceList{
		&Service{Node: "frontend01", ID: "1"},
		&Service{Node: "frontend01", ID: "2"},
		&Service{Node: "frontend02", ID: "1"},
	}
	b := ServiceList{
		&Service{Node: "frontend02", ID: "1"},
		&Service{Node: "frontend01", ID: "2"},
		&Service{Node: "frontend01", ID: "1"},
	}
	c := ServiceList{
		&Service{Node: "frontend01", ID: "2"},
		&Service{Node: "frontend01", ID: "1"},
		&Service{Node: "frontend02", ID: "1"},
	}

	sort.Stable(a)
	sort.Stable(b)
	sort.Stable(c)

	expected := ServiceList{
		&Service{Node: "frontend01", ID: "1"},
		&Service{Node: "frontend01", ID: "2"},
		&Service{Node: "frontend02", ID: "1"},
	}

	if !reflect.DeepEqual(a, expected) {
		t.Fatal("invalid sort")
	}

	if !reflect.DeepEqual(b, expected) {
		t.Fatal("invalid sort")
	}

	if !reflect.DeepEqual(c, expected) {
		t.Fatal("invalid sort")
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

func TestParseServiceDependency_nameAndAnyStatus(t *testing.T) {
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

func TestParseServiceDependency_nameAndPassingStatus(t *testing.T) {
	sd, err := ParseServiceDependency("webapp", "passing")
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

func TestParseServiceDependency_nameAndMultipleStatuses(t *testing.T) {
	sd, err := ParseServiceDependency("webapp", "passing,warning,critical")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "webapp [passing,warning,critical]",
		Name:   "webapp",
		Status: ServiceStatusFilter{HealthPassing, HealthWarning, HealthCritical},
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseServiceDependency_nameAndMultipleStatusesIncludingAny(t *testing.T) {
	sd, err := ParseServiceDependency("webapp", "passing,any,warning")
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

func TestParseServiceDependency_nameAndInvalidStatus(t *testing.T) {
	_, err := ParseServiceDependency("webapp", "passing,invalid")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "expected some of \"any, passing, warning, critical\" as health status"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseServiceDependency_nameAndMultipleStatusesWithWeirdWhitespace(t *testing.T) {
	sd, err := ParseServiceDependency("webapp", "  passing,\nwarning  ,critical     \t ")
	if err != nil {
		t.Fatal(err)
	}

	expected := &ServiceDependency{
		rawKey: "webapp [passing,warning,critical]",
		Name:   "webapp",
		Status: ServiceStatusFilter{HealthPassing, HealthWarning, HealthCritical},
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
