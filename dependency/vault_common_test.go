package dependency

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func init() {
	VaultDefaultLeaseDuration = 0
}

func TestVaultRenewDuration(t *testing.T) {
	renewable := Secret{LeaseDuration: 100, Renewable: true}
	renewableDur := leaseCheckWait(&renewable).Seconds()
	if renewableDur < 16 || renewableDur >= 34 {
		t.Fatalf("renewable duration is not within 1/6 to 1/3 of lease duration: %f", renewableDur)
	}

	nonRenewable := Secret{LeaseDuration: 100}
	nonRenewableDur := leaseCheckWait(&nonRenewable).Seconds()
	if nonRenewableDur < 85 || nonRenewableDur > 95 {
		t.Fatalf("renewable duration is not within 85%% to 95%% of lease duration: %f", nonRenewableDur)
	}

	var data = map[string]interface{}{
		"rotation_period": json.Number("60"),
		"ttl":             json.Number("30"),
	}

	nonRenewableRotated := Secret{LeaseDuration: 100, Data: data}
	nonRenewableRotatedDur := leaseCheckWait(&nonRenewableRotated).Seconds()

	// We expect a 1 second cushion
	if nonRenewableRotatedDur != 31 {
		t.Fatalf("renewable duration is not 31: %f", nonRenewableRotatedDur)
	}

	data = map[string]interface{}{
		"rotation_period": json.Number("30"),
		"ttl":             json.Number("5"),
	}

	nonRenewableRotated = Secret{LeaseDuration: 100, Data: data}
	nonRenewableRotatedDur = leaseCheckWait(&nonRenewableRotated).Seconds()

	// We expect a 1 second cushion
	if nonRenewableRotatedDur != 6 {
		t.Fatalf("renewable duration is not 6: %f", nonRenewableRotatedDur)
	}

	rawExpiration := time.Now().Unix() + 100
	expiration := strconv.FormatInt(rawExpiration, 10)

	data = map[string]interface{}{
		"expiration":  json.Number(expiration),
		"certificate": "foobar",
	}

	nonRenewableCert := Secret{LeaseDuration: 100, Data: data}
	nonRenewableCertDur := leaseCheckWait(&nonRenewableCert).Seconds()
	if nonRenewableCertDur < 85 || nonRenewableCertDur > 95 {
		t.Fatalf("non renewable certificate duration is not within 85%% to 95%%: %f", nonRenewableCertDur)
	}

	t.Run("secret ID handling", func(t *testing.T) {
		t.Run("normal case", func(t *testing.T) {
			// Secret ID TTL handling
			data := map[string]interface{}{
				"secret_id":     "abc",
				"secret_id_ttl": json.Number("60"),
			}

			nonRenewableSecretID := Secret{LeaseDuration: 100, Data: data}
			nonRenewableSecretIDDur := leaseCheckWait(&nonRenewableSecretID).Seconds()

			if nonRenewableSecretIDDur < 0.85*(60+1) || nonRenewableSecretIDDur > 0.95*(60+1) {
				t.Fatalf("renewable duration is not within 85%% to 95%% of lease duration: %f", nonRenewableSecretIDDur)
			}
		})

		t.Run("0 ttl", func(t *testing.T) {
			const leaseDuration = 1000

			data := map[string]interface{}{
				"secret_id":     "abc",
				"secret_id_ttl": json.Number("0"),
			}

			nonRenewableSecretID := Secret{LeaseDuration: leaseDuration, Data: data}
			nonRenewableSecretIDDur := leaseCheckWait(&nonRenewableSecretID).Seconds()

			if nonRenewableSecretIDDur < 0.85*(leaseDuration+1) || nonRenewableSecretIDDur > 0.95*(leaseDuration+1) {
				t.Fatalf("renewable duration is not within 85%% to 95%% of lease duration: %f", nonRenewableSecretIDDur)
			}
		})

		t.Run("ttl missing", func(t *testing.T) {
			const leaseDuration = 1000

			data := map[string]interface{}{
				"secret_id": "abc",
			}

			nonRenewableSecretID := Secret{LeaseDuration: leaseDuration, Data: data}
			nonRenewableSecretIDDur := leaseCheckWait(&nonRenewableSecretID).Seconds()

			if nonRenewableSecretIDDur < 0.85*(leaseDuration+1) || nonRenewableSecretIDDur > 0.95*(leaseDuration+1) {
				t.Fatalf("renewable duration is not within 85%% to 95%% of lease duration: %f", nonRenewableSecretIDDur)
			}
		})

	})
}
