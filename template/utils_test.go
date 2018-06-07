package template

import (
	"testing"

	dep "github.com/hashicorp/consul-template/dependency"
)

func TestIsTagExists(t *testing.T) {
	availableTags := dep.ServiceTags{"tagone", "tagtwo", "tagthree"}
	requiredTag := "required"

	if isTagExists(availableTags, requiredTag) {
		t.Errorf(
			"required tag `%s` exists in %s",
			requiredTag,
			availableTags,
		)
	}

	availableTags = dep.ServiceTags{"required", "tagone", "tagtwo"}
	if !isTagExists(availableTags, requiredTag) {
		t.Errorf(
			"required tag `%s` not exists in %v",
			requiredTag,
			availableTags,
		)
	}
}

func TestIsTagsExists(t *testing.T) {
	availableTags := dep.ServiceTags{"tagone", "tagtwo", "tagthree"}
	requiredTags := []string{"required", "uniquetag"}

	if isTagsExists(availableTags, requiredTags) {
		t.Errorf(
			"required tags `%v` exists in `%v`",
			requiredTags,
			availableTags,
		)
	}

	availableTags = dep.ServiceTags{"required", "uniquetag", "tagtwo"}
	if !isTagsExists(availableTags, requiredTags) {
		t.Errorf(
			"required tags`%s` not exists in %v",
			requiredTags,
			availableTags,
		)
	}
}

func TestFilterServicesByTag(t *testing.T) {
	services := []interface{}{
		&dep.HealthService{
			Node:    "one",
			Address: "1.1.1.1",
			Tags:    dep.ServiceTags{"one", "two"},
		},
		&dep.HealthService{
			Node:    "two",
			Address: "2.2.2.2",
			Tags:    dep.ServiceTags{"three", "four"},
		},
		&dep.HealthService{
			Node:    "three",
			Address: "3.3.3.3",
			Tags:    dep.ServiceTags{"five", "six", "one"},
		},
		&dep.CatalogService{
			Node:        "four",
			Address:     "4.4.4.4",
			ServiceTags: dep.ServiceTags{"seven", "eight", "six"},
		},
		&dep.CatalogService{
			Node:        "five",
			Address:     "5.5.5.5",
			ServiceTags: dep.ServiceTags{"nine", "ten"},
		},
		&dep.CatalogSnippet{
			Name: "six",
			Tags: dep.ServiceTags{"eleven", "twelve", "six", "one"},
		},
	}

	requiredTag := "six"
	successServiceCount := 3

	filteredServices, err := filterServicesByTag(services, requiredTag)
	if err != nil {
		t.Errorf(
			"got error %s", err.Error(),
		)
	}

	if len(filteredServices) != successServiceCount {
		t.Errorf(
			"unexpected count of filtered services, got %d, expected %d",
			len(filteredServices),
			successServiceCount,
		)
	}

	for _, service := range filteredServices {
		switch s := service.(type) {
		case *dep.CatalogService:
			if s.Address != "4.4.4.4" && s.Node != "four" {
				t.Errorf(
					"wrong filtered catalog service, got: node %s and "+
						"address %s expected: node `four` and "+
						"address `4.4.4.4`",
					s.Node, s.Address,
				)
			}
		case *dep.HealthService:
			if s.Address != "3.3.3.3" && s.Node != "three" {
				t.Errorf(
					"wrong filtered health service, got: node %s and "+
						"address %s expected: node `three` and "+
						"address `3.3.3.3`",
					s.Node, s.Address,
				)
			}
		case *dep.CatalogSnippet:
			if s.Name != "six" {
				t.Errorf(
					"wrong filtered catalog snippet, got: name %s "+
						"expected: name `six`",
					s.Name,
				)
			}
		default:
			t.Errorf("unexpected service type %T", s)
		}
	}
}
