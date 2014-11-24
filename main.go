package main

import (
	"os"
)

const Name = "consul-template"
const Version = "0.3.1"

func main() {
	cli := &CLI{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(cli.Run(os.Args))
}
