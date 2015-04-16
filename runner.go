package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-multierror"
)

const (
	// saneViewLimit is the number of views that we consider "sane" before we
	// warn the user that they might be DDoSing their Consul cluster.
	saneViewLimit = 128
)

// Runner responsible rendering Templates and invoking Commands.
type Runner struct {
	// ErrCh and DoneCh are channels where errors and finish notifications occur.
	ErrCh  chan error
	DoneCh chan struct{}

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *Config

	// client is the consul/api client.
	client *api.Client

	// dry signals that output should be sent to stdout instead of committed to
	// disk. once indicates the runner should execute each template exactly one
	// time and then stop.
	dry, once bool

	// outStream and errStream are the io.Writer streams where the runner will
	// write information. These streams can be set using the SetOutStream()
	// and SetErrStream() functions.
	outStream, errStream io.Writer

	// ctemplatesMap is a map of each template to the ConfigTemplates
	// that made it.
	ctemplatesMap map[string][]*ConfigTemplate

	// templates is the list of calculated templates.
	templates []*Template

	// renderedTemplates is a map of templates we have successfully rendered to
	// disk. It is used for once mode and internal tracking. The key is the Path
	// of the template.
	renderedTemplates map[string]struct{}

	// dependencies is the list of dependencies this runner is watching.
	dependencies map[string]dep.Dependency

	// watcher is the watcher this runner is using.
	watcher *watch.Watcher

	// brain is the internal storage database of returned dependency data.
	brain *Brain

	// quiescenceMap is the map of templates to their quiescence timers.
	// quiescenceCh is the channel where templates report returns from quiescence
	// fires.
	quiescenceMap map[string]*quiescence
	quiescenceCh  chan *Template
}

// NewRunner accepts a slice of ConfigTemplates and returns a pointer to the new
// Runner and any error that occurred during creation.
func NewRunner(config *Config, dry, once bool) (*Runner, error) {
	log.Printf("[INFO] (runner) creating new runner (dry: %v, once: %v)", dry, once)

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
// this function to push an item onto the runner's error channel and the halt
// execution. This function is blocking and should be called as a goroutine.
func (r *Runner) Start() {
	log.Printf("[INFO] (runner) starting")

	// Fire an initial run to parse all the templates and setup the first-pass
	// dependencies. This also forces any templates that have no dependencies to
	// be rendered immediately (since they are already renderable).
	log.Printf("[DEBUG] (runner) running initial templates")
	if err := r.Run(); err != nil {
		r.ErrCh <- err
		return
	}

	for {
		// Enable quiescence for all templates if we have specified wait intervals.
		if r.config.Wait.Min != 0 && r.config.Wait.Max != 0 {
			for _, t := range r.templates {
				if _, ok := r.quiescenceMap[t.Path]; !ok {
					log.Printf("[DEBUG] (runner) enabling quiescence for %q", t.Path)
					r.quiescenceMap[t.Path] = newQuiescence(
						r.quiescenceCh, r.config.Wait.Min, r.config.Wait.Max, t)
				}
			}
		}

		// Warn the user if they are watching too many dependencies.
		if r.watcher.Size() > saneViewLimit {
			log.Printf("[WARN] (runner) watching %d dependencies - watching this "+
				"many dependencies could DDoS your consul cluster", r.watcher.Size())
		} else {
			log.Printf("[INFO] (runner) watching %d dependencies", r.watcher.Size())
		}

		// If we are running in once mode and all our templates have been rendered,
		// then we should exit here.
		if r.once && r.allTemplatesRendered() {
			log.Printf("[INFO] (runner) once mode and all templates rendered")
			r.Stop()
			return
		}

	OUTER:
		select {
		case data := <-r.watcher.DataCh:
			// Receive this update
			r.Receive(data.Dependency, data.Data)

			// Drain all dependency data. Given a large number of dependencies, it is
			// feasible that we have data for more than one of them. Instead of
			// wasting CPU cycles rendering templates when we have more dependencies
			// waiting to be added to the brain, we drain the entire buffered channel
			// on the watcher and then reports when it is done receiving new data
			// which the parent select listens for.
			//
			// Please see https://github.com/hashicorp/consul-template/issues/168 for
			// more information about this optimization and the entire backstory.
			for {
				select {
				case data := <-r.watcher.DataCh:
					r.Receive(data.Dependency, data.Data)
				default:
					break OUTER
				}
			}
		case err := <-r.watcher.ErrCh:
			// Intentionally do not send the error back up to the runner. Eventually,
			// once Consul API implements errwrap and multierror, we can check the
			// "type" of error and conditionally alert back.
			//
			// if err.Contains(Something) {
			//   errCh <- err
			// }
			log.Printf("[ERR] (runner) watcher reported error: %s", err)
		case tmpl := <-r.quiescenceCh:
			// Remove the quiescence for this template from the map. This will force
			// the upcoming Run call to actually evaluate and render the template.
			log.Printf("[INFO] (runner) received template %q from quiescence", tmpl.Path)
			delete(r.quiescenceMap, tmpl.Path)
		case <-r.watcher.FinishCh:
			log.Printf("[INFO] (runner) watcher reported finish")
			return
		case <-r.DoneCh:
			log.Printf("[INFO] (runner) received finish")
			return
		}

		// If we got this far, that means we got new data or one of the timers fired,
		// so attempt to re-render.
		if err := r.Run(); err != nil {
			r.ErrCh <- err
			return
		}
	}
}

// Stop halts the execution of this runner and its subprocesses.
func (r *Runner) Stop() {
	log.Printf("[INFO] (runner) stopping")
	r.watcher.Stop()
	close(r.DoneCh)
}

// Receive accepts a Dependency and data for that dep. This data is
// cached on the Runner. This data is then used to determine if a Template
// is "renderable" (i.e. all its Dependencies have been downloaded at least
// once).
func (r *Runner) Receive(d dep.Dependency, data interface{}) {
	// Just because we received data, it does not mean that we are actually
	// watching for that data. How is that possible you may ask? Well, this
	// Runner's data channel is pooled, meaning it accepts multiple data views
	// before actually blocking. Whilest this runner is performing a Run() and
	// executing diffs, it may be possible that more data was pushed onto the
	// data channel pool for a dependency that we no longer care about.
	//
	// Accepting this dependency would introduce stale data into the brain, and
	// that is simply unacceptable. In fact, it is a fun little bug:
	//
	//     https://github.com/hashicorp/consul-template/issues/198
	//
	// and by "little" bug, I mean really big bug.
	if _, ok := r.dependencies[d.HashCode()]; ok {
		log.Printf("[DEBUG] (runner) receiving dependency %s", d.Display())
		r.brain.Remember(d, data)
	}
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
		log.Printf("[DEBUG] (runner) checking template %s", tmpl.Path)

		// If we are in once mode and this template was already rendered, move
		// onto the next one. We do not want to re-render the template if we are
		// in once mode, and we certainly do not want to re-run any commands.
		if r.once {
			if _, rendered := r.renderedTemplates[tmpl.Path]; rendered {
				log.Printf("[DEBUG] (runner) once mode and already rendered")
				continue
			}
		}

		// Attempt to render the template, returning any missing dependencies and
		// the rendered contents. If there are any missing dependencies, the
		// contents cannot be rendered or trusted!
		used, missing, contents, err := tmpl.Execute(r.brain)
		if err != nil {
			return err
		}

		// Add the dependency to the list of dependencies for this runner.
		for _, d := range used {
			if _, ok := depsMap[d.HashCode()]; !ok {
				depsMap[d.HashCode()] = d
			}
		}

		// Diff any missing dependencies the template reported with dependencies
		// the watcher is watching.
		var unwatched []dep.Dependency
		for _, d := range missing {
			if !r.watcher.Watching(d) {
				unwatched = append(unwatched, d)
			}
		}

		// If there are unwatched dependencies, start the watcher and move onto the
		// next one.
		if len(unwatched) > 0 {
			log.Printf("[INFO] (runner) was not watching %d dependencies", len(unwatched))
			for _, dep := range unwatched {
				r.watcher.Add(dep)
			}
			continue
		}

		// If the template is missing data for some dependencies then we are not
		// ready to render and need to move on to the next one.
		if len(missing) > 0 {
			log.Printf("[INFO] (runner) missing data for %d dependencies", len(missing))
			continue
		}

		// If quiescence is activated, start/update the timers and loop back around.
		// We do not want to render the templates yet.
		if q, ok := r.quiescenceMap[tmpl.Path]; ok {
			q.tick()
			continue
		}

		// For each configuration template that is tied to this template, attempt to
		// render it to disk and accumulate commands for later use.
		for _, ctemplate := range r.configTemplatesFor(tmpl) {
			log.Printf("[DEBUG] (runner) checking ctemplate %+v", ctemplate)

			// Render the template, taking dry mode into account
			wouldRender, didRender, err := r.render(contents, ctemplate.Destination)
			if err != nil {
				log.Printf("[DEBUG] (runner) error rendering %s", tmpl.Path)
				return err
			}

			log.Printf("[DEBUG] (runner) wouldRender: %t, didRender: %t", wouldRender, didRender)

			// If we would have rendered this template (but we did not because the
			// contents were the same or something), we should consider this template
			// rendered even though the contents on disk have not been updated. We
			// will not fire commands unless the template was _actually_ rendered to
			// disk though.
			if wouldRender {
				// Make a note that we have rendered this template (required for once
				// mode and just generally nice for debugging purposes).
				r.renderedTemplates[tmpl.Path] = struct{}{}
			}

			// If we _actually_ rendered the template to disk, we want to run the
			// appropriate commands.
			if didRender {
				if !r.dry {
					// If the template was rendered (changed) and we are not in dry-run mode,
					// aggregate commands, ignoring previously known commands
					//
					// Future-self Q&A: Why not use a map for the commands instead of an
					// array with an expensive lookup option? Well I'm glad you asked that
					// future-self! One of the API promises is that commands are executed
					// in the order in which they are provided in the ConfigTemplate
					// definitions. If we inserted commands into a map, we would lose that
					// relative ordering and people would be unhappy.
					if ctemplate.Command != "" && !exists(ctemplate.Command, commands) {
						log.Printf("[DEBUG] (runner) appending command: %s", ctemplate.Command)
						commands = append(commands, ctemplate.Command)
					}
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
	// Start with the default config
	config := DefaultConfig()

	// Merge multiple configs if given
	if r.config.Path != "" {
		err := buildConfig(config, r.config.Path)
		if err != nil {
			return fmt.Errorf("runner: %s", err)
		}
	}

	// Merge in default values for the config
	config.Merge(r.config)
	r.config = config

	// Print the final config for debugging
	result, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] runner: final config:\n\n%s\n\n", result)

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

	r.renderedTemplates = make(map[string]struct{})
	r.dependencies = make(map[string]dep.Dependency)

	r.ctemplatesMap = ctemplatesMap
	r.outStream = os.Stdout
	r.errStream = os.Stderr
	r.brain = NewBrain()

	r.ErrCh = make(chan error)
	r.DoneCh = make(chan struct{})

	r.quiescenceMap = make(map[string]*quiescence)
	r.quiescenceCh = make(chan *Template)

	return nil
}

// diffAndUpdateDeps iterates through the current map of dependencies on this
// runner and stops the watcher for any deps that are no longer required.
//
// At the end of this function, the given depsMap is converted to a slice and
// stored on the runner.
func (r *Runner) diffAndUpdateDeps(depsMap map[string]dep.Dependency) {
	// Diff and up the list of dependencies, stopping any unneeded watchers.
	log.Printf("[INFO] (runner) diffing and updating dependencies")

	for key, d := range r.dependencies {
		if _, ok := depsMap[key]; !ok {
			log.Printf("[DEBUG] (runner) %s is no longer needed", d.Display())
			r.watcher.Remove(d)
			r.brain.Forget(d)
		} else {
			log.Printf("[DEBUG] (runner) %s is still needed", d.Display())
		}
	}

	r.dependencies = depsMap
}

// ConfigTemplateFor returns the ConfigTemplate for the given Template
func (r *Runner) configTemplatesFor(tmpl *Template) []*ConfigTemplate {
	return r.ctemplatesMap[tmpl.Path]
}

// allTemplatesRendered returns true if all the templates in this Runner have
// been rendered at least one time.
func (r *Runner) allTemplatesRendered() bool {
	for _, t := range r.templates {
		if _, ok := r.renderedTemplates[t.Path]; !ok {
			return false
		}
	}
	return true
}

// Render accepts a Template and a destination on disk. The first return
// parameter is a boolean that indicates if the template would have been
// rendered. Since this function is idempotent (meaning it does not write the
// template if the contents are the same), it is possible that a template is
// renderable, but never actually rendered because the contents are already
// present on disk in the correct state. In this situation, we want to inform
// the parent that the template would have been rendered, but was not. The
// second return value indicates if the template was actually committed to disk.
// By the associative property, if the second return value is true, the first
// return value must also be true (but not necessarily the other direction). The
// second return value indicates whether the caller should take action given a
// template on disk has changed.
//
// No template exists on disk: true, true, nil
// Template exists, but contents are different: true, true, nil
// Template exists, but contents are the same: true, false, nil
func (r *Runner) render(contents []byte, dest string) (bool, bool, error) {
	existingContents, err := ioutil.ReadFile(dest)
	if err != nil && !os.IsNotExist(err) {
		return false, false, err
	}

	if bytes.Equal(contents, existingContents) {
		return true, false, nil
	}

	if r.dry {
		fmt.Fprintf(r.outStream, "> %s\n%s", dest, contents)
	} else {
		if err := atomicWrite(dest, contents); err != nil {
			return false, false, err
		}
	}

	return true, true, nil
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

	// Copy the current environment as well as some custom environment variables
	// that are read by other Consul tools (like Consul's HTTP address). This
	// allows the user to specify these values once (in the Consul Template config
	// or command line), instead of in multiple places.
	customEnv := map[string]string{
		"CONSUL_HTTP_ADDR":       r.config.Consul,
		"CONSUL_HTTP_TOKEN":      r.config.Token,
		"CONSUL_HTTP_AUTH":       r.config.Auth.String(),
		"CONSUL_HTTP_SSL":        strconv.FormatBool(r.config.SSL.Enabled),
		"CONSUL_HTTP_SSL_VERIFY": strconv.FormatBool(r.config.SSL.Verify),
	}
	currentEnv := os.Environ()
	cmdEnv := make([]string, len(currentEnv), len(currentEnv)+len(customEnv))
	copy(cmdEnv, currentEnv)
	for k, v := range customEnv {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}

	// Create an invoke the command
	cmd := exec.Command(shell, flag, command)
	cmd.Stdout = r.outStream
	cmd.Stderr = r.errStream
	cmd.Env = cmdEnv
	return cmd.Run()
}

// quiescence is an internal representation of a single template's quiescence
// state.
type quiescence struct {
	sync.Mutex

	template           *Template
	min                time.Duration
	minTimer, maxTimer <-chan time.Time
	ch                 chan *Template
	stopCh             chan struct{}
}

// newQuiescence creates a new quiescence timer for the given template.
func newQuiescence(ch chan *Template, min, max time.Duration, t *Template) *quiescence {
	return &quiescence{
		template: t,
		min:      min,
		minTimer: time.After(min),
		maxTimer: time.After(max),
		ch:       ch,
		stopCh:   make(chan struct{}),
	}
}

// start begins the quiescence timer for this quiescence.
func (q *quiescence) start() {
	select {
	case <-q.minTimer:
		log.Printf("[INFO] (runner) quiescence minTimer fired for %s", q.template.Path)
		q.minTimer, q.maxTimer = nil, nil
		q.ch <- q.template
	case <-q.maxTimer:
		log.Printf("[INFO] (runner) quiescence maxTimer fired for %s", q.template.Path)
		q.minTimer, q.maxTimer = nil, nil
		q.ch <- q.template
	case <-q.stopCh:
		return
	}
}

// tick updates the minimum quiescence timer.
func (q *quiescence) tick() {
	q.Lock()
	defer q.Unlock()

	// Stop an existing poll so we can reset the minTimer and restart.
	close(q.stopCh)
	q.stopCh = make(chan struct{})

	// Update the timer value and start a new poller
	q.minTimer = time.After(q.min)
	go q.start()
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

	if config.SSL.Enabled {
		log.Printf("[DEBUG] (runner) enabling SSL")
		consulConfig.Scheme = "https"
	}

	if !config.SSL.Verify {
		log.Printf("[WARN] (runner) disabling SSL verification")
		consulConfig.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	if config.Auth.Enabled {
		log.Printf("[DEBUG] (runner) setting basic auth")
		consulConfig.HttpAuth = &api.HttpBasicAuth{
			Username: config.Auth.Username,
			Password: config.Auth.Password,
		}
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// newWatcher creates a new watcher.
func newWatcher(config *Config, client *api.Client, once bool) (*watch.Watcher, error) {
	log.Printf("[INFO] (runner) creating Watcher")

	watcher, err := watch.NewWatcher(&watch.WatcherConfig{
		Client:   client,
		Once:     once,
		MaxStale: config.MaxStale,
		RetryFunc: func(current time.Duration) time.Duration {
			return config.Retry
		},
	})
	if err != nil {
		return nil, err
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
		_, err := ioutil.ReadDir(path)
		if err != nil {
			return fmt.Errorf("config: error listing directory: %s", err)
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
