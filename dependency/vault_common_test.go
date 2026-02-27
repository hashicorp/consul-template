// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

func init() {
	VaultDefaultLeaseDuration = 0
	VaultLeaseRenewalThreshold = .90
}

func TestVaultRenewDuration(t *testing.T) {
	renewable := Secret{LeaseDuration: 100, Renewable: true}
	renewableDur, _ := leaseCheckWait(&renewable)
	if renewableDur < 16*time.Second || renewableDur >= 34*time.Second {
		t.Fatalf("renewable duration is not within 1/6 to 1/3 of lease duration: %f", renewableDur.Seconds())
	}

	nonRenewable := Secret{LeaseDuration: 100}
	nonRenewableDur, _ := leaseCheckWait(&nonRenewable)
	if nonRenewableDur < 80*time.Second || nonRenewableDur > 95*time.Second {
		t.Fatalf("renewable duration is not within 80%% to 95%% of lease duration: %f", nonRenewableDur.Seconds())
	}

	data := map[string]interface{}{
		"rotation_period": json.Number("60"),
		"ttl":             json.Number("30"),
	}

	nonRenewableRotated := Secret{LeaseDuration: 100, Data: data}
	nonRenewableRotatedDur, _ := leaseCheckWait(&nonRenewableRotated)
	// We expect a 1 second cushion
	if nonRenewableRotatedDur != 31*time.Second {
		t.Fatalf("renewable duration is not 31: %f", nonRenewableRotatedDur.Seconds())
	}

	data = map[string]interface{}{
		"rotation_period": json.Number("30"),
		"ttl":             json.Number("5"),
	}

	nonRenewableRotated = Secret{LeaseDuration: 100, Data: data}
	nonRenewableRotatedDur, _ = leaseCheckWait(&nonRenewableRotated)
	// We expect a 1 second cushion
	if nonRenewableRotatedDur != 6*time.Second {
		t.Fatalf("renewable duration is not 6: %f", nonRenewableRotatedDur.Seconds())
	}
	// Test TTL=0 case - should return error
	data = map[string]interface{}{
		"rotation_period": json.Number("30"),
		"ttl":             json.Number("0"),
	}
	nonRenewableRotatedZero := Secret{LeaseDuration: 100, Data: data}
	_, err := leaseCheckWait(&nonRenewableRotatedZero)
	if err == nil {
		t.Fatalf("expected error for ttl=0, got nil")
	}
	if err.Error() != "vault rotating secret returned ttl=0, will retry" {
		t.Fatalf("expected ttl=0 error message, got: %v", err)
	}

	rawExpiration := time.Now().Unix() + 100
	expiration := strconv.FormatInt(rawExpiration, 10)

	data = map[string]interface{}{
		"expiration":  json.Number(expiration),
		"certificate": "foobar",
	}

	nonRenewableCert := Secret{LeaseDuration: 100, Data: data}
	nonRenewableCertDur, _ := leaseCheckWait(&nonRenewableCert)
	if nonRenewableCertDur < 80*time.Second || nonRenewableCertDur > 95*time.Second {
		t.Fatalf("non renewable certificate duration is not within 80%% to 95%%: %f", nonRenewableCertDur.Seconds())
	}

	t.Run("secret ID handling", func(t *testing.T) {
		t.Run("normal case", func(t *testing.T) {
			// Secret ID TTL handling
			data := map[string]interface{}{
				"secret_id":     "abc",
				"secret_id_ttl": json.Number("60"),
			}

			nonRenewableSecretID := Secret{LeaseDuration: 100, Data: data}
			nonRenewableSecretIDDur, _ := leaseCheckWait(&nonRenewableSecretID)
			secs := nonRenewableSecretIDDur.Seconds()

			if secs < 0.80*(60+1) || secs > 0.95*(60+1) {
				t.Fatalf("renewable duration is not within 80%% to 95%% of lease duration: %f", secs)
			}
		})

		t.Run("0 ttl", func(t *testing.T) {
			const leaseDuration = 1000

			data := map[string]interface{}{
				"secret_id":     "abc",
				"secret_id_ttl": json.Number("0"),
			}

			nonRenewableSecretID := Secret{LeaseDuration: leaseDuration, Data: data}
			nonRenewableSecretIDDur, _ := leaseCheckWait(&nonRenewableSecretID)
			secs := nonRenewableSecretIDDur.Seconds()

			if secs < 0.80*(leaseDuration+1) || secs > 0.95*(leaseDuration+1) {
				t.Fatalf("renewable duration is not within 80%% to 95%% of lease duration: %f", secs)
			}
		})

		t.Run("ttl missing", func(t *testing.T) {
			const leaseDuration = 1000

			data := map[string]interface{}{
				"secret_id": "abc",
			}

			nonRenewableSecretID := Secret{LeaseDuration: leaseDuration, Data: data}
			nonRenewableSecretIDDur, _ := leaseCheckWait(&nonRenewableSecretID)
			secs := nonRenewableSecretIDDur.Seconds()

			if secs < 0.80*(leaseDuration+1) || secs > 0.95*(leaseDuration+1) {
				t.Fatalf("renewable duration is not within 80%% to 95%% of lease duration: %f", secs)
			}
		})
	})
}

func setupVaultPKI(clients *ClientSet) {
	err := clients.Vault().Sys().Mount("pki", &api.MountInput{
		Type: "pki",
	})
	switch {
	case err == nil:
	case strings.Contains(err.Error(), "path is already in use"):
		// for idempotency
		return
	default:
		panic(err)
	}

	vc := clients.Vault()

	_, err = vc.Logical().Write("pki/root/generate/internal",
		map[string]interface{}{
			"common_name": "example.com",
			"ttl":         "48h",
		})
	if err != nil {
		panic(err)
	}

	for needCA, count := true, 0; needCA && count < 5; count++ {
		l, err := vc.Logical().List("pki/keys")
		if err != nil && !strings.Contains(err.Error(), "connection refused") {
			panic(err)
		}
		if l != nil {
			needCA = false
		}
		time.Sleep(time.Millisecond)
	}

	_, err = vc.Logical().Write("pki/roles/example-dot-com",
		map[string]interface{}{
			"allowed_domains":     "example.com",
			"allow_subdomains":    "true",
			"not_before_duration": "1s",
		})
	if err != nil {
		panic(err)
	}
}
