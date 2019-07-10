package dependency

import "testing"

func init() {
	VaultDefaultLeaseDuration = 0
}

func TestVaultRenewDuration(t *testing.T) {
	renewable := Secret{LeaseDuration: 100, Renewable: true}
	renewableDur := vaultRenewDuration(&renewable).Seconds()
	if renewableDur < 16 || renewableDur >= 34 {
		t.Fatalf("renewable duration is not within 1/6 to 1/3 of lease duration: %f", renewableDur)
	}

	nonRenewable := Secret{LeaseDuration: 100}
	nonRenewableDur := vaultRenewDuration(&nonRenewable).Seconds()
	if nonRenewableDur < 85 || nonRenewableDur > 95 {
		t.Fatalf("renewable duration is not within 85%% to 95%% of lease duration: %f", nonRenewableDur)
	}
}
