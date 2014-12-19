package main

import (
	"os"
)

// Name is the exported name of this application.
const Name = "consul-template"

// Version is the current version of this application.
const Version = "0.5.0"

func main() {
	cli := &CLI{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(cli.Run(os.Args))
}
