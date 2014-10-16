package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"text/template"
)

type Template struct {
	//
	path string

	//
	dependencies []Dependency

	// Internal variable to represent that a template has been rendered
	rendered bool
}

//
func NewTemplate(path string) (*Template, error) {
	template := &Template{path: path}
	if err := template.init(); err != nil {
		return nil, err
	}

	return template, nil
}

// Path returns the path to this Template
func (t *Template) Path() string {
	return t.path
}

// HashCode returns the map value for this Template
func (t *Template) HashCode() string {
	return fmt.Sprintf("Template|%s", t.path)
}

// Rendered returns true if the template has been executed
func (t *Template) Rendered() bool {
	return t.rendered
}

// GoString returns the detailed format of this object
func (t *Template) GoString() string {
	return fmt.Sprintf("*%#v", *t)
}

// Dependencies returns the dependencies that this template has.
func (t *Template) Dependencies() []Dependency {
	return t.dependencies
}

// Execute takes the given template context and processes the template.
//
// If the TemplateContext is nil, an error will be returned.
//
// If the TemplateContext does not have all required Dependencies, an error will
// be returned.
func (t *Template) Execute(wr io.Writer, c *TemplateContext) error {
	if wr == nil {
		return errors.New("wr must be given")
	}

	if c == nil {
		return errors.New("templateContext must be given")
	}

	// Make sure the context contains everything we need
	if err := t.validateDependencies(c); err != nil {
		return err
	}

	// Render the template
	contents, err := ioutil.ReadFile(t.path)
	if err != nil {
		return err
	}

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		"service":   c.Evaluator(DependencyTypeService),
		"key":       c.Evaluator(DependencyTypeKey),
		"keyPrefix": c.Evaluator(DependencyTypeKeyPrefix),
	}).Parse(string(contents))

	if err != nil {
		return err
	}

	err = tmpl.Execute(wr, c)
	if err != nil {
		return err
	}

	t.rendered = true

	return nil
}

// init reads the template file and parses all the required dependencies into a
// dependencies slice which is then added onto the Template.
func (t *Template) init() error {
	contents, err := ioutil.ReadFile(t.path)
	if err != nil {
		return err
	}

	depsMap := make(map[string]Dependency)

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		"service":   t.dependencyAcc(depsMap, DependencyTypeService),
		"key":       t.dependencyAcc(depsMap, DependencyTypeKey),
		"keyPrefix": t.dependencyAcc(depsMap, DependencyTypeKeyPrefix),
	}).Parse(string(contents))

	if err != nil {
		return err
	}

	err = tmpl.Execute(ioutil.Discard, nil)
	if err != nil {
		return err
	}

	dependencies := make([]Dependency, 0, len(depsMap))
	for _, dep := range depsMap {
		dependencies = append(dependencies, dep)
	}
	depsMap = nil

	t.dependencies = dependencies

	return nil
}

// Helper function that is used by the dependency collecting.
func (t *Template) dependencyAcc(depsMap map[string]Dependency, dt DependencyType) func(string) (interface{}, error) {
	return func(s string) (interface{}, error) {
		switch dt {
		case DependencyTypeService:
			d, err := ParseServiceDependency(s)
			if err != nil {
				return nil, err
			}
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}

			return []*Service{}, nil
		case DependencyTypeKey:
			d, err := ParseKeyDependency(s)
			if err != nil {
				return nil, err
			}
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}

			return "", nil
		case DependencyTypeKeyPrefix:
			d, err := ParseKeyPrefixDependency(s)
			if err != nil {
				return nil, err
			}
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}

			return []*KeyPair{}, nil
		default:
			return nil, fmt.Errorf("unknown DependencyType %#v", dt)
		}
	}
}

// Validates that all required dependencies in t are defined in c.
func (t *Template) validateDependencies(c *TemplateContext) error {
	for _, dep := range t.Dependencies() {
		switch d := dep.(type) {
		case *ServiceDependency:
			if _, ok := c.Services[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing service `%s'", d.Key())
			}
		case *KeyDependency:
			if _, ok := c.Keys[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing key `%s'", d.Key())
			}
		case *KeyPrefixDependency:
			if _, ok := c.KeyPrefixes[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing keyPrefix `%s'", d.Key())
			}
		default:
			return fmt.Errorf("unknown dependency type %#v", d)
		}
	}

	return nil
}

/// ------------------------- ///

// TemplateContext is what Template uses to determine the values that are
// available for template parsing.
type TemplateContext struct {
	Services    map[string][]*Service
	Keys        map[string]string
	KeyPrefixes map[string][]*KeyPair
}

// GoString returns the detailed format of this object
func (c *TemplateContext) GoString() string {
	return fmt.Sprintf("*%#v", *c)
}

// Evaluator takes a DependencyType and returns a function which returns the
// value in the TemplateContext that corresponds to the requested item.
func (c *TemplateContext) Evaluator(dt DependencyType) func(string) (interface{}, error) {
	return func(s string) (interface{}, error) {
		switch dt {
		case DependencyTypeService:
			return c.Services[s], nil
		case DependencyTypeKey:
			return c.Keys[s], nil
		case DependencyTypeKeyPrefix:
			return c.KeyPrefixes[s], nil
		default:
			return nil, fmt.Errorf("unexpected DependencyType %#v", dt)
		}
	}
}

/// ------------------------- ///

type Service struct {
	Node    string
	Address string
	ID      string
	Name    string
	Tags    []string
	Port    uint64
}

// GoString returns the detailed format of this object
func (s *Service) GoString() string {
	return fmt.Sprintf("*%#v", *s)
}

/// ------------------------- ///

type KeyPair struct {
	Key   string
	Value string
}

// GoString returns the detailed format of this object
func (kp *KeyPair) GoString() string {
	return fmt.Sprintf("*%#v", *kp)
}

// DependencyType is an enum type that says the kind of the dependency.
type DependencyType byte

const (
	DependencyTypeInvalid DependencyType = iota
	DependencyTypeService
	DependencyTypeKey
	DependencyTypeKeyPrefix
)
