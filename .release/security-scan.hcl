# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

container {
	dependencies = true
	alpine_secdb = true
	secrets      = true
	triage {
		suppress {
			vulnerabilites = []
		}
	}
}

binary {
	secrets      = true
	go_modules   = true
	osv          = true
	oss_index    = false
	nvd          = false
	triage {
		suppress {
			# golang.org/x/crypto/openpgp is unmaintained and has no fixed
			# version. consul-template does not import the openpgp packages,
			# so this advisory is not reachable.
			vulnerabilites = [
				"GO-2026-5932",
			]
		}
	}
}
