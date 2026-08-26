// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"testing"
	"time"
)

// A caching proxy in front of Vault replays a stored response verbatim and
// reports how stale it is via the HTTP Age header. The lease_duration in that
// response is measured from when the lease was issued, not from when the client
// received it, so the lifetime a lease has left is lease_duration - Age.
//
// leaseCheckWait decides how long to sleep before re-rendering a non-renewable
// secret. It must wake before the lease has run out, or the secret it renders
// on waking is already dead. The package init sets VaultLeaseRenewalThreshold
// to 0.90, so the sleep is a deterministic fraction of the remaining lease.
func TestLeaseCheckWait_AccountsForCacheAge(t *testing.T) {
	const (
		leaseDurationSeconds = 100
		age                  = 90 * time.Second
	)
	remaining := time.Duration(leaseDurationSeconds)*time.Second - age

	secret := &Secret{
		LeaseID:       "database/creds/readonly/abcd1234",
		LeaseDuration: leaseDurationSeconds,
		Age:           age,
	}

	dur, err := leaseCheckWait(secret)
	if err != nil {
		t.Fatal(err)
	}

	if dur >= remaining {
		t.Fatalf("a lease %ds old on a %ds duration has only %s left; slept %s, waking after it expired",
			int(age.Seconds()), leaseDurationSeconds, remaining, dur)
	}
}
