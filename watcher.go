package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	api "github.com/armon/consul-api"
)

const (
	// The amount of time to do a blocking query for
	defaultWaitTime = 60 * time.Second
)

type Watcher struct {
	// config is a Consul Template Config object to be read by the watcher
	config *Config
}

// NewWatcher accepts a Config and creates a new Watcher.
func NewWatcher(config *Config) (*Watcher, error) {
	if config == nil {
		return nil, errors.New("cannot specify empty config")
	}

	watcher := &Watcher{
		config: config,
	}

	return watcher, nil
}

// Watch starts the Watcher process, querying the Consul API and rendering any
// configuration file changes.
func (w *Watcher) Watch() error {
	client, err := w.client()
	if err != nil {
		return err
	}

	views, templates, ctemplates, err := w.createViews()
	if err != nil {
		return err
	}

	doneCh := make(chan struct{})
	viewCh := w.waitForChanges(views, client, doneCh)

	if w.config.Once {
		// If we are running in "once" mode, quit after the first poll
		close(doneCh)
	} else {
		// Ensure we stop background polling when we quit the Watcher
		defer close(doneCh)
	}

	for {
		if w.config.Once && w.renderedAll(views) {
			// If we are in "once" mode and all the templates have been rendered,
			// exit gracefully
			return nil
		}

		select {
		case view := <-viewCh:
			for _, template := range view.Templates {
				if w.config.Once && template.Rendered() {
					// If we are in "once" mode, do not rendered the same template twice
					continue
				}

				deps := templates[template]
				context := &TemplateContext{
					Services:    make(map[string][]*Service),
					Keys:        make(map[string]string),
					KeyPrefixes: make(map[string][]*KeyPair),
				}

				// Continue if not all the required dependencies have been loaded
				// into the views
				if !w.ready(views, deps) {
					continue
				}

				for _, dep := range deps {
					v := views[dep]
					switch d := v.Dependency.(type) {
					case *ServiceDependency:
						context.Services[d.Key()] = v.Data.([]*Service)
					case *KeyDependency:
						context.Keys[d.Key()] = v.Data.(string)
					case *KeyPrefixDependency:
						context.KeyPrefixes[d.Key()] = v.Data.([]*KeyPair)
					default:
						panic(fmt.Sprintf("unknown dependency type: %q", d))
					}
				}

				if w.config.Dry {
					template.Execute(os.Stderr, context)
				} else {
					ctemplate := ctemplates[template]
					out, err := os.OpenFile(ctemplate.Destination, os.O_WRONLY, 0666)
					if err != nil {
						panic(err)
					}

					template.Execute(out, context)
				}
			}
		}
	}

	return nil
}

//
func (w *Watcher) waitForChanges(views map[Dependency]*DataView, client *api.Client, doneCh chan struct{}) <-chan *DataView {
	viewCh := make(chan *DataView, len(views))
	for _, view := range views {
		go view.poll(viewCh, client, doneCh)
	}
	return viewCh
}

//
func (w *Watcher) createViews() (map[Dependency]*DataView, map[*Template][]Dependency, map[*Template]*ConfigTemplate, error) {
	views := make(map[Dependency]*DataView)
	templates := make(map[*Template][]Dependency)
	ctemplates := make(map[*Template]*ConfigTemplate)

	// For each Dependency per ConfigTemplate, construct a DataView object which
	// ties the dependency to the Templates which depend on it
	for _, ctemplate := range w.config.ConfigTemplates {
		template := &Template{Input: ctemplate.Source}
		deps, err := template.Dependencies()
		if err != nil {
			return nil, nil, nil, err
		}

		for _, dep := range deps {
			// Create the DataView if it does not exist
			if _, ok := views[dep]; !ok {
				views[dep] = &DataView{Dependency: dep}
			}

			// Create the Templtes slice if it does not exist
			if len(views[dep].Templates) == 0 {
				views[dep].Templates = make([]*Template, 0, 1)
			}

			// Append the ConfigTemplate to the slice
			views[dep].Templates = append(views[dep].Templates, template)
		}

		// Add the template and its dependencies to the list
		templates[template] = deps

		// Map the template to its ConfigTemplate
		ctemplates[template] = ctemplate
	}

	return views, templates, ctemplates, nil
}

// ready determines if the views have loaded all the required information
// from the dependency.
func (w *Watcher) ready(views map[Dependency]*DataView, deps []Dependency) bool {
	for _, dep := range deps {
		v := views[dep]
		if !v.loaded() {
			return false
		}
	}
	return true
}

//
func (w *Watcher) renderedAll(views map[Dependency]*DataView) bool {
	for _, view := range views {
		for _, template := range view.Templates {
			if !template.Rendered() {
				return false
			}
		}
	}

	return true
}

//
func (w *Watcher) client() (*api.Client, error) {
	consulConfig := api.DefaultConfig()
	if w.config.Consul != "" {
		consulConfig.Address = w.config.Consul
	}
	if w.config.Token != "" {
		consulConfig.Token = w.config.Token
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	if _, err := client.Agent().NodeName(); err != nil {
		return nil, err
	}

	return client, nil
}

/// ------------------------- ///

type DataView struct {
	Templates  []*Template
	Dependency Dependency
	Data       interface{}
	LastIndex  uint64
}

//
func (view *DataView) poll(ch chan *DataView, client *api.Client, doneCh chan struct{}) {
	for {
		options := &api.QueryOptions{
			WaitTime:  defaultWaitTime,
			WaitIndex: view.LastIndex,
		}
		data, qm, err := view.Dependency.Fetch(client, options)
		if err != nil {
			panic(err) // TODO: push err to err ch or something
		}

		println(fmt.Sprintf("[%d] Poll: %#v", time.Now().Unix(), view.Dependency))

		// Consul is allowed to return even if there's no new data. Ignore data if
		// the index is the same.
		if qm.LastIndex == view.LastIndex {
			continue
		}

		// Update the index in case we got a new version, but the data is the same
		view.LastIndex = qm.LastIndex

		// Do not trigger a render if the data is the same
		if reflect.DeepEqual(data, view.Data) {
			continue
		}

		// If we got this far, there is new data!
		view.Data = data
		ch <- view

		// Break from the function if we are done - this happens at the end of the
		// function to ensure it runs at least once
		select {
		case <-doneCh:
			return
		}
	}
}

//
func (view *DataView) loaded() bool {
	return view.LastIndex != 0
}
