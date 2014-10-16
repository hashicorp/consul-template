package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Renderer is a struct responsible for determining if a Template is renderable
// and performing the actual rendering steps.
type Renderer struct {
	// dependencies is the slice of Dependencies this Renderer knows how to
	// render.
	dependencies []Dependency

	// dry is the mode in which Templates are "rendered" to a stream (default is
	// stdout). This stream can be set using the SetDryStream() function.
	dry       bool
	dryStream io.Writer

	// dependencyDataMap is a map of each dependency to the data received from a
	// poll.
	dependencyDataMap map[Dependency]interface{}
}

// NewRenderer accepts a slice of Dependencies and a "dry" flag. The slice of
// Dependencies corresponds to all the Dependencies this Renderer cares about.
// The "dry" flag triggers a special mode in the Renderer where output files are
// written to an IO stream instead of being written to disk. This mode is useful
// for debugging or testing changes before actually applying them. This IO
// stream defaults to os.Stdout but can be changed used the SetDryStream()
// function.
//
// This function returns a pointer to the new Renderer and any error(s) that
// occurred during creation.
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

// SetDryStream accepts an io.Writer and sets the internal dryStream for this
// Renderer.
func (r *Renderer) SetDryStream(s io.Writer) {
	r.dryStream = s
}

// Receive accepts a Dependency and data for that Dependency. This data is
// cached on the Renderer. This data is then used to determine if a Template
// is "renderable" (i.e. all its Dependencies have been downloaded at least
// once).
func (r *Renderer) Receive(dependency Dependency, data interface{}) {
	r.dependencyDataMap[dependency] = data
}

// MaybeRender accepts a Template and slice of ConfigTemplates that created that
// Template. If the template is "renderable" (i.e. all its Dependencies have
// been downlaoded at least once), the most recent version of the data from the
// Template's Dependencies is added to a TemplateContext and Executed. If there
// is a failure, an error is returned.
func (r *Renderer) MaybeRender(template *Template, configTemplates []*ConfigTemplate) error {
	if !r.canRender(template) {
		return nil
	}

	context, err := r.templateContextFor(template)
	if err != nil {
		return err
	}

	contents, err := template.Execute(context)
	if err != nil {
		return err
	}

	for _, configTemplate := range configTemplates {
		destination := configTemplate.Destination

		if r.dry {
			fmt.Fprintf(r.dryStream, "> %s\n%s", destination, contents)
		} else {
			if err := r.atomicWrite(destination, contents); err != nil {
				return err
			}
		}
	}

	return nil
}

// init() creates the Renderer's underlying data structures and returns an error
// if any problems occur.
func (r *Renderer) init() error {
	if len(r.dependencies) == 0 {
		return fmt.Errorf("renderer: must supply at least one Dependency")
	}

	r.dependencyDataMap = make(map[Dependency]interface{})
	r.dryStream = os.Stdout

	return nil
}

// canRender accepts a Template and returns true if and only if all of the
// Dependencies of that template have data in the Renderer.
func (r *Renderer) canRender(template *Template) bool {
	for _, dependency := range template.Dependencies() {
		if r.dependencyDataMap[dependency] == nil {
			return false
		}
	}

	return true
}

// templateContextFor creates and returns a new TemplateContext for the given
// Template, iterating through all the Template's Dependencies and appending
// them where appropriate in the TemplateContext.
//
// If an unknown Dependency.(type) is encountered, an error is returned.
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

// atomicWrite accepts a destination path and the template contents. It writes
// the template contents to a TempFile on disk, returning if any errors occur.
//
// If the parent destination directory does not exist, it will be created
// automatically with permissions 0755. To use a different permission, create
// the directory first or use `chmod` in a Command.
//
// If the destination path exists, all attempts will be made to preserve the
// existing file permissions. If those permissions cannot be read, an error is
// returned. If the file does not exist, it will be created automatically with
// permissions 0644. To use a different permission, create the destination file
// first or use `chmod` in a Command.
//
// If no errors occur, the Tempfile is "renamed" (moved) to the destination
// path.
func (r *Renderer) atomicWrite(path string, contents []byte) error {
	var mode os.FileMode

	// If the current file exists, get permissions so we can preserve them
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			mode = 0644
		} else {
			return err
		}
	} else {
		mode = stat.Mode()
	}

	f, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(contents); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	if err := os.Chmod(f.Name(), mode); err != nil {
		return err
	}

	parent := filepath.Dir(path)
	if _, err := os.Stat(parent); os.IsNotExist(err) {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return err
		}
	}

	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}

	return nil
}
