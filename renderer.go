package main

import (
	"fmt"
	"io"
	"os"
)

//
type Renderer struct {
	dependencies []Dependency
	dry          bool
	dryStream    io.Writer

	dependencyDataMap map[Dependency]interface{}
}

//
func NewRenderer(dependencies []Dependency, dry bool) (*Renderer, error) {
	renderer := &Renderer{
		dependencies: dependencies,
		dry:          dry,
	}
	if err := renderer.init(); err != nil {
		return nil, err
	}
	return renderer, nil
}

//
func (r *Renderer) SetDryStream(s io.Writer) {
	r.dryStream = s
}

//
func (r *Renderer) Receive(dependency Dependency, data interface{}) {
	r.dependencyDataMap[dependency] = data
}

//
func (r *Renderer) MaybeRender(template *Template, configTemplates []*ConfigTemplate) error {
	if r.canRender(template) {
		context, err := r.templateContextFor(template)
		if err != nil {
			return err
		}

		contents, err := template.Execute(context)
		if err != nil {
			return err
		}

		for _, configTemplate := range configTemplates {
			if r.dry {
				fmt.Fprintf(r.dryStream, "> %s\n%s", configTemplate.Destination, contents)
			} else {
				f, err := os.Create(configTemplate.Destination)
				if err != nil {
					return err
				}
				defer f.Close()

				f.Write(contents)
				f.Sync()
			}
		}
	}

	return nil
}

//
func (r *Renderer) init() error {
	if len(r.dependencies) == 0 {
		return fmt.Errorf("renderer: must supply at least one Dependency")
	}

	r.dependencyDataMap = make(map[Dependency]interface{})
	r.dryStream = os.Stdout

	return nil
}

// canRender accepts a Template and returns true iff all the Dependencies of
// that template have data in the Renderer.
func (r *Renderer) canRender(template *Template) bool {
	for _, dependency := range template.Dependencies() {
		if r.dependencyDataMap[dependency] == nil {
			return false
		}
	}

	return true
}

// templateContextFor creates a TemplateContext for the given Template,
// iterating through all the Template's Dependencies and appending them where
// appropriate in the TemplateContext. If an unknown Dependency is encountered,
// an error will be returned.
func (r *Renderer) templateContextFor(template *Template) (*TemplateContext, error) {
	context := &TemplateContext{
		Services:    make(map[string][]*Service),
		Keys:        make(map[string]string),
		KeyPrefixes: make(map[string][]*KeyPair),
	}

	for _, dependency := range template.Dependencies() {
		data := r.dependencyDataMap[dependency]

		switch dependency := dependency.(type) {
		case *ServiceDependency:
			context.Services[dependency.Key()] = data.([]*Service)
		case *KeyDependency:
			context.Keys[dependency.Key()] = data.(string)
		case *KeyPrefixDependency:
			context.KeyPrefixes[dependency.Key()] = data.([]*KeyPair)
		default:
			return nil, fmt.Errorf("unknown dependency type %#v", dependency)
		}
	}

	return context, nil
}
