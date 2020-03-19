package dependency

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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
	}

	nonRenewableRotated := Secret{LeaseDuration: 100, Data: data}
	nonRenewableRotatedDur := leaseCheckWait(&nonRenewableRotated).Seconds()
	if nonRenewableRotatedDur != 60 {
		t.Fatalf("renewable duration is not 60: %f", nonRenewableRotatedDur)
	}
}

const testGoodCert = `-----BEGIN CERTIFICATE-----
MIICAjCCAWugAwIBAgIJALDrJbXZKXXnMA0GCSqGSIb3DQEBCwUAMBoxGDAWBgNV
BAMMD2NvbnN1bC10ZW1wbGF0ZTAeFw0xODA1MjUxNTAzNDdaFw0xODA2MDQxNTAz
NDdaMBoxGDAWBgNVBAMMD2NvbnN1bC10ZW1wbGF0ZTCBnzANBgkqhkiG9w0BAQEF
AAOBjQAwgYkCgYEAuT1yS2FvX2bpNvEkrapt4wC68NIfTU9Xx55DC4/Pq1ZkuI8b
tC64x1oiJdM7ABEmT58rofTXoEpeHxcLTpXtJcrfLdgHUkPxNdrBgLWJi0BGI3m6
zLF9KLTwEpFfBBTLgM6HIvTqqBD4itFtI0BDS/mqQKqa33Ai6hX0zPAH6AECAwEA
AaNQME4wHQYDVR0OBBYEFLldqcFQ+RF40xBNgSjdNGBN78yHMB8GA1UdIwQYMBaA
FLldqcFQ+RF40xBNgSjdNGBN78yHMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEL
BQADgYEAUXeDp5pyGhH3RCxdJgjbQ67D5nqTVbTJnetEw1UdMEDQGrgCIUrbsJWm
G4SbKUjKP+4wVUJLZpmv9PwJcN0ZxntNkJBDzTk+KULu4+8cCj6A27bBhmzeOu1y
zZlyse1m1NECY3ryPtkst4U/0wCiKcI4ZW58RrhXgKucB3Y0C0w=
-----END CERTIFICATE-----`

const testBadCert = `-----BEGIN CERTIFICATE-----
THIS IS NOT A VALID CERT
-----END CERTIFICATE-----`

func TestDurationFromCert(t *testing.T) {
	t.Parallel()

	dur := durationFromCert(testGoodCert)

	// 10 days in seconds
	assert.Equal(t, 864000, dur)

	dur = durationFromCert(testBadCert)

	// Negative duration means an invalid cert
	assert.Equal(t, -1, dur)
}
