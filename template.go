package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"text/template"

	dep "github.com/hashicorp/consul-template/dependency"
)

// Template is the Go representation of a Consul Template template on disk.
type Template struct {
	sync.Mutex

	// The path to this Template on disk.
	Path string

	// The internal list of dependencies for this Template.
	dependencies []dep.Dependency

	// contents is string contents for this file from disk
	contents string
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

//
func (t *Template) Execute(brain *Brain) ([]dep.Dependency, []byte, error) {
	usedMap := make(map[string]dep.Dependency)
	missingMap := make(map[string]dep.Dependency)
	name := filepath.Base(t.Path)
	funcs := funcMap(brain, usedMap, missingMap)

	tmpl, err := template.New(name).Funcs(funcs).Parse(t.contents)
	if err != nil {
		return nil, nil, fmt.Errorf("template: %s", err)
	}

	// TODO: accept an io.Writer instead
	buff := new(bytes.Buffer)
	if err := tmpl.Execute(buff, nil); err != nil {
		return nil, nil, fmt.Errorf("template: %s", err)
	}

	// Lock because we are about to update the internal state of this template,
	// which could be happening concurrently...
	t.Lock()
	defer t.Unlock()

	// Update this list of this template's dependencies
	var used []dep.Dependency
	for _, dep := range usedMap {
		used = append(used, dep)
	}
	t.dependencies = used

	// Compile the list of missing dependencies
	var missing []dep.Dependency
	for _, dep := range missingMap {
		missing = append(missing, dep)
	}

	return missing, buff.Bytes(), nil
}

// Dependencies returns the dependencies that this template currently has
// recorded to require.
func (t *Template) Dependencies() []dep.Dependency {
	t.Lock()
	defer t.Unlock()

	return t.dependencies
}

// HashCode returns the map value for this Template
func (t *Template) HashCode() string {
	return fmt.Sprintf("Template|%s", t.Path)
}

// init reads the template file and initializes required variables.
func (t *Template) init() error {
	// Render the template
	contents, err := ioutil.ReadFile(t.Path)
	if err != nil {
		return err
	}
	t.contents = string(contents)

	return nil
}

// funcMap is the map of template functions to their respective functions.
func funcMap(brain *Brain, used, missing map[string]dep.Dependency) template.FuncMap {
	return template.FuncMap{
		// API functions
		"file":     fileFunc(brain, used, missing),
		"key":      keyFunc(brain, used, missing),
		"ls":       lsFunc(brain, used, missing),
		"nodes":    nodesFunc(brain, used, missing),
		"service":  serviceFunc(brain, used, missing),
		"services": servicesFunc(brain, used, missing),
		"tree":     treeFunc(brain, used, missing),

		// Helper functions
		"byTag":           byTag,
		"env":             env,
		"parseJSON":       parseJSON,
		"regexReplaceAll": regexReplaceAll,
		"replaceAll":      replaceAll,
		"toLower":         toLower,
		"toTitle":         toTitle,
		"toUpper":         toUpper,
	}
}
