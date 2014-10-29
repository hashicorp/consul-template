package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/yasuyuky/jsonpath"
	"io/ioutil"
	"text/template"
)

type Template struct {
	//
	path string

	//
	dependencies []Dependency
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

// GoString returns the detailed format of this object
func (t *Template) GoString() string {
	return fmt.Sprintf("*%#v", *t)
}

// Dependencies returns the dependencies that this template has.
func (t *Template) Dependencies() []Dependency {
	return t.dependencies
}

// Decodestring calls jsonpath.DecodeString, which returns a structure for valid json
func DecodeString(s string) (interface{}, error) {
	if len(s) > 0 {
		return jsonpath.DecodeString(s)
	} else {
		return map[string]interface{}{}, nil
	}
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
	contents, err := ioutil.ReadFile(t.path)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		"service":   c.Evaluator(DependencyTypeService),
		"key":       c.Evaluator(DependencyTypeKey),
		"keyPrefix": c.Evaluator(DependencyTypeKeyPrefix),
		"file":      c.Evaluator(DependencyTypeFile),
		"json":      DecodeString,
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
	contents, err := ioutil.ReadFile(t.path)
	if err != nil {
		return err
	}

	depsMap := make(map[string]Dependency)

	tmpl, err := template.New("out").Funcs(template.FuncMap{
		"service":   t.dependencyAcc(depsMap, DependencyTypeService),
		"key":       t.dependencyAcc(depsMap, DependencyTypeKey),
		"keyPrefix": t.dependencyAcc(depsMap, DependencyTypeKeyPrefix),
		"file":      t.dependencyAcc(depsMap, DependencyTypeFile),
		"json":      DecodeString,
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
		case DependencyTypeFile:
			d, err := ParseFileDependency(s)
			if err != nil {
				return nil, err
			}
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}

			return "", nil
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
		case *FileDependency:
			if _, ok := c.File[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing file `%s'", d.Key())
			}
		case *KeyPrefixDependency:
			if _, ok := c.KeyPrefixes[d.Key()]; !ok {
				return fmt.Errorf("templateContext missing keyPrefix `%s'", d.Key())
			}
		default:
			return fmt.Errorf("unknown dependency type in template e %#v", d)
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
	File        map[string]string
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
		case DependencyTypeFile:
			return c.File[s], nil
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

type ServiceList []*Service

func (s ServiceList) Len() int {
	return len(s)
}

func (s ServiceList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ServiceList) Less(i, j int) bool {
	if s[i].Node < s[j].Node {
		return true
	} else if s[i].Node == s[j].Node {
		return s[i].ID <= s[j].ID
	}
	return false
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
	DependencyTypeFile
)
