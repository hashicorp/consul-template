package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/consul/api"
	multierror "github.com/hashicorp/go-multierror"
)

// Runner responsible rendering Templates and invoking Commands.
type Runner struct {
	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *Config

	// client is the consul/api client.
	client *api.Client

	// dry signals that output should be sent to stdout instead of committed to
	// disk. once indicates the runner should execute each template exactly one
	// time and then stop.
	dry, once bool

	// minTimer and maxTimer are used for quiescence.
	minTimer, maxTimer <-chan time.Time

	// outStream and errStream are the io.Writer streams where the runner will
	// write information. These streams can be set using the SetOutStream()
	// and SetErrStream() functions.
	outStream, errStream io.Writer

	// ctemplatesMap is a map of each template to the ConfigTemplates
	// that made it.
	ctemplatesMap map[string][]*ConfigTemplate

	// templates is the list of calculated templates.
	templates []*Template

	// dependencies is the list of dependencies this runner is watching.
	dependencies []dep.Dependency

	// watcher is the watcher this runner is using.
	watcher *watch.Watcher

	// brain is the internal storage database of returned dependency data.
	brain *Brain

	// stopCh is the channel the runner listens on for stopping.
	stopCh chan struct{}
}

// NewRunner accepts a slice of ConfigTemplates and returns a pointer to the new
// Runner and any error that occurred during creation.
func NewRunner(config *Config, dry, once bool) (*Runner, error) {
	runner := &Runner{
		config: config,
		dry:    dry,
		once:   once,
	}

	if err := runner.init(); err != nil {
		return nil, err
	}

	return runner, nil
}

// Start begins the polling for this runner. Any errors that occur will cause
// this function to push an item onto the given error channel and the halt
// execution. This function is blocking and should be called as a goroutine.
func (r *Runner) Start(errCh chan<- error) {
	// Create the stopCh
	r.stopCh = make(chan struct{})

	// Fire an initial run to parse all the templates and setup the first-pass
	// dependencies. This also forces any templates that have no dependencies to
	// be rendered immediately (since they are already renderable).
	if err := r.Run(); err != nil {
		errCh <- err
		return
	}

	for {
		select {
		case data := <-r.watcher.DataCh:
			r.Receive(data.Dependency, data.Data)

			// If we are waiting for quiescence, setup the timers
			if r.config.Wait != nil {
				log.Printf("[DEBUG] (runner) starting quiescence timers")

				// Reset the minTimer and set the maxTimer if it does not already exist
				r.minTimer = time.After(r.config.Wait.Min)
				if r.maxTimer == nil {
					r.maxTimer = time.After(r.config.Wait.Max)
				}

				// Since we are in quiescence mode, do not attempt to render and start
				// the loop from the top again (thus avoiding a pre-mature render).
				continue
			}
		case err := <-r.watcher.ErrCh:
			errCh <- err
			return
		case <-r.minTimer:
			log.Printf("[INFO] (runner) quiescence minTimer fired")
			r.minTimer, r.maxTimer = nil, nil
		case <-r.maxTimer:
			log.Printf("[INFO] (runner) quiescence maxTimer fired")
			r.minTimer, r.maxTimer = nil, nil
		case err := <-r.watcher.ErrCh:
			errCh <- err
			return
		case <-r.watcher.FinishCh:
			log.Printf("[INFO] (runner) watcher reported finish")
			return
		case <-r.stopCh:
			log.Printf("[INFO] (runner) received finish")
			return
		}

		// If we got this far, that means we got new data or one of the timers fired,
		// so attempt to re-render.
		if err := r.Run(); err != nil {
			errCh <- err
			return
		}
	}
}

// Stop halts the execution of this runner and its subprocesses.
func (r *Runner) Stop() {
	r.watcher.Stop()
	close(r.stopCh)
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

// Receive accepts a Dependency and data for that dep. This data is
// cached on the Runner. This data is then used to determine if a Template
// is "renderable" (i.e. all its Dependencies have been downloaded at least
// once).
func (r *Runner) Receive(d dep.Dependency, data interface{}) {
	log.Printf("[DEBUG] (runner) receiving dependency %s", d.Display())
	r.brain.Remember(d, data)
}

// Run iterates over each template in this Runner and conditionally executes
// the template rendering and command execution.
//
// The template is rendered atomicly. If and only if the template render
// completes successfully, the optional commands will be executed, if given.
// Please note that all templates are rendered **and then** any commands are
// executed.
func (r *Runner) Run() error {
	log.Printf("[INFO] (runner) running")

	var commands []string
	depsMap := make(map[string]dep.Dependency)

	for _, tmpl := range r.templates {
		log.Printf("[DEBUG] (runner) checking template: %#v", tmpl)

		missing, contents, err := tmpl.Execute(r.brain)
		if err != nil {
			return err
		}

		// Add the dependency to the list of dependencies for this runner.
		for _, d := range tmpl.Dependencies() {
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}
		}

		// If there are missing dependencies, start the watcher and move onto the
		// next one.
		if len(missing) > 0 {
			log.Printf("[INFO] (runner) was missing %d dependencies", len(missing))
			for _, dep := range missing {
				r.watcher.Add(dep)
			}
			continue
		}

		// If the template is missing data for some dependencies then we are not
		// ready to render and need to move on to the next one.
		if !r.canRender(tmpl) {
			log.Printf("[DEBUG] (runner) cannot render (some dependencies do not have data yet)")
			continue
		}

		for _, ctemplate := range r.configTemplatesFor(tmpl) {
			log.Printf("[DEBUG] (runner) checking ctemplate: %v", ctemplate.Command)

			// Render the template, taking dry mode into account
			rendered, err := r.render(contents, ctemplate.Destination)
			if err != nil {
				log.Printf("[DEBUG] (runner) error rendering template: `%s`", tmpl.Path)
				return err
			}

			log.Printf("[DEBUG] (runner) rendered: %t", rendered)

			// If the template was rendered (changed) and we are not in dry-run mode,
			// aggregate commands, ignoring previously known commands
			if rendered && !r.dry {
				if ctemplate.Command != "" && !exists(ctemplate.Command, commands) {
					log.Printf("[DEBUG] (runner) appending command: `%s`", ctemplate.Command)
					commands = append(commands, ctemplate.Command)
				}
			}
		}
	}

	// Perform the diff and update the known dependencies.
	r.diffAndUpdateDeps(depsMap)

	// Execute each command in sequence, collecting any errors that occur - this
	// ensures all commands execute at least once.
	var errs []error
	for _, command := range commands {
		log.Printf("[DEBUG] (runner) running command: `%s`", command)
		if err := r.execute(command); err != nil {
			log.Printf("[ERR] (runner) error running command: %s", err)
			errs = append(errs, err)
		}
	}

	// If any errors were returned, convert them to an ErrorList for human
	// readability.
	if len(errs) != 0 {
		var result *multierror.Error
		for _, err := range errs {
			result = multierror.Append(result, err)
		}
		return result.ErrorOrNil()
	}

	return nil
}

// init() creates the Runner's underlying data structures and returns an error
// if any problems occur.
func (r *Runner) init() error {
	// Merge multiple configs if given
	if r.config.Path != "" {
		err := buildConfig(r.config, r.config.Path)
		if err != nil {
			return fmt.Errorf("runner: %s", err)
		}
	}

	// Create the client
	client, err := newAPIClient(r.config)
	if err != nil {
		return fmt.Errorf("runner: %s", err)
	}
	r.client = client

	// Create the watcher
	watcher, err := newWatcher(r.config, client, r.once)
	if err != nil {
		return fmt.Errorf("runner: %s", err)
	}
	r.watcher = watcher

	templatesMap := make(map[string]*Template)
	ctemplatesMap := make(map[string][]*ConfigTemplate)

	// Iterate over each ConfigTemplate, creating a new Template resource for each
	// entry. Templates are parsed and saved, and a map of templates to their
	// config templates is kept so templates can lookup their commands and output
	// destinations.
	for _, ctmpl := range r.config.ConfigTemplates {
		tmpl, err := NewTemplate(ctmpl.Source)
		if err != nil {
			return err
		}

		if _, ok := templatesMap[tmpl.Path]; !ok {
			templatesMap[tmpl.Path] = tmpl
		}

		if _, ok := ctemplatesMap[tmpl.Path]; !ok {
			ctemplatesMap[tmpl.Path] = make([]*ConfigTemplate, 0, 1)
		}
		ctemplatesMap[tmpl.Path] = append(ctemplatesMap[tmpl.Path], ctmpl)
	}

	// Convert the map of templates (which was only used to ensure uniqueness)
	// back into an array of templates.
	templates := make([]*Template, 0, len(templatesMap))
	for _, tmpl := range templatesMap {
		templates = append(templates, tmpl)
	}
	r.templates = templates

	r.dependencies = make([]dep.Dependency, 0)

	r.ctemplatesMap = ctemplatesMap
	r.outStream = os.Stdout
	r.brain = NewBrain()

	return nil
}

// diffAndUpdateDeps iterates through the current map of dependencies on this
// runner and stops the watcher for any deps that are no longer required.
//
// At the end of this function, the given depsMap is converted to a slice and
// stored on the runner.
func (r *Runner) diffAndUpdateDeps(depsMap map[string]dep.Dependency) {
	// Diff and up the list of dependencies, stopping any unneeded watchers.
	log.Printf("[INFO] (runner) updating dependencies")
	for _, d := range r.dependencies {
		log.Printf("[DEBUG] (runner) checking if %s still needed", d.Display())
		if _, ok := depsMap[d.HashCode()]; !ok {
			log.Printf("[DEBUG] (runner) %s is no longer needed", d.Display())
			r.watcher.Remove(d)
			r.brain.Forget(d)
		} else {
			log.Printf("[DEBUG] (runner) %s is still needed", d.Display())
		}
	}

	deps := make([]dep.Dependency, 0, len(depsMap))
	for _, d := range depsMap {
		deps = append(deps, d)
	}
	r.dependencies = deps
}

// ConfigTemplateFor returns the ConfigTemplate for the given Template
func (r *Runner) configTemplatesFor(tmpl *Template) []*ConfigTemplate {
	return r.ctemplatesMap[tmpl.Path]
}

// canRender accepts a template and returns true if and only if all of the
// dependencies of that template have received data. This function assumes the
// template has been completely compiled and all required dependencies exist
// on the template.
func (r *Runner) canRender(tmpl *Template) bool {
	for _, d := range tmpl.Dependencies() {
		if !r.brain.Remembered(d) {
			log.Printf("[DEBUG] (runner) %q missing data for %s", tmpl.Path, d.Display())
			return false
		}
	}
	return true
}

// Render accepts a Template and a destinations.
//
// If the template has changed on disk, this method return true.
//
// If the template already exists and has the same contents as the "would-be"
// template, no action is taken. In this scenario, the render function returns
// false, indicating no template change has occurred.
func (r *Runner) render(contents []byte, dest string) (bool, error) {
	existingContents, err := ioutil.ReadFile(dest)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(contents, existingContents) {
		return false, nil
	}

	if r.dry {
		fmt.Fprintf(r.outStream, "> %s\n%s", dest, contents)
	} else {
		if err := atomicWrite(dest, contents); err != nil {
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
func atomicWrite(path string, contents []byte) error {
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

// Checks if a value exists in an array
func exists(needle string, haystack []string) bool {
	needle = strings.TrimSpace(needle)
	for _, value := range haystack {
		if needle == strings.TrimSpace(value) {
			return true
		}
	}

	return false
}

// newAPIClient creates a new API client from the given config and
func newAPIClient(config *Config) (*api.Client, error) {
	log.Printf("[INFO] (runner) creating consul/api client")

	consulConfig := api.DefaultConfig()

	if config.Consul != "" {
		log.Printf("[DEBUG] (runner) setting address to %s", config.Consul)
		consulConfig.Address = config.Consul
	}

	if config.Token != "" {
		log.Printf("[DEBUG] (runner) setting token to %s", config.Token)
		consulConfig.Token = config.Token
	}

	if config.SSL {
		log.Printf("[DEBUG] (runner) enabling SSL")
		consulConfig.Scheme = "https"
	}

	if config.SSLNoVerify {
		log.Printf("[WARN] (runner) disabling SSL verification")
		consulConfig.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// TODO: Basic auth

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// newWatcher creates a new watcher.
func newWatcher(config *Config, client *api.Client, once bool) (*watch.Watcher, error) {
	log.Printf("[INFO] (runner) creating Watcher")

	watcher, err := watch.NewWatcher(client, once)
	if err != nil {
		return nil, err
	}

	if config.Retry != 0 {
		watcher.SetRetry(config.Retry)
	}

	return watcher, err
}

// buildConfig iterates and merges all configuration files in a given directory.
// The config parameter will be modified and merged with subsequent configs
// found in the directory.
func buildConfig(config *Config, path string) error {
	log.Printf("[DEBUG] merging with config at %s", path)

	// Ensure the given filepath exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config: missing file/folder: %s", path)
	}

	// Check if a file was given or a path to a directory
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("config: error stating file: %s", err)
	}

	// Recursively parse directories, single load files
	if stat.Mode().IsDir() {
		// Ensure the given filepath has at least one config file
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return fmt.Errorf("config: error listing directory: %s", err)
		}
		if len(files) == 0 {
			return fmt.Errorf("config: must contain at least one configuration file")
		}

		// Potential bug: Walk does not follow symlinks!
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			// If WalkFunc had an error, just return it
			if err != nil {
				return err
			}

			// Do nothing for directories
			if info.IsDir() {
				return nil
			}

			// Parse and merge the config
			newConfig, err := ParseConfig(path)
			if err != nil {
				return err
			}
			config.Merge(newConfig)

			return nil
		})

		if err != nil {
			return fmt.Errorf("config: walk error: %s", err)
		}
	} else if stat.Mode().IsRegular() {
		newConfig, err := ParseConfig(path)
		if err != nil {
			return err
		}
		config.Merge(newConfig)
	} else {
		return fmt.Errorf("config: unknown filetype: %s", stat.Mode().String())
	}

	return nil
}
