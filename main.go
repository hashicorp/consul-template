package main // import "github.com/hashicorp/consul-template"

import (
	"github.com/hashicorp/consul-template/cli"
	"os"
)

func main() {
	cliInstance := cli.NewCLI(os.Stdout, os.Stderr)
	os.Exit(cliInstance.Run(os.Args))
}
