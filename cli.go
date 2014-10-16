package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	api "github.com/armon/consul-api"
)

/// ------------------------- ///

// Exit codes are int valuse that represent an exit code for a particular error.
// Sub-systems may check this unique error to determine the cause of an error
// without parsing the output or help text.
const (
	ExitCodeOK int = 0

	// Errors start at 10
	ExitCodeError = 10 + iota
	ExitCodeParseFlagsError
	ExitCodeParseWaitError
	ExitCodeParseConfigError
	ExitCodeRunnerError
	ExitCodeConsulAPIError
	ExitCodeWatcherError
)

/// ------------------------- ///

type CLI struct {
	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func (cli *CLI) Run(args []string) int {
	var version, dry, once bool
	var config = new(Config)

	// Parse the flags and options
	cmd := filepath.Base(args[0])
	flags := flag.NewFlagSet("consul-template", flag.ContinueOnError)
	flags.Usage = func() { fmt.Fprint(cli.outStream, usage) }
	flags.SetOutput(cli.outStream)
	flags.StringVar(&config.Consul, "consul", "",
		"address of the Consul instance")
	flags.Var((*configTemplateVar)(&config.ConfigTemplates), "template",
		"new template declaration")
	flags.StringVar(&config.Token, "token", "",
		"a consul API token")
	flags.StringVar(&config.WaitRaw, "wait", "",
		"the minimum(:maximum) to wait before rendering a new template")
	flags.StringVar(&config.Path, "config", "",
		"the path to a config file on disk")
	flags.BoolVar(&once, "once", false,
		"do not run as a daemon")
	flags.BoolVar(&dry, "dry", false,
		"write generated templates to stdout")
	flags.BoolVar(&version, "version", false, "display the version")

	// If there was a parser error, stop
	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintf(cli.errStream, "%s\n\n%s", err.Error(), usage)
		return ExitCodeParseFlagsError
	}

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if version {
		fmt.Fprintf(cli.errStream, "%s v%s\n", cmd, Version)
		return ExitCodeOK
	}

	// Parse the raw wait value into a Wait object
	if config.WaitRaw != "" {
		wait, err := ParseWait(config.WaitRaw)
		if err != nil {
			fmt.Fprintf(cli.errStream, "%s\n\n%s", err.Error(), usage)
			return ExitCodeParseWaitError
		}
		config.Wait = wait
	}

	// Merge a path config with the command line options. Command line options
	// take precedence over config file options for easy overriding.
	if config.Path != "" {
		fileConfig, err := ParseConfig(config.Path)
		if err != nil {
			fmt.Fprintf(cli.errStream, "%s\n\n%s", err.Error(), usage)
			return ExitCodeParseConfigError
		}

		fileConfig.Merge(config)
		config = fileConfig
	}

	runner, err := NewRunner(config.ConfigTemplates)
	if err != nil {
		return cli.handleError(err, ExitCodeRunnerError)
	}

	consulConfig := api.DefaultConfig()
	if config.Consul != "" {
		consulConfig.Address = config.Consul
	}
	if config.Token != "" {
		consulConfig.Token = config.Token
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return cli.handleError(err, ExitCodeConsulAPIError)
	}
	if _, err := client.Agent().NodeName(); err != nil {
		return cli.handleError(err, ExitCodeConsulAPIError)
	}

	watcher, err := NewWatcher(client, runner.Dependencies())
	if err != nil {
		return cli.handleError(err, ExitCodeWatcherError)
	}

	if once {
		err = watcher.WatchOnce()
	} else {
		err = watcher.Watch()
		defer watcher.Stop()
	}
	if err != nil {
		return cli.handleError(err, ExitCodeWatcherError)
	}

	var minTimer, maxTimer <-chan time.Time

	for {
		select {
		case view := <-watcher.DataCh:
			runner.Receive(view.dependency, view.data)

			// If we are waiting for quiescence, setup the timers
			if config.Wait != nil {
				// Reset the min timer
				minTimer = time.After(config.Wait.Min)

				// Set the max timer if it does not already exist
				if maxTimer == nil {
					maxTimer = time.After(config.Wait.Max)
				}
			} else {
				if err := runner.RunAll(dry); err != nil {
					return cli.handleError(err, ExitCodeRunnerError)
				}
			}
		case <-minTimer:
			minTimer, maxTimer = nil, nil

			if err := runner.RunAll(dry); err != nil {
				return cli.handleError(err, ExitCodeRunnerError)
			}
		case <-maxTimer:
			minTimer, maxTimer = nil, nil

			if err := runner.RunAll(dry); err != nil {
				return cli.handleError(err, ExitCodeRunnerError)
			}
		case err := <-watcher.ErrCh:
			return cli.handleError(err, ExitCodeError)
		case <-watcher.stopCh:
			break
		default:
			continue
		}
	}

	return ExitCodeOK
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (cli *CLI) handleError(err error, status int) int {
	fmt.Fprintf(cli.errStream, err.Error())
	return status
}

const usage = `
Usage: %s [options]

  Watches a series of templates on the file system, writing new changes when
  Consul is updated. It runs until an interrupt is received unless the -once
  flag is specified.

Options:

  -consul=<address>        Sets the address of the Consul instance
  -token=<token>           Sets the Consul API token
  -template=<template>      Adds a new template to watch on disk in the format
                           'templatePath:outputPath(:command)'.
  -wait=<duration>         Sets the 'minumum(:maximum)' amount of time to wait
                           before writing a template (and triggering a command)
  -config=<path>           Sets the path to a configuration file on disk

  -dry                     Dump generated templates to stdout
  -once                    Do not run the process as a daemon
  -version                 Print the version of this daemon
`

/// ------------------------- ///

// configTemplateVar implements the Flag.Value interface and allows the user
// to specify multiple -template keys in the CLI where each option is parsed
// as a template.
type configTemplateVar []*ConfigTemplate

func (ctv configTemplateVar) String() string {
	buff := new(bytes.Buffer)
	for _, template := range ctv {
		fmt.Fprintf(buff, "%s", template.Source)
		if template.Destination != "" {
			fmt.Fprintf(buff, ":%s", template.Destination)

			if template.Command != "" {
				fmt.Fprintf(buff, ":%s", template.Command)
			}
		}
	}

	return buff.String()
}

func (ctv *configTemplateVar) Set(value string) error {
	template, err := ParseConfigTemplate(value)
	if err != nil {
		return err
	}

	if *ctv == nil {
		*ctv = make([]*ConfigTemplate, 0, 1)
	}
	*ctv = append(*ctv, template)

	return nil
}
