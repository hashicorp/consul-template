package main

import (
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

func NewWatcher(config *Config) *Watcher {
	return &Watcher{
		config: config,
	}
}

// Watch starts the Watcher process, querying the Consul API and rendering any
// configuration file changes.
func (w *Watcher) Watch() error {
	client, err := w.client()
	if err != nil {
		return err
	}

	views, templates, err := w.createViews()
	if err != nil {
		return err
	}

	changes := w.waitForChanges(views, client)
	for {
		select {
		case view := <-changes:
			for _, template := range view.Templates {
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

				template.Execute(os.Stderr, context)
			}
		}
	}

	return nil
}

//
func (w *Watcher) waitForChanges(views map[Dependency]*DataView, client *api.Client) <-chan *DataView {
	ch := make(chan *DataView, len(views))
	for _, view := range views {
		go view.poll(ch, client)
	}
	return ch
}

//
func (w *Watcher) createViews() (map[Dependency]*DataView, map[*Template][]Dependency, error) {
	// Use a sane starting size - it is assumed that each ConfigTemplate has at
	// least one Dependency
	views, templates := make(map[Dependency]*DataView), make(map[*Template][]Dependency)

	// For each Dependency per ConfigTemplate, construct a DataView object which
	// ties the dependency to the Templates which depend on it
	for _, ctemplate := range w.config.ConfigTemplates {
		template := &Template{Input: ctemplate.Source}
		deps, err := template.Dependencies()
		if err != nil {
			return nil, nil, err
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
	}

	return views, templates, nil
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
func (view *DataView) poll(ch chan *DataView, client *api.Client) {
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
	}
}

//
func (view *DataView) loaded() bool {
	return view.LastIndex != 0
}
