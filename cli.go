package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	api "github.com/armon/consul-api"
	"github.com/hashicorp/consul-template/util"
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
	ExitCodeParseConfigError
	ExitCodeRunnerError
	ExitCodeConsulAPIError
	ExitCodeWatcherError
)

/// ------------------------- ///

// CLI is the main entry point for Consul Template.
type CLI struct {
	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func (cli *CLI) Run(args []string) int {
	cli.initLogger()

	var version, dry, once bool
	var config = new(Config)

	// Parse the flags and options
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprintf(cli.errStream, usage, Name)
	}
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

	// Parse the raw wait value into a Wait object
	if config.WaitRaw != "" {
		log.Printf("[DEBUG] (cli) detected -wait, parsing")
		wait, err := util.ParseWait(config.WaitRaw)
		if err != nil {
			return cli.handleError(err, ExitCodeParseWaitError)
		}
		config.Wait = wait
	}

	// Initial bootstrap
	runner, watcher, err := bootstrap(config, dry, once)
	if err != nil {
		return cli.handleError(err, 1)
	}

	var minTimer, maxTimer <-chan time.Time

	// Listen for signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	for {
		log.Printf("[DEBUG] (cli) looping for data")

		select {
		case data := <-watcher.DataCh:
			log.Printf("[INFO] (cli) received data from Watcher for %s",
				data.Dependency.Display())

			// Tell the Runner about the data
			runner.Receive(data.Dependency, data.Data)

			// If we are waiting for quiescence, setup the timers
			if config.Wait != nil {
				log.Printf("[DEBUG] (cli) detected quiescence, starting timers")

				// Reset the min timer
				minTimer = time.After(config.Wait.Min)

				// Set the max timer if it does not already exist
				if maxTimer == nil {
					maxTimer = time.After(config.Wait.Max)
				}
			} else {
				log.Printf("[INFO] (cli) invoking Runner")
				if err := runner.RunAll(dry); err != nil {
					return cli.handleError(err, ExitCodeRunnerError)
				}
			}
		case <-minTimer:
			log.Printf("[DEBUG] (cli) quiescence minTimer fired, invoking Runner")

			minTimer, maxTimer = nil, nil

			if err := runner.RunAll(dry); err != nil {
				return cli.handleError(err, ExitCodeRunnerError)
			}
		case <-maxTimer:
			log.Printf("[DEBUG] (cli) quiescence maxTimer fired, invoking Runner")

			minTimer, maxTimer = nil, nil

			if err := runner.RunAll(dry); err != nil {
				return cli.handleError(err, ExitCodeRunnerError)
			}
		case err := <-watcher.ErrCh:
			return cli.handleError(err, ExitCodeError)
		case <-watcher.FinishCh:
			log.Printf("[INFO] (cli) received finished signal, exiting now")
			return ExitCodeOK
		case s := <-signalCh:
			switch s {
			case os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				fmt.Fprintf(cli.errStream, "Received interrupt, cleaning up...\n")
				watcher.Stop()
				return ExitCodeInterrupt
			case syscall.SIGHUP:
				fmt.Fprintf(cli.errStream, "Received HUP, reloading configuration...\n")
				watcher.Stop()
				runner, watcher, err = bootstrap(config, dry, once)
				if err != nil {
					return cli.handleError(err, 1)
				}
			default:
				fmt.Fprintf(cli.errStream, "wtf: %#v", s)
			}
		}
	}
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (cli *CLI) handleError(err error, status int) int {
	log.Printf("[ERR] %s", err.Error())
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

// bootstrap accepts the configuration, a dry flag, and a once flag and creates
// all the required components to make a new watcher object. This function
// returns the created Runner and Watcher. If an error occurs, it is returned as
// the final parameter.
func bootstrap(config *Config, dry bool, once bool) (*Runner, *util.Watcher, error) {
	// Merge a path config with the command line options. Command line options
	// take precedence over config file options for easy overriding.
	if config.Path != "" {
		log.Printf("[DEBUG] (cli) detected -config, merging")
		err := buildConfig(config, config.Path)
		if err != nil {
			return nil, nil, err
		}
	}

	log.Printf("[DEBUG] (cli) creating Runner")
	runner, err := NewRunner(config.ConfigTemplates)
	if err != nil {
		return nil, nil, err
	}

	// Run all templates now. There are currently no dependencies because the
	// watcher has not been started. As a result, this will render all templates
	// that have no dependencies (once), before we even begin watching.
	if err := runner.RunAll(dry); err != nil {
		return nil, nil, err
	}

	log.Printf("[DEBUG] (cli) creating Consul API client")
	consulConfig := api.DefaultConfig()
	if config.Consul != "" {
		consulConfig.Address = config.Consul
	}
	if config.Token != "" {
		consulConfig.Token = config.Token
	}
	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("[DEBUG] (cli) creating Watcher")
	watcher, err := util.NewWatcher(client, runner.Dependencies())
	if err != nil {
		return nil, nil, err
	}

	// Set the retry timeout on the watcher if one was given
	if config.Retry != 0 {
		watcher.SetRetry(config.Retry)
	}

	// Start the watcher in the background
	go watcher.Watch(once)

	return runner, watcher, nil
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

	return nil
}

const usage = `
Usage: %s [options]

  Watches a series of templates on the file system, writing new changes when
  Consul is updated. It runs until an interrupt is received unless the -once
  flag is specified.

Options:

  -consul=<address>        Sets the address of the Consul instance
  -token=<token>           Sets the Consul API token
  -template=<template>     Adds a new template to watch on disk in the format
                           'templatePath:outputPath(:command)'.
  -wait=<duration>         Sets the 'minumum(:maximum)' amount of time to wait
                           before writing a template (and triggering a command)
  -retry=<duration>        The amount of time to wait if Consul returns an
                           error when communicating with the API.
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
