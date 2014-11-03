package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/hashicorp/consul-template/util"
)

type Template struct {
	//
	Path string

	//
	dependencies []util.Dependency
}

//
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

// Decodestring calls jsonpath.DecodeString, which returns a structure for valid json
func DecodeString(s string) (interface{}, error) {
	// Empty string returns an empty interface
	if len(s) < 2 {
		return map[string]interface{}{}, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return nil, err
	}

	return data, nil
}

// ServiceByTag is a template func that takes the provided services and
// produces a map based on service tags.
//
// The map key is a string representing the service tag. The map value is a
// slice of Services which have the tag assigned.
func ServiceByTag(in []*util.Service) map[string][]*util.Service {
	m := make(map[string][]*util.Service)
	for _, s := range in {
		for _, t := range s.Tags {
			m[t] = append(m[t], s)
		}
	}
	return m
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
		"file":      c.Evaluator(DependencyTypeFile),
		"keyPrefix": c.Evaluator(DependencyTypeKeyPrefix),
		"key":       c.Evaluator(DependencyTypeKey),
		"service":   c.Evaluator(DependencyTypeService),

		// Helper functions
		"byTag":   c.groupByTag,
		"toJSON":  c.decodeJSON,
		"toLower": c.toLower,
		"toTitle": c.toTitle,
		"toUpper": c.toUpper,
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

	depsMap := make(map[string]util.Dependency)

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		// API functions
		"file":      t.dependencyAcc(depsMap, DependencyTypeFile),
		"keyPrefix": t.dependencyAcc(depsMap, DependencyTypeKeyPrefix),
		"key":       t.dependencyAcc(depsMap, DependencyTypeKey),
		"service":   t.dependencyAcc(depsMap, DependencyTypeService),

		// Helper functions
		"byTag":   t.noop,
		"toJSON":  t.noop,
		"toLower": t.noop,
		"toTitle": t.noop,
		"toUpper": t.noop,
	}).Parse(string(contents))

	if err != nil {
		return err
	}

	err = tmpl.Execute(ioutil.Discard, nil)
	if err != nil {
		return err
	}

	dependencies := make([]util.Dependency, 0, len(depsMap))
	for _, dep := range depsMap {
		dependencies = append(dependencies, dep)
	}
	depsMap = nil

	t.dependencies = dependencies

	return nil
}

// Helper function that is used by the dependency collecting.
func (t *Template) dependencyAcc(depsMap map[string]util.Dependency, dt DependencyType) func(string) (interface{}, error) {
	return func(s string) (interface{}, error) {
		switch dt {
		case DependencyTypeFile:
			d, err := util.ParseFileDependency(s)
			if err != nil {
				return nil, err
			}
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}

			return "", nil
		case DependencyTypeKeyPrefix:
			d, err := util.ParseKeyPrefixDependency(s)
			if err != nil {
				return nil, err
			}
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}

			return []*util.KeyPair{}, nil
		case DependencyTypeKey:
			d, err := util.ParseKeyDependency(s)
			if err != nil {
				return nil, err
			}
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}

			return "", nil
		case DependencyTypeService:
			d, err := util.ParseServiceDependency(s)
			if err != nil {
				return nil, err
			}
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}

			return []*util.Service{}, nil
		default:
			return nil, fmt.Errorf("unknown DependencyType %#v", dt)
		}
	}
}

// Validates that all required dependencies in t are defined in c.
func (t *Template) validateDependencies(c *TemplateContext) error {
	for _, dep := range t.Dependencies() {
		switch d := dep.(type) {
		case *util.FileDependency:
			if _, ok := c.File[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing file `%s'", d.Key())
			}
		case *util.KeyPrefixDependency:
			if _, ok := c.KeyPrefixes[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing keyPrefix `%s'", d.Key())
			}
		case *util.KeyDependency:
			if _, ok := c.Keys[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing key `%s'", d.Key())
			}
		case *util.ServiceDependency:
			if _, ok := c.Services[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing service `%s'", d.Key())
			}
		default:
			return fmt.Errorf("unknown dependency type in template e %#v", d)
		}
	}

	return nil
}

// noop is a special function that returns itself. This is used during the
// dependency accumulation to allow the template to be processed once.
func (t *Template) noop(thing interface{}) (interface{}, error) {
	return thing, nil
}

// TemplateContext is what Template uses to determine the values that are
// available for template parsing.
type TemplateContext struct {
	Services    map[string][]*util.Service
	Keys        map[string]string
	KeyPrefixes map[string][]*util.KeyPair
	File        map[string]string
}

// Evaluator takes a DependencyType and returns a function which returns the
// value in the TemplateContext that corresponds to the requested item.
func (c *TemplateContext) Evaluator(dt DependencyType) func(string) (interface{}, error) {
	return func(s string) (interface{}, error) {
		switch dt {
		case DependencyTypeFile:
			return c.File[s], nil
		case DependencyTypeKeyPrefix:
			return c.KeyPrefixes[s], nil
		case DependencyTypeKey:
			return c.Keys[s], nil
		case DependencyTypeService:
			return c.Services[s], nil
		default:
			return nil, fmt.Errorf("unexpected DependencyType %#v", dt)
		}
	}
}

// decodeJSON returns a structure for valid JSON
func (c *TemplateContext) decodeJSON(s string) (interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return nil, err
	}

	return data, nil
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

// DependencyType is an enum type that says the kind of the dependency.
type DependencyType byte

const (
	DependencyTypeInvalid DependencyType = iota
	DependencyTypeService
	DependencyTypeKey
	DependencyTypeKeyPrefix
	DependencyTypeFile
)
