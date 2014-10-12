package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type CLI struct {
}

func (c *CLI) Parse(args []string) (*Config, error) {
	var dry, version bool
	config := &Config{}

	cmd := filepath.Base(args[0])

	flags := flag.NewFlagSet("consul-template", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, helpText, cmd)
	}
	flags.StringVar(&config.Consul, "consul", "127.0.0.1:8500",
		"address of the Consul instance")
	flags.Var((*configTemplateVar)(&config.ConfigTemplates), "template",
		"new template declaration")
	flags.StringVar(&config.Token, "token", "abcd1234",
		"a consul API token")
	flags.StringVar(&config.WaitRaw, "wait", "",
		"the minimum(:maximum) to wait before rendering a new template")
	flags.StringVar(&config.Path, "config", "",
		"the path to a config file on disk")
	flags.BoolVar(&config.Once, "once", false,
		"do not run as a daemon")
	flags.BoolVar(&dry, "dry", false,
		"write generated templates to stdout")

	// -version is special
	flags.BoolVar(&version, "version", false, "display the version")

	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		flags.Usage()
		os.Exit(1)
	}

	// If the version was requested, print and exit
	if version {
		fmt.Fprintf(os.Stderr, "%s v%s\n", cmd, Version)
		os.Exit(1)
	}

	// Parse the raw wait value into a Wait object
	if config.WaitRaw != "" {
		wait, err := ParseWait(config.WaitRaw)
		if err != nil {
			return nil, err
		}
		config.Wait = wait
	}

	// Merge a path config with the command line options. Command line options
	// take precedence over config file options for easy overriding.
	if config.Path != "" {
		c, err := ParseConfig(config.Path)
		if err != nil {
			panic(err)
			return nil, err
		}
		c.Merge(config)
		config = c
	}

	println(fmt.Sprintf("%#v", config))

	return config, nil
}

const helpText = `
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
