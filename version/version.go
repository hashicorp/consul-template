// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package version

import "fmt"

const (
	Version           = "0.41.3"
	VersionPrerelease = "" // "-dev", "-beta", "-rc1", etc. (include dash)
)

var (
	Name      string = "consul-template"
	GitCommit string

	HumanVersion = fmt.Sprintf("%s v%s%s (%s)",
		Name, Version, VersionPrerelease, GitCommit)
)
