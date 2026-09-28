// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package manager

// ForcePKIRefresh wakes the active pkiCert dependencies for an exact destination.
// The existing watchers publish the new certificate through the normal render
// and command path. Other destinations and the current Vault token are unchanged.
// Zero means no matching active PKI dependency (including before initial render).
func (r *Runner) ForcePKIRefresh(destination string) int {
	return r.watcher.ForcePKIRefresh(destination)
}
