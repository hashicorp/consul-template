package template

import (
	"fmt"

	dep "github.com/hashicorp/consul-template/dependency"
)

func isTagExists(
	serviceTags dep.ServiceTags,
	filterTag string,
) bool {
	for _, serviceTag := range serviceTags {
		if filterTag == serviceTag {
			return true
		}
	}

	return false
}

func isTagsExists(
	serviceTags dep.ServiceTags,
	requiredTags []string,
) bool {
	for _, requiredTag := range requiredTags {
		if !isTagExists(serviceTags, requiredTag) {
			return false
		}
	}

	return true
}

func filterServicesByTag(
	services []interface{},
	tag string,
) ([]interface{}, error) {
	var m []interface{}

	for _, service := range services {
		switch s := service.(type) {
		case nil:
		case *dep.CatalogSnippet:
			if isTagExists(s.Tags, tag) {
				m = append(m, s)
			}

		case *dep.CatalogService:
			if isTagExists(s.ServiceTags, tag) {
				m = append(m, s)
			}

		case *dep.HealthService:
			if isTagExists(s.Tags, tag) {
				m = append(m, s)
			}

		default:
			return nil, fmt.Errorf(
				"withTag: wrong argument type %T",
				service,
			)

		}
	}

	return m, nil
}
