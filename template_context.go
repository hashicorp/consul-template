package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/consul-template/util"
)

// TemplateContext is what Template uses to determine the values that are
// available for template parsing.
type TemplateContext struct {
	catalogServices map[string][]*util.CatalogService
	files           map[string]string
	keys            map[string]string
	keyPrefixes     map[string][]*util.KeyPair
	nodes           map[string][]*util.Node
	services        map[string][]*util.HealthService
}

// NewTemplateContext creates a new TemplateContext with empty values for each
// of the key structs.
func NewTemplateContext() (*TemplateContext, error) {
	return &TemplateContext{
		catalogServices: make(map[string][]*util.CatalogService),
		files:           make(map[string]string),
		keys:            make(map[string]string),
		keyPrefixes:     make(map[string][]*util.KeyPair),
		nodes:           make(map[string][]*util.Node),
		services:        make(map[string][]*util.HealthService),
	}, nil
}

// decodeJSON returns a structure for valid JSON
func (c *TemplateContext) decodeJSON(s string) (interface{}, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return nil, err
	}

	return data, nil
}

// returns the value of the environment variable set
func (c *TemplateContext) env(s string) (string, error) {
	return os.Getenv(s), nil
}

// groupByTag is a template func that takes the provided services and
// produces a map based on Service tags.
//
// The map key is a string representing the service tag. The map value is a
// slice of Services which have the tag assigned.
func (c *TemplateContext) groupByTag(in []*util.HealthService) map[string][]*util.HealthService {
	m := make(map[string][]*util.HealthService)
	for _, s := range in {
		for _, t := range s.Tags {
			m[t] = append(m[t], s)
		}
	}
	return m
}

// toLower converts the given string (usually by a pipe) to lowercase.
func (c *TemplateContext) toLower(s string) (string, error) {
	return strings.ToLower(s), nil
}

// toTitle converts the given string (usually by a pipe) to titlecase.
func (c *TemplateContext) toTitle(s string) (string, error) {
	return strings.Title(s), nil
}

// toUpper converts the given string (usually by a pipe) to uppercase.
func (c *TemplateContext) toUpper(s string) (string, error) {
	return strings.ToUpper(s), nil
}

// replaceAll replaces all occurrences of a value in a string with the given
// replacement value.
func (c *TemplateContext) replaceAll(f, t, s string) (string, error) {
	return strings.Replace(s, f, t, -1), nil
}

// regexReplaceAll replaces all occurrences of a regular expression with
// the given replacement value.
func (c *TemplateContext) regexReplaceAll(re, pl, s string) (string, error) {
	compiled, err := regexp.Compile(re)
	if err == nil {
		return compiled.ReplaceAllString(s, pl), nil
	}

	return "", err
}
