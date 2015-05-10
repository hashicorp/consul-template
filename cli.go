package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hashicorp/consul-template/logging"
	"github.com/hashicorp/consul-template/watch"
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
	ExitCodeLoggingError
	ExitCodeParseFlagsError
	ExitCodeRunnerError
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

	// Setup the logging
	if err := logging.Setup(&logging.Config{
		Name:           Name,
		Level:          config.LogLevel,
		Syslog:         config.Syslog.Enabled == BoolTrue,
		SyslogFacility: config.Syslog.Facility,
		Writer:         cli.errStream,
	}); err != nil {
		return cli.handleError(err, ExitCodeLoggingError)
	}

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if version {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s v%s\n", Name, Version)
		return ExitCodeOK
	}

	// Initial runner
	runner, err := NewRunner(config, dry, once)
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
				runner, err = NewRunner(config, dry, once)
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
	var config = DefaultConfig()

	// Unset default boolean values - since CLI takes precedence during a merge,
	// we must reset these to the unset values
	config.SSL.Enabled = BoolUnset
	config.SSL.Verify = BoolUnset
	config.Vault.SSL.Enabled = BoolUnset
	config.Vault.SSL.Verify = BoolUnset

	var sslEnabled, sslVerify, syslogEnabled bool

	// Parse the flags and options
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprintf(cli.errStream, usage, Name)
	}
	flags.StringVar(&config.Consul, "consul", config.Consul, "")
	flags.StringVar(&config.Token, "token", config.Token, "")
	flags.Var((*authConfigVar)(config.Auth), "auth", "")
	flags.BoolVar(&sslEnabled, "ssl", false, "")
	flags.BoolVar(&sslVerify, "ssl-verify", true, "")
	flags.StringVar(&config.SSL.Cert, "ssl-cert", config.SSL.Cert, "")
	flags.StringVar(&config.SSL.CaCert, "ssl-ca-cert", config.SSL.CaCert, "")
	flags.DurationVar(&config.MaxStale, "max-stale", config.MaxStale, "")
	flags.Var((*configTemplateVar)(&config.ConfigTemplates), "template", "")
	flags.BoolVar(&syslogEnabled, "syslog", false, "")
	flags.StringVar(&config.Syslog.Facility, "syslog-facility", config.Syslog.Facility, "")
	flags.Var((*watch.WaitVar)(config.Wait), "wait", "")
	flags.DurationVar(&config.Retry, "retry", config.Retry, "")
	flags.StringVar(&config.Path, "config", config.Path, "")
	flags.StringVar(&config.LogLevel, "log-level", config.LogLevel, "")
	flags.BoolVar(&once, "once", false, "")
	flags.BoolVar(&dry, "dry", false, "")
	flags.BoolVar(&version, "v", false, "")
	flags.BoolVar(&version, "version", false, "")

	// Deprecated options
	var deprecatedSSLNoVerify bool
	flags.BoolVar(&deprecatedSSLNoVerify, "ssl-no-verify", false, "")

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

	// Handle Go's terrible default boolean values
	if sslEnabled {
		config.SSL.Enabled = BoolTrue
	}

	if !sslVerify {
		config.SSL.Verify = BoolFalse
	}

	if syslogEnabled {
		config.Syslog.Enabled = BoolTrue
	}

	// Handle deprecations
	if deprecatedSSLNoVerify {
		log.Printf("[WARN] -ssl-no-verify is deprecated - please use " +
			"-ssl-verify=false instead")
		config.SSL.Verify = BoolFalse
	}

	return config, once, dry, version, nil
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (cli *CLI) handleError(err error, status int) int {
	fmt.Fprintf(cli.errStream, "Consul Template returned errors:\n%s", err)
	return status
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
  -ssl                     Use SSL when connecting to Consul
  -ssl-verify              Verify certificates when connecting via SSL
  -ssl-cert                SSL client certificate to send to server
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

  -log-level=<level>       Set the logging level - valid values are "debug",
                           "info", "warn" (default), and "err"

  -dry                     Dump generated templates to stdout
  -once                    Do not run the process as a daemon
  -v, -version             Print the version of this daemon
`
