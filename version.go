package main

import "fmt"

var (
	Name      string
	Version   string
	GitCommit string

	humanVersion = fmt.Sprintf("%s v%s (%s)", Name, Version, GitCommit)
)
