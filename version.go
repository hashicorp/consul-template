package main

import (
	"bytes"
	"fmt"
)

var (
	// Name is the name of the binary.
	Name string = "consul-template"

	// Version is the major version. VersionPrerelease is the prerelease version.
	// If VersionPrerelease is empty, this is an official release.
	Version           string = "0.16.0"
	VersionPrerelease string = "rc1"

	// GitCommit is the commit. It will be filled in by the compier.
	GitCommit string
)

// formattedVersion returns a formatted version string which includes the git
// commit and development information.
func formattedVersion() string {
	var versionString bytes.Buffer
	fmt.Fprintf(&versionString, "%s v%s", Name, Version)

	if VersionPrerelease != "" {
		fmt.Fprintf(&versionString, "%s", VersionPrerelease)

		if GitCommit != "" {
			fmt.Fprintf(&versionString, " (%s)", GitCommit)
		}
	}
	return versionString.String()
}
