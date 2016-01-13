package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/consul-template/logging"
	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/go-reap"
)

// Exit codes are int values that represent an exit code for a particular error.
// Sub-systems may check this unique error to determine the cause of an error
// without parsing the output or help text.
//
// Errors start at 10
const (
	ExitCodeOK int = 0

	ExitCodeError = 10 + iota
	ExitCodeInterrupt
	ExitCodeParseFlagsError
	ExitCodeRunnerError
	ExitCodeConfigError
)

/// ------------------------- ///

// CLI is the main entry point for Consul Template.
type CLI struct {
	sync.Mutex

	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer

	// stopCh is an internal channel used to trigger a shutdown of the CLI.
	stopCh  chan struct{}
	stopped bool
}

func NewCLI(out, err io.Writer) *CLI {
	return &CLI{
		outStream: out,
		errStream: err,
		stopCh:    make(chan struct{}),
	}
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func (cli *CLI) Run(args []string) int {
	// Parse the flags
	config, once, dry, version, err := cli.parseFlags(args[1:])
	if err != nil {
		return cli.handleError(err, ExitCodeParseFlagsError)
	}

	// Save original config (defaults + parsed flags) for handling reloads
	base_config := config

	// Setup the config and logging
	config, err = cli.setup(config)
	if err != nil {
		return cli.handleError(err, ExitCodeConfigError)
	}

	// Print version information for debugging
	log.Printf("[INFO] %s", formattedVersion())

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if version {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s\n", formattedVersion())
		return ExitCodeOK
	}

	// If they configured a child process reaper, start that now
	var reapLock sync.RWMutex
	if config.Reap || (!config.WasSet("reap") && os.Getpid() == 1) {
		if !reap.IsSupported() {
			err := fmt.Errorf("[WARN] Child process reaping is not supported on this platform, please set the 'reap' option to false")
			return cli.handleError(err, ExitCodeConfigError)
		}
		log.Printf("[DEBUG] Automatically reaping child processes")

		pids := make(reap.PidCh, 1)
		errors := make(reap.ErrorCh, 1)
		go func() {
			for {
				select {
				case pid := <-pids:
					log.Printf("[DEBUG] Reaped child process %d", pid)
				case err := <-errors:
					log.Printf("[ERR] Error reaping child process: %v", err)
				case <-cli.stopCh:
					return
				}
			}
		}()
		go reap.ReapChildren(pids, errors, cli.stopCh, &reapLock)
	}

	// Initial runner
	runner, err := NewRunner(config, dry, once, &reapLock)
	if err != nil {
		return cli.handleError(err, ExitCodeRunnerError)
	}
	go runner.Start()

	// Listen for signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	for {
		select {
		case err := <-runner.ErrCh:
			return cli.handleError(err, ExitCodeRunnerError)
		case <-runner.DoneCh:
			return ExitCodeOK
		case s := <-signalCh:
			switch s {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				fmt.Fprintf(cli.errStream, "Received interrupt, cleaning up...\n")
				runner.Stop()
				return ExitCodeInterrupt
			case syscall.SIGHUP:
				fmt.Fprintf(cli.errStream, "Received HUP, reloading configuration...\n")
				runner.Stop()

				// Load the new configuration from disk
				config, err = cli.setup(base_config)
				if err != nil {
					return cli.handleError(err, ExitCodeConfigError)
				}

				runner, err = NewRunner(config, dry, once, &reapLock)
				if err != nil {
					return cli.handleError(err, ExitCodeRunnerError)
				}
				go runner.Start()
			}
		case <-cli.stopCh:
			return ExitCodeOK
		}
	}
}

// stop is used internally to shutdown a running CLI
func (cli *CLI) stop() {
	cli.Lock()
	defer cli.Unlock()

	if cli.stopped {
		return
	}

	close(cli.stopCh)
	cli.stopped = true
}

// parseFlags is a helper function for parsing command line flags using Go's
// Flag library. This is extracted into a helper to keep the main function
// small, but it also makes writing tests for parsing command line arguments
// much easier and cleaner.
func (cli *CLI) parseFlags(args []string) (*Config, bool, bool, bool, error) {
	var dry, once, version bool
	config := DefaultConfig()

	// Parse the flags and options
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() { fmt.Fprintf(cli.errStream, usage, Name) }

	flags.Var((funcVar)(func(s string) error {
		config.Consul = s
		config.set("consul")
		return nil
	}), "consul", "")

	flags.Var((funcVar)(func(s string) error {
		config.Token = s
		config.set("token")
		return nil
	}), "token", "")

	flags.Var((funcVar)(func(s string) error {
		config.Auth.Enabled = true
		config.set("auth.enabled")
		if strings.Contains(s, ":") {
			split := strings.SplitN(s, ":", 2)
			config.Auth.Username = split[0]
			config.set("auth.username")
			config.Auth.Password = split[1]
			config.set("auth.password")
		} else {
			config.Auth.Username = s
			config.set("auth.username")
		}
		return nil
	}), "auth", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		config.SSL.Enabled = b
		config.set("ssl")
		config.set("ssl.enabled")
		return nil
	}), "ssl", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		config.SSL.Verify = b
		config.set("ssl")
		config.set("ssl.verify")
		return nil
	}), "ssl-verify", "")

	flags.Var((funcVar)(func(s string) error {
		config.SSL.Cert = s
		config.set("ssl")
		config.set("ssl.cert")
		return nil
	}), "ssl-cert", "")

	flags.Var((funcVar)(func(s string) error {
		config.SSL.Key = s
		config.set("ssl")
		config.set("ssl.key")
		return nil
	}), "ssl-key", "")

	flags.Var((funcVar)(func(s string) error {
		config.SSL.CaCert = s
		config.set("ssl")
		config.set("ssl.ca_cert")
		return nil
	}), "ssl-ca-cert", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		config.MaxStale = d
		config.set("max_stale")
		return nil
	}), "max-stale", "")

	flags.Var((funcVar)(func(s string) error {
		t, err := ParseConfigTemplate(s)
		if err != nil {
			return err
		}
		if config.ConfigTemplates == nil {
			config.ConfigTemplates = make([]*ConfigTemplate, 0, 1)
		}
		config.ConfigTemplates = append(config.ConfigTemplates, t)
		return nil
	}), "template", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		config.Syslog.Enabled = b
		config.set("syslog")
		config.set("syslog.enabled")
		return nil
	}), "syslog", "")

	flags.Var((funcVar)(func(s string) error {
		config.Syslog.Facility = s
		config.set("syslog.facility")
		return nil
	}), "syslog-facility", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		config.Deduplicate.Enabled = b
		config.set("deduplicate")
		config.set("deduplicate.enabled")
		return nil
	}), "dedup", "")

	flags.Var((funcVar)(func(s string) error {
		w, err := watch.ParseWait(s)
		if err != nil {
			return err
		}
		config.Wait.Min = w.Min
		config.Wait.Max = w.Max
		config.set("wait")
		return nil
	}), "wait", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		config.Retry = d
		config.set("retry")
		return nil
	}), "retry", "")

	flags.Var((funcVar)(func(s string) error {
		config.Path = s
		config.set("path")
		return nil
	}), "config", "")

	flags.Var((funcVar)(func(s string) error {
		config.PidFile = s
		config.set("pid_file")
		return nil
	}), "pid-file", "")

	flags.Var((funcVar)(func(s string) error {
		config.LogLevel = s
		config.set("log_level")
		return nil
	}), "log-level", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		config.Reap = b
		config.set("reap")
		return nil
	}), "reap", "")

	flags.BoolVar(&once, "once", false, "")
	flags.BoolVar(&dry, "dry", false, "")
	flags.BoolVar(&version, "v", false, "")
	flags.BoolVar(&version, "version", false, "")

	// If there was a parser error, stop
	if err := flags.Parse(args); err != nil {
		return nil, false, false, false, err
	}

	// Error if extra arguments are present
	args = flags.Args()
	if len(args) > 0 {
		return nil, false, false, false, fmt.Errorf("cli: extra argument(s): %q",
			args)
	}

	return config, once, dry, version, nil
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (cli *CLI) handleError(err error, status int) int {
	fmt.Fprintf(cli.errStream, "Consul Template returned errors:\n%s", err)
	return status
}

func (cli *CLI) setup(config *Config) (*Config, error) {
	if config.Path != "" {
		newConfig, err := ConfigFromPath(config.Path)
		if err != nil {
			return nil, err
		}

		// Merge ensuring that the CLI options still take precedence
		newConfig.Merge(config)
		config = newConfig
	}

	// Setup the logging
	if err := logging.Setup(&logging.Config{
		Name:           Name,
		Level:          config.LogLevel,
		Syslog:         config.Syslog.Enabled,
		SyslogFacility: config.Syslog.Facility,
		Writer:         cli.errStream,
	}); err != nil {
		return nil, err
	}

	return config, nil
}

const usage = `
Usage: %s [options]

  Watches a series of templates on the file system, writing new changes when
  Consul is updated. It runs until an interrupt is received unless the -once
  flag is specified.

Options:

  -auth=<user[:pass]>      Set the basic authentication username (and password)
  -consul=<address>        Sets the address of the Consul instance
  -max-stale=<duration>    Set the maximum staleness and allow stale queries to
                           Consul which will distribute work among all servers
                           instead of just the leader
  -dedup                   Enable de-duplication mode. Reduces load on Consul when
                           many instances of Consul Template are rendering a common
                           template.
  -ssl                     Use SSL when connecting to Consul
  -ssl-verify              Verify certificates when connecting via SSL
  -ssl-cert                SSL client certificate to send to server
  -ssl-key                 SSL/TLS private key for use in client authentication
                           key exchange
  -ssl-ca-cert             Validate server certificate against this CA
                           certificate file list
  -token=<token>           Sets the Consul API token

  -syslog                  Send the output to syslog instead of standard error
                           and standard out. The syslog facility defaults to
                           LOCAL0 and can be changed using a configuration file
  -syslog-facility=<f>     Set the facility where syslog should log. If this
                           attribute is supplied, the -syslog flag must also be
                           supplied.

  -template=<template>     Adds a new template to watch on disk in the format
                           'templatePath:outputPath(:command)'
  -wait=<duration>         Sets the 'minumum(:maximum)' amount of time to wait
                           before writing a template (and triggering a command)
  -retry=<duration>        The amount of time to wait if Consul returns an
                           error when communicating with the API

  -config=<path>           Sets the path to a configuration file on disk


  -pid-file=<path>         Path on disk to write the PID of the process
  -log-level=<level>       Set the logging level - valid values are "debug",
                           "info", "warn" (default), and "err"

  -dry                     Dump generated templates to stdout
  -once                    Do not run the process as a daemon
  -reap                    Control automatic reaping of child processes, useful
                           if running as PID 1 in a Docker container. By default,
                           if Consul Template detects that it is running as PID 1
                           it will automatically enable child process reaping.
                           Setting this option to false disables this behavior,
                           and setting it to true enables child process reaping
                           regardless of Consul Template's PID.
  -v, -version             Print the version of this daemon
`
