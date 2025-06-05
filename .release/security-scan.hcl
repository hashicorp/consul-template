# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

container {
	dependencies = true
	alpine_secdb = true
	secrets      = true
	triage {
		suppress {
			vulnerabilites = [
				"CVE-2024-58251",	# fix unavailable at time of writing
				"CVE-2025-46394"	# fix unavailable at time of writing
			]
		}
	}

}

binary {
	secrets      = true
	go_modules   = true
	osv          = true
	oss_index    = false
	nvd          = false
}