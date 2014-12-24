package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"text/template"

	"github.com/hashicorp/consul-template/util"
)

// Template is the Go representation of a Consul Template template on disk.
type Template struct {
	// The path to this Template on disk.
	Path string

	// The internal list of dependencies for this Template.
	dependencies []dependencyContextBridge
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
func (t *Template) Dependencies() []dependencyContextBridge {
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
			return c.files[s]
		},
		"keyPrefix": func(s string) []*util.KeyPair {
			log.Printf("[WARN] DEPRECATED: Please use `ls` or `tree` instead of `keyPrefix`")
			return c.keyPrefixes[s]
		},
		"key": func(s string) string {
			return c.keys[s]
		},
		"ls": func(s string) []*util.KeyPair {
			var result [](*util.KeyPair)
			// Only return non-empty top-level keys
			for _, pair := range c.keyPrefixes[s] {
				if pair.Key != "" && !strings.Contains(pair.Key, "/") {
					result = append(result, pair)
				}
			}
			return result
		},
		"nodes": func(s ...string) ([]*util.Node, error) {
			d, err := util.ParseCatalogNodes(s...)
			if err != nil {
				return nil, err
			}
			return c.nodes[d.Key()], nil
		},
		"service": func(s ...string) ([]*util.HealthService, error) {
			d, err := util.ParseHealthServices(s...)
			if err != nil {
				return nil, err
			}
			return c.services[d.Key()], nil
		},
		"services": func(s ...string) ([]*util.CatalogService, error) {
			d, err := util.ParseCatalogServices(s...)
			if err != nil {
				return nil, err
			}
			return c.catalogServices[d.Key()], nil
		},
		"tree": func(s string) []*util.KeyPair {
			var result []*util.KeyPair
			// Filter empty keys (folder nodes)
			for _, pair := range c.keyPrefixes[s] {
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

	deps := make(map[string]dependencyContextBridge)

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		// API functions
		"file":      fileFunc(deps),
		"key":       keyFunc(deps),
		"keyPrefix": keyPrefixFunc(deps),
		"ls":        keyPrefixFunc(deps),
		"nodes":     nodesFunc(deps),
		"service":   serviceFunc(deps),
		"services":  catalogServicesFunc(deps),
		"tree":      keyPrefixFunc(deps),

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

	dependencies := make([]dependencyContextBridge, 0, len(deps))
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
		if !dep.inContext(c) {
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

// catalogServicesFunc parses the value from the template into a usable object.
func catalogServicesFunc(deps map[string]dependencyContextBridge) func(...string) (interface{}, error) {
	return func(s ...string) (interface{}, error) {
		d, err := util.ParseCatalogServices(s...)
		if err != nil {
			return nil, err
		}

		if _, ok := deps[d.HashCode()]; !ok {
			deps[d.HashCode()] = &catalogServicesBridge{d}
		}

		var result map[string][]string
		return result, nil
	}
}

// fileFunc parses the value from the template into a usable object.
func fileFunc(deps map[string]dependencyContextBridge) func(...string) (interface{}, error) {
	return func(s ...string) (interface{}, error) {
		if len(s) != 1 {
			return nil, fmt.Errorf("file: expected 1 argument, got %d", len(s))
		}

		d, err := util.ParseFile(s[0])
		if err != nil {
			return nil, err
		}

		if _, ok := deps[d.HashCode()]; !ok {
			deps[d.HashCode()] = &fileBridge{d}
		}

		return "", nil
	}
}

// keyFunc parses the value from the template into a usable object.
func keyFunc(deps map[string]dependencyContextBridge) func(...string) (interface{}, error) {
	return func(s ...string) (interface{}, error) {
		if len(s) != 1 {
			return nil, fmt.Errorf("key: expected 1 argument, got %d", len(s))
		}

		d, err := util.ParseStoreKey(s[0])
		if err != nil {
			return nil, err
		}

		if _, ok := deps[d.HashCode()]; !ok {
			deps[d.HashCode()] = &storeKeyBridge{d}
		}

		return "", nil
	}
}

// keyPrefixFunc parses the value from the template into a usable object.
func keyPrefixFunc(deps map[string]dependencyContextBridge) func(...string) (interface{}, error) {
	return func(s ...string) (interface{}, error) {
		if len(s) != 1 {
			return nil, fmt.Errorf("keyPrefix: expected 1 argument, got %d", len(s))
		}

		d, err := util.ParseKeyPrefix(s[0])
		if err != nil {
			return nil, err
		}

		if _, ok := deps[d.HashCode()]; !ok {
			deps[d.HashCode()] = &keyPrefixBridge{d}
		}

		return []*util.KeyPair{}, nil
	}
}

// nodesFunc parses the value from the template into a usable object.
func nodesFunc(deps map[string]dependencyContextBridge) func(...string) (interface{}, error) {
	return func(s ...string) (interface{}, error) {
		d, err := util.ParseCatalogNodes(s...)
		if err != nil {
			return nil, err
		}

		if _, ok := deps[d.HashCode()]; !ok {
			deps[d.HashCode()] = &catalogNodesBridge{d}
		}

		return []*util.Node{}, nil
	}
}

// serviceFunc parses the value from the template into a usable object.
func serviceFunc(deps map[string]dependencyContextBridge) func(...string) (interface{}, error) {
	return func(s ...string) (interface{}, error) {
		d, err := util.ParseHealthServices(s...)
		if err != nil {
			return nil, err
		}

		if _, ok := deps[d.HashCode()]; !ok {
			deps[d.HashCode()] = &serviceDependencyBridge{d}
		}

		return []*util.HealthService{}, nil
	}
}
