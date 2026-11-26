// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package watch

import dep "github.com/hashicorp/consul-template/dependency"

// ForcePKIRefresh requests fresh certificates for the active pkiCert dependencies
// of exactly one destination. It returns the number of matching dependencies,
// not a certificate issuance or template rendering completion result.
func (w *Watcher) ForcePKIRefresh(destination string) int {
	w.Lock()
	defer w.Unlock()
	if w.stopped || destination == "" {
		return 0
	}
	count := 0
	for _, view := range w.depViewMap {
		if view == nil {
			continue
		}
		if query, ok := view.Dependency().(*dep.VaultPKIQuery); ok && query.Destination() == destination {
			query.ForceRefresh()
			count++
		}
	}
	return count
}
