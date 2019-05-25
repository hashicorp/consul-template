package main

import (
	"flag"
	"fmt"

	"github.com/pierrec/lz4/internal/cmdflag"
	"github.com/pierrec/lz4/internal/cmds"
)

func init() {
	const onError = flag.ExitOnError
	cmdflag.New(
		"compress", "[arguments] [<file name> ...]",
		"Compress the given files or from stdin to stdout.",
		onError, cmds.Compress)
	cmdflag.New(
		"uncompress", "[arguments] [<file name> ...]",
		"Uncompress the given files or from stdin to stdout.",
		onError, cmds.Uncompress)
}

func main() {
	flag.CommandLine.Bool(cmdflag.VersionBoolFlag, false, "print the program version")

	err := cmdflag.Parse()
	if err != nil {
		fmt.Println(err)
		return
	}
}
