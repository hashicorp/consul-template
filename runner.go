package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/consul-template/util"
)

// Runner responsible rendering Templates and invoking Commands.
type Runner struct {
	// outStream and errStream are the io.Writer streams where the runner will
	// write information. These streams can be set using the SetOutStream()
	// and SetErrStream() functions.
	outStream, errStream io.Writer

	// configTemplates, templates, and dependencies are internally calculated
	// caches of all the data this Runner knows about.
	configTemplates []*ConfigTemplate
	templates       []*Template
	dependencies    []util.Dependency

	// templateConfigTemplateMap is a map of each template to the ConfigTemplates
	// that made it.
	templateConfigTemplateMap map[string][]*ConfigTemplate

	// dependencyDataMap is a map of each dependency to its data.
	dependencyDataReceivedMap map[string]struct{}
	dependencyDataMap         map[string]interface{}
}

// NewRunner accepts a slice of ConfigTemplates and returns a pointer to the new
// Runner and any error that occurred during creation.
func NewRunner(configTemplates []*ConfigTemplate) (*Runner, error) {
	runner := &Runner{configTemplates: configTemplates}
	if err := runner.init(); err != nil {
		return nil, err
	}

	return runner, nil
}

// SetOutStream accepts an io.Writer and sets the internal outStream for this
// Runner.
func (r *Runner) SetOutStream(s io.Writer) {
	r.outStream = s
}

// SetErrStream accepts an io.Writer and sets the internal outStream for this
// Runner.
func (r *Runner) SetErrStream(s io.Writer) {
	r.errStream = s
}

// Receive accepts a Dependency and data for that Dependency. This data is
// cached on the Runner. This data is then used to determine if a Template
// is "renderable" (i.e. all its Dependencies have been downloaded at least
// once).
func (r *Runner) Receive(dependency util.Dependency, data interface{}) {
	r.dependencyDataReceivedMap[dependency.HashCode()] = struct{}{}
	r.dependencyDataMap[dependency.HashCode()] = data
}

// Dependencies returns the unique slice of Dependency in the Runner
func (r *Runner) Dependencies() []util.Dependency {
	return r.dependencies
}

// RunAll iterates over each template in this Runner and conditionally executes
// the template rendering and command execution.
//
// The template is rendered atomicly. If and only if the template render
// completes successfully, the optional commands will be executed, if given.
// Please note that all templates are rendered **and then** any commands are
// executed.
//
// If the dry flag is given, the template will be rendered to the outStream,
// which defaults to os.Stdout. In dry mode, commands are never executed.
func (r *Runner) RunAll(dry bool) error {
	var commands []string

	for _, template := range r.templates {
		// If the template is not ready to be rendered, just return
		if !r.canRender(template) {
			return nil
		}

		for _, ctemplate := range r.configTemplatesFor(template) {
			// Render the template, taking dry mode into account
			rendered, err := r.render(template, ctemplate.Destination, dry)
			if err != nil {
				return err
			}

			// If the template was rendered (changed) and we are not in dry-run mode,
			// aggregate commands
			if rendered && !dry {
				if ctemplate.Command != "" {
					commands = append(commands, ctemplate.Command)
				}
			}
		}
	}

	// Execute each command in sequence, collecting any errors that occur - this
	// ensures all commands execute at least once
	var errs []error
	for _, command := range commands {
		if err := r.execute(command); err != nil {
			errs = append(errs, err)
		}
	}

	// If any errors were returned, convert them to an ErrorList for human
	// readability
	if len(errs) != 0 {
		errors := NewErrorList("running commands")
		for _, err := range errs {
			errors.Append(err)
		}
		return errors.GetError()
	}

	return nil
}

// init() creates the Runner's underlying data structures and returns an error
// if any problems occur.
func (r *Runner) init() error {
	if len(r.configTemplates) == 0 {
		r.configTemplates = make([]*ConfigTemplate, 0)
	}

	templatesMap := make(map[string]*Template)
	dependenciesMap := make(map[string]util.Dependency)

	r.templateConfigTemplateMap = make(map[string][]*ConfigTemplate)

	// Process all Template first, so we can return errors
	for _, configTemplate := range r.configTemplates {
		template := &Template{Path: configTemplate.Source}
		if _, ok := templatesMap[template.HashCode()]; !ok {
			template, err := NewTemplate(configTemplate.Source)
			if err != nil {
				return err
			}
			templatesMap[template.HashCode()] = template
		}

		if len(r.templateConfigTemplateMap[template.HashCode()]) == 0 {
			r.templateConfigTemplateMap[template.HashCode()] = make([]*ConfigTemplate, 0, 1)
		}
		r.templateConfigTemplateMap[template.HashCode()] = append(r.templateConfigTemplateMap[template.HashCode()], configTemplate)
	}

	// For each Template, setup the mappings for O(1) lookups
	for _, template := range templatesMap {
		for _, dep := range template.Dependencies() {
			if _, ok := dependenciesMap[dep.HashCode()]; !ok {
				dependenciesMap[dep.HashCode()] = dep
			}
		}
	}

	// Calculate the list of Templates
	r.templates = make([]*Template, 0, len(templatesMap))
	for _, template := range templatesMap {
		r.templates = append(r.templates, template)
	}

	// Calculate the list of Dependency
	r.dependencies = make([]util.Dependency, 0, len(dependenciesMap))
	for _, dependency := range dependenciesMap {
		r.dependencies = append(r.dependencies, dependency)
	}

	r.dependencyDataReceivedMap = make(map[string]struct{})
	r.dependencyDataMap = make(map[string]interface{})
	r.outStream = os.Stdout

	return nil
}

// canRender accepts a Template and returns true if and only if all of the
// Dependencies of that template have data in the Runner.
func (r *Runner) canRender(template *Template) bool {
	for _, dependency := range template.Dependencies() {
		if !r.receivedData(dependency) {
			return false
		}
	}

	return true
}

// Render accepts a Template and a destinations. This will return an error if
// the Template is not ready to be rendered. You can check if a Template is
// renderable using canRender().
//
// If the template has changed on disk, this method return true.
//
// If the template already exists and has the same contents as the "would-be"
// template, no action is taken. In this scenario, the render function returns
// false, indicating no template change has occurred.
func (r *Runner) render(template *Template, destination string, dry bool) (bool, error) {
	if !r.canRender(template) {
		return false, fmt.Errorf("runner: template data not ready")
	}

	context, err := r.templateContextFor(template)
	if err != nil {
		return false, err
	}

	contents, err := template.Execute(context)
	if err != nil {
		return false, err
	}

	existingContents, err := ioutil.ReadFile(destination)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(contents, existingContents) {
		return false, nil
	}

	if dry {
		fmt.Fprintf(r.outStream, "> %s\n%s", destination, contents)
	} else {
		if err := r.atomicWrite(destination, contents); err != nil {
			return false, err
		}
	}

	return true, nil
}

// execute accepts a command string and runs that command string on the current
// system.
func (r *Runner) execute(command string) error {
	var shell, flag string
	if runtime.GOOS == "windows" {
		shell, flag = "cmd", "/C"
	} else {
		shell, flag = "/bin/sh", "-c"
	}

	// Create an invoke the command
	cmd := exec.Command(shell, flag, command)
	cmd.Stdout = r.outStream
	cmd.Stderr = r.errStream
	return cmd.Run()
}

// receivedData returns true if the Runner has ever received data for the given
// dependency and false otherwise.
func (r *Runner) receivedData(dependency util.Dependency) bool {
	_, ok := r.dependencyDataReceivedMap[dependency.HashCode()]
	return ok
}

// data returns the data for the given dependency.
func (r *Runner) data(dependency util.Dependency) interface{} {
	return r.dependencyDataMap[dependency.HashCode()]
}

// ConfigTemplateFor returns the ConfigTemplate for the given Template
func (r *Runner) configTemplatesFor(template *Template) []*ConfigTemplate {
	return r.templateConfigTemplateMap[template.HashCode()]
}

// templateContextFor creates and returns a new TemplateContext for the given
// Template, iterating through all the Template's Dependencies and appending
// them where appropriate in the TemplateContext.
//
// If an unknown Dependency.(type) is encountered, an error is returned.
func (r *Runner) templateContextFor(template *Template) (*TemplateContext, error) {
	context := &TemplateContext{
		File:        make(map[string]string),
		KeyPrefixes: make(map[string][]*util.KeyPair),
		Keys:        make(map[string]string),
		Nodes:       make(map[string][]*util.Node),
		Services:    make(map[string][]*util.Service),
	}

	for _, dependency := range template.Dependencies() {
		data := r.data(dependency)

		switch dependency := dependency.(type) {
		case *util.FileDependency:
			context.File[dependency.Key()] = data.(string)
		case *util.KeyPrefixDependency:
			context.KeyPrefixes[dependency.Key()] = data.([]*util.KeyPair)
		case *util.KeyDependency:
			context.Keys[dependency.Key()] = data.(string)
		case *util.NodeDependency:
			context.Nodes[dependency.Key()] = data.([]*util.Node)
		case *util.ServiceDependency:
			context.Services[dependency.Key()] = data.([]*util.Service)
		default:
			return nil, fmt.Errorf("unknown dependency type %+v", dependency)
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
func (r *Runner) atomicWrite(path string, contents []byte) error {
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

	parent := filepath.Dir(path)
	if _, err := os.Stat(parent); os.IsNotExist(err) {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return err
		}
	}

	f, err := ioutil.TempFile(parent, "")
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

	// Remove the file if we are running on Windows. There is a bug in Go on
	// Windows such that Go uses MoveFile which raises an exception if the file
	// already exists.
	//
	// See: http://grokbase.com/t/gg/golang-nuts/13aab5f210/go-nuts-atomic-replacement-of-files-on-windows
	// for more information.
	if runtime.GOOS == "windows" {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}

	return nil
}
