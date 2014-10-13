package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"text/template"

	"github.com/armon/consul-api"
)

type Template struct {
	Input string
}

// GoString returns the detailed format of this object
func (t *Template) GoString() string {
	return fmt.Sprintf("*%#v", *t)
}

// Dependencies returns the dependencies that this template has.
func (t *Template) Dependencies() ([]*Dependency, error) {
	var deps []*Dependency

	contents, err := ioutil.ReadFile(t.Input)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		"service":   t.dependencyAcc(&deps, DependencyTypeService),
		"key":       t.dependencyAcc(&deps, DependencyTypeKey),
		"keyPrefix": t.dependencyAcc(&deps, DependencyTypeKeyPrefix),
	}).Parse(string(contents))

	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(ioutil.Discard, nil)
	if err != nil {
		return nil, err
	}

	return deps, nil
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
	contents, err := ioutil.ReadFile(t.Input)
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

	return nil
}

// Helper function that is used by the dependency collecting.
func (t *Template) dependencyAcc(d *[]*Dependency, dt DependencyType) interface{} {
	return func(s string) interface{} {
		*d = append(*d, &Dependency{Type: dt, Value: s})

		switch dt {
		case DependencyTypeService:
			return []*Service{}
		case DependencyTypeKey:
			return ""
		case DependencyTypeKeyPrefix:
			return []*KeyPair{}
		default:
			panic(fmt.Sprintf("unexpected DependencyType %#v", dt))
		}
	}
}

// Validates that all required dependencies in t are defined in c.
func (t *Template) validateDependencies(c *TemplateContext) error {
	deps, err := t.Dependencies()
	if err != nil {
		return err
	}

	for _, dep := range deps {
		switch dep.Type {
		case DependencyTypeService:
			if _, ok := c.Services[dep.Value]; !ok {
				return fmt.Errorf("templateContext missing service `%s'", dep.Value)
			}
		case DependencyTypeKey:
			if _, ok := c.Keys[dep.Value]; !ok {
				return fmt.Errorf("templateContext missing key `%s'", dep.Value)
			}
		case DependencyTypeKeyPrefix:
			if _, ok := c.KeyPrefixes[dep.Value]; !ok {
				return fmt.Errorf("templateContext missing keyPrefix `%s'", dep.Value)
			}
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
func (c *TemplateContext) Evaluator(dt DependencyType) interface{} {
	return func(s string) interface{} {
		switch dt {
		case DependencyTypeService:
			return c.Services[s]
		case DependencyTypeKey:
			return c.Keys[s]
		case DependencyTypeKeyPrefix:
			return c.KeyPrefixes[s]
		default:
			panic(fmt.Sprintf("unexpected DependencyType %#v", dt))
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
	Port    uint
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

// NewFromConsul creates a new KeyPair object by parsing the values in the
// consulapi.KVPair. Not all values are transferred.
func (kp KeyPair) NewFromConsul(c *consulapi.KVPair) {
	// TODO: lol
	panic("not done!")
}

// DependencyType is an enum type that says the kind of the dependency.
type DependencyType byte

const (
	DependencyTypeInvalid DependencyType = iota
	DependencyTypeService
	DependencyTypeKey
	DependencyTypeKeyPrefix
)
