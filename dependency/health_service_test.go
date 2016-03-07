package dependency

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
)

func TestServiceDependencyFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseHealthServices("consul")
	if err != nil {
		t.Fatal(err)
	}

	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := results.([]*HealthService)
	if !ok {
		t.Fatal("could not convert result to []*HealthService")
	}

	if typed[0].ID != "consul" {
		t.Errorf("expected %q to be %q", typed[0].ID, "consul")
	}

	if len(typed[0].Checks) == 0 {
		t.Fatalf("expected to have at least one associated health check")
	}

	if typed[0].Checks[0].CheckID != "serfHealth" {
		t.Errorf("expected %q to be %q", typed[0].Checks[0].CheckID, "serfHealth")
	}
}

func TestServiceDependencyFetch_stopped(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseHealthServices("consul")
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error)
	go func() {
		results, _, err := dep.Fetch(clients, &QueryOptions{WaitIndex: 100})
		if results != nil {
			t.Fatalf("should not get results: %#v", results)
		}
		errCh <- err
	}()

	dep.Stop()

	select {
	case err := <-errCh:
		if err != ErrStopped {
			t.Errorf("expected %q to be %q", err, ErrStopped)
		}
	case <-time.After(50 * time.Millisecond):
		t.Errorf("did not return in 50ms")
	}
}

func TestHealthServiceList_sorts(t *testing.T) {
	a := HealthServiceList{
		&HealthService{Node: "frontend01", ID: "1"},
		&HealthService{Node: "frontend01", ID: "2"},
		&HealthService{Node: "frontend02", ID: "1"},
	}
	b := HealthServiceList{
		&HealthService{Node: "frontend02", ID: "1"},
		&HealthService{Node: "frontend01", ID: "2"},
		&HealthService{Node: "frontend01", ID: "1"},
	}
	c := HealthServiceList{
		&HealthService{Node: "frontend01", ID: "2"},
		&HealthService{Node: "frontend01", ID: "1"},
		&HealthService{Node: "frontend02", ID: "1"},
	}

	sort.Stable(a)
	sort.Stable(b)
	sort.Stable(c)

	expected := HealthServiceList{
		&HealthService{Node: "frontend01", ID: "1"},
		&HealthService{Node: "frontend01", ID: "2"},
		&HealthService{Node: "frontend02", ID: "1"},
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

func TestServiceDependencyHashCode_isUnique(t *testing.T) {
	dep1, err := ParseHealthServices("redis@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	dep2, err := ParseHealthServices("redis@nyc2")
	if err != nil {
		t.Fatal(err)
	}

	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseHealthServices_emptyString(t *testing.T) {
	_, err := ParseHealthServices("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "cannot specify empty health service dependency"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseHealthServices_name(t *testing.T) {
	sd, err := ParseHealthServices("webapp")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "webapp" {
		t.Errorf("expected %q to be %q", sd.rawKey, "webapp")
	}

	if sd.Name != "webapp" {
		t.Errorf("expected %q to be %q", sd.Name, "webapp")
	}

	if sd.StatusFilter != nil {
		t.Errorf("expected %q to be nil", sd.StatusFilter)
	}
}

func TestParseHealthServices_nameAndAnyStatus(t *testing.T) {
	sd, err := ParseHealthServices("webapp", "passing")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "webapp [passing]" {
		t.Errorf("expected %q to be %q", sd.rawKey, "webapp")
	}

	if sd.Name != "webapp" {
		t.Errorf("expected %q to be %q", sd.Name, "webapp")
	}

	filter := ServiceStatusFilter{HealthPassing}
	if !reflect.DeepEqual(sd.StatusFilter, filter) {
		t.Errorf("expected %q to be %q", sd.StatusFilter, filter)
	}
}

func TestParseHealthServices_slashName(t *testing.T) {
	sd, err := ParseHealthServices("web/app")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "web/app" {
		t.Errorf("expected %q to be %q", sd.rawKey, "web/app")
	}

	if sd.Name != "web/app" {
		t.Errorf("expected %q to be %q", sd.Name, "web/app")
	}

	if sd.StatusFilter != nil {
		t.Errorf("expected %q to be nil", sd.StatusFilter)
	}
}

func TestParseHealthServices_underscoreName(t *testing.T) {
	sd, err := ParseHealthServices("web_app")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "web_app" {
		t.Errorf("expected %q to be %q", sd.rawKey, "web_app")
	}

	if sd.Name != "web_app" {
		t.Errorf("expected %q to be %q", sd.Name, "web_app")
	}

	if sd.StatusFilter != nil {
		t.Errorf("expected %q to be nil", sd.StatusFilter)
	}
}

func TestParseHealthServices_dotTag(t *testing.T) {
	sd, err := ParseHealthServices("first.release.webapp")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "first.release.webapp" {
		t.Errorf("expected %q to be %q", sd.rawKey, "first.release.webapp")
	}

	if sd.Name != "webapp" {
		t.Errorf("expected %q to be %q", sd.Name, "webapp")
	}

	if sd.Tag != "first.release" {
		t.Errorf("expected %q to be %q", sd.Tag, "first.release")
	}

	if sd.StatusFilter != nil {
		t.Errorf("expected %q to be nil", sd.StatusFilter)
	}
}

func TestParseHealthServices_nameTag(t *testing.T) {
	sd, err := ParseHealthServices("release.webapp")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "release.webapp" {
		t.Errorf("expected %q to be %q", sd.rawKey, "release.webapp")
	}

	if sd.Name != "webapp" {
		t.Errorf("expected %q to be %q", sd.Name, "webapp")
	}

	if sd.Tag != "release" {
		t.Errorf("expected %q to be %q", sd.Tag, "release")
	}

	if sd.StatusFilter != nil {
		t.Errorf("expected %q to be nil", sd.StatusFilter)
	}
}

func TestParseHealthServices_nameTagDataCenter(t *testing.T) {
	sd, err := ParseHealthServices("release.webapp@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "release.webapp@nyc1" {
		t.Errorf("expected %q to be %q", sd.rawKey, "release.webapp@nyc1")
	}

	if sd.Name != "webapp" {
		t.Errorf("expected %q to be %q", sd.Name, "webapp")
	}

	if sd.Tag != "release" {
		t.Errorf("expected %q to be %q", sd.Tag, "release")
	}

	if sd.DataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", sd.DataCenter, "nyc1")
	}

	if sd.StatusFilter != nil {
		t.Errorf("expected %q to be nil", sd.StatusFilter)
	}
}

func TestParseHealthServices_dataCenterOnly(t *testing.T) {
	_, err := ParseHealthServices("@nyc1")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "invalid health service dependency format"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseHealthServices_nameAndDataCenter(t *testing.T) {
	sd, err := ParseHealthServices("webapp@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "webapp@nyc1" {
		t.Errorf("expected %q to be %q", sd.rawKey, "webapp@nyc1")
	}

	if sd.Name != "webapp" {
		t.Errorf("expected %q to be %q", sd.Name, "webapp")
	}

	if sd.Tag != "" {
		t.Errorf("expected %q to be %q", sd.Tag, "")
	}

	if sd.DataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", sd.DataCenter, "nyc1")
	}

	if sd.StatusFilter != nil {
		t.Errorf("expected %q to be nil", sd.StatusFilter)
	}
}

func TestServiceTagsContains(t *testing.T) {
	s := &HealthService{
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

func TestStatusFromChecks_nil(t *testing.T) {
	status, err := statusFromChecks(nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := "passing"
	if status != expected {
		t.Fatalf("expected %q to be %q", status, expected)
	}
}

func TestStatusFromChecks_badCbeck(t *testing.T) {
	checks := []*api.HealthCheck{
		&api.HealthCheck{Status: "bacon"},
	}
	_, err := statusFromChecks(checks)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := fmt.Sprintf("unknown status: %q", "bacon")
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to include %q", err.Error(), expected)
	}
}

func TestStatusFromChecks_passing(t *testing.T) {
	checks := []*api.HealthCheck{
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "passing"},
	}
	status, err := statusFromChecks(checks)
	if err != nil {
		t.Fatal(err)
	}

	expected := "passing"
	if status != expected {
		t.Fatalf("expected %q to be %q", status, expected)
	}
}

func TestStatusFromChecks_warning(t *testing.T) {
	checks := []*api.HealthCheck{
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "warning"},
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "passing"},
	}
	status, err := statusFromChecks(checks)
	if err != nil {
		t.Fatal(err)
	}

	expected := "warning"
	if status != expected {
		t.Fatalf("expected %q to be %q", status, expected)
	}
}

func TestStatusFromChecks_unknown(t *testing.T) {
	checks := []*api.HealthCheck{
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "warning"},
		&api.HealthCheck{Status: "unknown"},
		&api.HealthCheck{Status: "passing"},
	}
	status, err := statusFromChecks(checks)
	if err != nil {
		t.Fatal(err)
	}

	expected := "unknown"
	if status != expected {
		t.Fatalf("expected %q to be %q", status, expected)
	}
}

func TestStatusFromChecks_critical(t *testing.T) {
	checks := []*api.HealthCheck{
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "warning"},
		&api.HealthCheck{Status: "unknown"},
		&api.HealthCheck{Status: "critical"},
	}
	status, err := statusFromChecks(checks)
	if err != nil {
		t.Fatal(err)
	}

	expected := "critical"
	if status != expected {
		t.Fatalf("expected %q to be %q", status, expected)
	}
}

func TestStatusFromChecks_nodeMaintenance(t *testing.T) {
	checks := []*api.HealthCheck{
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "warning"},
		&api.HealthCheck{Status: "unknown"},
		&api.HealthCheck{Status: "critical"},
		&api.HealthCheck{CheckID: "_node_maintenance"},
	}
	status, err := statusFromChecks(checks)
	if err != nil {
		t.Fatal(err)
	}

	expected := "maintenance"
	if status != expected {
		t.Fatalf("expected %q to be %q", status, expected)
	}
}

func TestStatusFromChecks_serviceMaintenance(t *testing.T) {
	checks := []*api.HealthCheck{
		&api.HealthCheck{Status: "passing"},
		&api.HealthCheck{Status: "warning"},
		&api.HealthCheck{Status: "unknown"},
		&api.HealthCheck{Status: "critical"},
		&api.HealthCheck{CheckID: "_service_maintenance:1234"},
	}
	status, err := statusFromChecks(checks)
	if err != nil {
		t.Fatal(err)
	}

	expected := "maintenance"
	if status != expected {
		t.Fatalf("expected %q to be %q", status, expected)
	}
}

// Tests specifically relating to health service filtering
// -------------------------

func TestNewServiceStatusFilter_emptyString(t *testing.T) {
	filter, err := NewServiceStatusFilter("")
	if err != nil {
		t.Fatal(err)
	}

	var expected ServiceStatusFilter
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %#v to be %#v", filter, expected)
	}
}

func TestNewServiceStatusFilter_statuses(t *testing.T) {
	statuses := []string{
		HealthAny,
		HealthUnknown,
		HealthPassing,
		HealthWarning,
		HealthCritical,
	}
	for _, status := range statuses {
		filter, err := NewServiceStatusFilter(status)
		if err != nil {
			t.Fatal(err)
		}

		expected := ServiceStatusFilter{status}
		if !reflect.DeepEqual(filter, expected) {
			t.Errorf("expected %#v to be %#v", filter, expected)
		}
	}
}

func TestNewServiceStatusFilter_any(t *testing.T) {
	filter, err := NewServiceStatusFilter("any")
	if err != nil {
		t.Fatal(err)
	}

	expected := ServiceStatusFilter{HealthAny}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %#v to be %#v", filter, expected)
	}
}

func TestNewServiceStatusFilter_anyWithMore(t *testing.T) {
	_, err := NewServiceStatusFilter("any, passing")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := fmt.Sprintf("cannot specify extra keys when using %q", "any")
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestNewServiceStatusFilter_statusWithSpaces(t *testing.T) {
	filter, err := NewServiceStatusFilter("passing,     critical, , unknown")
	if err != nil {
		t.Fatal(err)
	}

	expected := ServiceStatusFilter{HealthPassing, HealthCritical, HealthUnknown}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %#v to be %#v", filter, expected)
	}
}

func TestNewServiceStatusFilter_invalidStatus(t *testing.T) {
	_, err := NewServiceStatusFilter("not_a_valid_status")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := fmt.Sprintf("invalid filter %q", "not_a_valid_status")
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestAccept_any(t *testing.T) {
	filter, err := NewServiceStatusFilter("any")
	if err != nil {
		t.Fatal(err)
	}

	statuses := []string{"passing", "warning", "critical"}
	for _, status := range statuses {
		if !filter.Accept(status) {
			t.Errorf("expected filter to accept %q", status)
		}
	}
}

func TestAccept_multiple(t *testing.T) {
	filter, err := NewServiceStatusFilter("passing, critical")
	if err != nil {
		t.Fatal(err)
	}

	statuses := []string{"passing", "critical"}
	for _, status := range statuses {
		if !filter.Accept(status) {
			t.Errorf("expected filter to accept %q", status)
		}
	}

	if filter.Accept("warning") {
		t.Fatalf("expected filter to not accept %q", "warning")
	}
}
