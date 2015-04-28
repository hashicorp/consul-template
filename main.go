package main // import "github.com/marouenj/consul-template"

import (
	"os"
)

// Name is the exported name of this application.
const Name = "consul-template"

// Version is the current version of this application.
const Version = "0.8.1.dev"

func main() {
	cli := NewCLI(os.Stdout, os.Stderr)
	os.Exit(cli.Run(os.Args))
}
