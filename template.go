package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/hashicorp/consul-template/util"
)

// Template is the Go representation of a Consul Template template on disk.
type Template struct {
	// The path to this Template on disk.
	Path string

	// The internal list of dependencies for this Template.
	dependencies []util.Dependency
}

// NewTemplate creates and parses a new Consul Template template at the given
// path. If the template does not exist, an error is returned. During
// initialization, the template is read and is parsed for dependencies. Any
// errors that occur are returned.
func NewTemplate(path string) (*Template, error) {
	template := &Template{Path: path}
	if err := template.init(); err != nil {
		return nil, err
	}

	return template, nil
}

// HashCode returns the map value for this Template
func (t *Template) HashCode() string {
	return fmt.Sprintf("Template|%s", t.Path)
}

// Dependencies returns the dependencies that this template has.
func (t *Template) Dependencies() []util.Dependency {
	return t.dependencies
}

// Execute takes the given template context and processes the template.
//
// If the TemplateContext is nil, an error will be returned.
//
// If the TemplateContext does not have all required Dependencies, an error will
// be returned.
func (t *Template) Execute(c *TemplateContext) ([]byte, error) {
	if c == nil {
		return nil, errors.New("templateContext must be given")
	}

	// Make sure the context contains everything we need
	if err := t.validateDependencies(c); err != nil {
		return nil, err
	}

	// Render the template
	contents, err := ioutil.ReadFile(t.Path)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		// API functions
		"file": func(s string) string {
			return c.File[s]
		},
		"keyPrefix": func(s string) []*util.KeyPair {
			log.Printf("[WARN] DEPRECATED: Please use `ls` or `tree` instead of `keyPrefix`")
			return c.KeyPrefixes[s]
		},
		"key": func(s string) string {
			return c.Keys[s]
		},
		"ls": func(s string) []*util.KeyPair {
			var result [](*util.KeyPair)
			// Only return non-empty top-level keys
			for _, pair := range c.KeyPrefixes[s] {
				if pair.Key != "" && !strings.Contains(pair.Key, "/") {
					result = append(result, pair)
				}
			}
			return result
		},
		"nodes": func(s ...string) ([]*util.Node, error) {
			d, err := util.ParseNodesDependency(s...)
			if err != nil {
				return nil, err
			}
			return c.Nodes[d.Key()], nil
		},
		"service": func(s ...string) ([]*util.Service, error) {
			d, err := util.ParseServiceDependency(s...)
			if err != nil {
				return nil, err
			}
			return c.Services[d.Key()], nil
		},
		"services": func(s ...string) ([]*util.CatalogService, error) {
			d, err := util.ParseCatalogServicesDependency(s...)
			if err != nil {
				return nil, err
			}
			return c.CatalogServices[d.Key()], nil
		},
		"tree": func(s string) []*util.KeyPair {
			var result []*util.KeyPair
			// Filter empty keys (folder nodes)
			for _, pair := range c.KeyPrefixes[s] {
				parts := strings.Split(pair.Key, "/")
				if parts[len(parts)-1] != "" {
					result = append(result, pair)
				}
			}
			return result
		},

		// Helper functions
		"byTag":           c.groupByTag,
		"env":             c.env,
		"parseJSON":       c.decodeJSON,
		"regexReplaceAll": c.regexReplaceAll,
		"replaceAll":      c.replaceAll,
		"toLower":         c.toLower,
		"toTitle":         c.toTitle,
		"toUpper":         c.toUpper,
	}).Parse(string(contents))

	if err != nil {
		return nil, err
	}

	var buff = new(bytes.Buffer)
	err = tmpl.Execute(buff, c)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// init reads the template file and parses all the required dependencies into a
// dependencies slice which is then added onto the Template.
func (t *Template) init() error {
	contents, err := ioutil.ReadFile(t.Path)
	if err != nil {
		return err
	}

	deps := make(map[string]util.Dependency)

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		// API functions
		"file":      util.FileDependencyFunc(deps),
		"key":       util.KeyFunc(deps),
		"keyPrefix": util.KeyPrefixFunc(deps),
		"ls":        util.KeyPrefixFunc(deps),
		"nodes":     util.NodesFunc(deps),
		"service":   util.ServiceFunc(deps),
		"services":  util.CatalogServicesFunc(deps),
		"tree":      util.KeyPrefixFunc(deps),

		// Helper functions
		"byTag":           t.noop,
		"env":             t.noop,
		"parseJSON":       t.noop,
		"regexReplaceAll": t.noop,
		"replaceAll":      t.noop,
		"toLower":         t.noop,
		"toTitle":         t.noop,
		"toUpper":         t.noop,
	}).Parse(string(contents))

	if err != nil {
		return err
	}

	err = tmpl.Execute(ioutil.Discard, nil)
	if err != nil {
		return err
	}

	dependencies := make([]util.Dependency, 0, len(deps))
	for _, dep := range deps {
		dependencies = append(dependencies, dep)
	}
	deps = nil

	t.dependencies = dependencies

	return nil
}

// Validates that all required dependencies in t are defined in c.
func (t *Template) validateDependencies(c *TemplateContext) error {
	for _, dep := range t.Dependencies() {
		if !dep.InContext(c) {
			return fmt.Errorf("template context missing %s", dep.Display())
		}
	}

	return nil
}

// noop is a special function that returns itself. This is used during the
// dependency accumulation to allow the template to be processed once.
func (t *Template) noop(thing ...interface{}) (interface{}, error) {
	return thing[len(thing)-1], nil
}

// TemplateContext is what Template uses to determine the values that are
// available for template parsing.
type TemplateContext struct {
	CatalogServices map[string][]*util.CatalogService
	Files           map[string]string
	Keys            map[string]string
	KeyPrefixes     map[string][]*util.KeyPair
	Nodes           map[string][]*util.Node
	Services        map[string][]*util.Service
}

// NewTemplateContext creates a new TemplateContext with empty values for each
// of the key structs.
func NewTemplateContext() (*TemplateContext, error) {
	return &TemplateContext{
		CatalogServices: make(map[string][]*util.CatalogService),
		Files:           make(map[string]string),
		Keys:            make(map[string]string),
		KeyPrefixes:     make(map[string][]*util.KeyPair),
		Nodes:           make(map[string][]*util.Node),
		Services:        make(map[string][]*util.Service),
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
func (c *TemplateContext) groupByTag(in []*util.Service) map[string][]*util.Service {
	m := make(map[string][]*util.Service)
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
