package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/logutils"
)

/// ------------------------- ///

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
	ExitCodeParseWaitError
	ExitCodeRunnerError
)

/// ------------------------- ///

// CLI is the main entry point for Consul Template.
type CLI struct {
	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer

	// stopCh is an internal channel used to trigger a shutdown of the CLI.
	stopCh chan struct{}
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
	cli.initLogger()

	var version, dry, once bool
	var auth string
	var config = new(Config)

	// Parse the flags and options
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprintf(cli.errStream, usage, Name)
	}
	flags.StringVar(&config.Consul, "consul", "",
		"address of the Consul instance")
	flags.BoolVar(&config.SSL, "ssl", false,
		"use https while talking to consul")
	flags.BoolVar(&config.SSLNoVerify, "ssl-no-verify", false,
		"ignore certificate warnings under https")
	flags.StringVar(&auth, "auth", "",
		"set basic auth username[:password]")
	flags.DurationVar(&config.MaxStale, "max-stale", 0,
		"the maximum time to wait for stale queries")
	flags.Var((*configTemplateVar)(&config.ConfigTemplates), "template",
		"new template declaration")
	flags.StringVar(&config.Token, "token", "",
		"a consul API token")
	flags.IntVar(&config.BatchSize, "batch-size", 0,
		"the size of the batch of dependencies")
	flags.StringVar(&config.WaitRaw, "wait", "",
		"the minimum(:maximum) to wait before rendering a new template")
	flags.StringVar(&config.Path, "config", "",
		"the path to a config file on disk")
	flags.DurationVar(&config.Retry, "retry", 0,
		"the duration to wait when Consul is not available")
	flags.BoolVar(&once, "once", false,
		"do not run as a daemon")
	flags.BoolVar(&dry, "dry", false,
		"write generated templates to stdout")
	flags.BoolVar(&version, "version", false, "display the version")

	// If there was a parser error, stop
	if err := flags.Parse(args[1:]); err != nil {
		return cli.handleError(err, ExitCodeParseFlagsError)
	}

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if version {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s v%s\n", Name, Version)
		return ExitCodeOK
	}

	// Setup authentication
	if auth != "" {
		log.Printf("[DEBUG] (cli) detected -auth, parsing")
		config.Auth = new(Auth)
		if strings.Contains(auth, ":") {
			split := strings.SplitN(auth, ":", 2)
			config.Auth.Username = split[0]
			config.Auth.Password = split[1]
		} else {
			config.Auth.Username = auth
		}
	}

	// Parse the raw wait value into a Wait object
	if config.WaitRaw != "" {
		log.Printf("[DEBUG] (cli) detected -wait, parsing")
		wait, err := watch.ParseWait(config.WaitRaw)
		if err != nil {
			return cli.handleError(err, ExitCodeParseWaitError)
		}
		config.Wait = wait
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
	close(cli.stopCh)
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (cli *CLI) handleError(err error, status int) int {
	fmt.Fprintf(cli.errStream, "Consul Template returned errors:\n%s", err)
	return status
}

// initLogger gets the log level from the environment, falling back to DEBUG if
// nothing was given.
func (cli *CLI) initLogger() {
	minLevel := strings.ToUpper(strings.TrimSpace(os.Getenv("CONSUL_TEMPLATE_LOG")))
	if minLevel == "" {
		minLevel = "WARN"
	}

	levelFilter := &logutils.LevelFilter{
		Levels: []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERR"},
		Writer: cli.errStream,
	}

	levelFilter.SetMinLevel(logutils.LogLevel(minLevel))

	log.SetOutput(levelFilter)
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
  -ssl-no-verify           Ignore certificate warnings when connecting via SSL
  -token=<token>           Sets the Consul API token

  -template=<template>     Adds a new template to watch on disk in the format
                           'templatePath:outputPath(:command)'
  -batch-size=<size>       Set the size of the batch when polling multiple
                           dependencies
  -wait=<duration>         Sets the 'minumum(:maximum)' amount of time to wait
                           before writing a template (and triggering a command)
  -retry=<duration>        The amount of time to wait if Consul returns an
                           error when communicating with the API

  -config=<path>           Sets the path to a configuration file on disk

  -dry                     Dump generated templates to stdout
  -once                    Do not run the process as a daemon
  -version                 Print the version of this daemon
`
