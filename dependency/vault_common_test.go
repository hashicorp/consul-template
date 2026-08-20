// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
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
	renewableDurSec := renewableDur.Seconds()
	if renewableDurSec < 16 || renewableDurSec >= 34 {
		t.Fatalf("renewable duration is not within 1/6 to 1/3 of lease duration: %f", renewableDurSec)
	}

	nonRenewable := Secret{LeaseDuration: 100}
	nonRenewableDur, _ := leaseCheckWait(&nonRenewable)
	nonRenewableDurSec := nonRenewableDur.Seconds()

	if nonRenewableDurSec < 80 || nonRenewableDurSec > 95 {
		t.Fatalf("renewable duration is not within 80%% to 95%% of lease duration: %f", nonRenewableDurSec)
	}

	data := map[string]interface{}{
		"rotation_period": json.Number("60"),
		"ttl":             json.Number("30"),
	}

	nonRenewableRotated := Secret{LeaseDuration: 100, Data: data}
	nonRenewableRotatedDur, _ := leaseCheckWait(&nonRenewableRotated)
	nonRenewableRotatedDurSec := nonRenewableRotatedDur.Seconds()
	// We expect a 1 second cushion
	if nonRenewableRotatedDurSec != 31 {
		t.Fatalf("renewable duration is not 31: %f", nonRenewableRotatedDurSec)
	}

	data = map[string]interface{}{
		"rotation_period": json.Number("30"),
		"ttl":             json.Number("5"),
	}

	nonRenewableRotated = Secret{LeaseDuration: 100, Data: data}
	nonRenewableRotatedDur, _ = leaseCheckWait(&nonRenewableRotated)
	nonRenewableRotatedDurSeconds := nonRenewableRotatedDur.Seconds()
	// We expect a 1 second cushion
	if nonRenewableRotatedDurSeconds != 6 {
		t.Fatalf("renewable duration is not 6: %f", nonRenewableRotatedDurSeconds)
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
	nonRenewableCertDurSec := nonRenewableCertDur.Seconds()

	if nonRenewableCertDurSec < 80 || nonRenewableCertDurSec > 95 {
		t.Fatalf("non renewable certificate duration is not within 80%% to 95%%: %f", nonRenewableCertDurSec)
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
			nonRenewableSecretIDDurSec := nonRenewableSecretIDDur.Seconds()

			if nonRenewableSecretIDDurSec < 0.80*(60+1) || nonRenewableSecretIDDurSec > 0.95*(60+1) {
				t.Fatalf("renewable duration is not within 80%% to 95%% of lease duration: %f", nonRenewableSecretIDDurSec)
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
			nonRenewableSecretIDDurSec := nonRenewableSecretIDDur.Seconds()

			if nonRenewableSecretIDDurSec < 0.80*(leaseDuration+1) || nonRenewableSecretIDDurSec > 0.95*(leaseDuration+1) {
				t.Fatalf("renewable duration is not within 80%% to 95%% of lease duration: %f", nonRenewableSecretIDDurSec)
			}
		})

		t.Run("ttl missing", func(t *testing.T) {
			const leaseDuration = 1000

			data := map[string]interface{}{
				"secret_id": "abc",
			}

			nonRenewableSecretID := Secret{LeaseDuration: leaseDuration, Data: data}
			nonRenewableSecretIDDur, _ := leaseCheckWait(&nonRenewableSecretID)
			nonRenewableSecretIDDurSec := nonRenewableSecretIDDur.Seconds()

			if nonRenewableSecretIDDurSec < 0.80*(leaseDuration+1) || nonRenewableSecretIDDurSec > 0.95*(leaseDuration+1) {
				t.Fatalf("renewable duration is not within 80%% to 95%% of lease duration: %f", nonRenewableSecretIDDurSec)
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

// TestRenewSecretBoundedOnRenewalFailure verifies that renewSecret does not
// return immediately when every renewal attempt fails with HTTP 400. The
// LifetimeWatcher must block via exponential backoff for the duration of the
// lease rather than exiting on the first error and triggering a new credential
// fetch on each iteration.
func TestRenewSecretBoundedOnRenewalFailure(t *testing.T) {
	var renewAttempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/v1/sys/leases/renew" {
			atomic.AddInt32(&renewAttempts, 1)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, `{"errors":["failed to renew entry: bad renew_statement"]}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{}`)
	}))
	defer srv.Close()

	cfg := api.DefaultConfig()
	cfg.Address = srv.URL
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}
	client.SetToken("test-token")

	clients := &ClientSet{
		vault: &vaultClient{
			client:     client,
			httpClient: cfg.HttpClient,
		},
	}

	// 60s TTL — typical database credential lease.
	vaultSecret := &api.Secret{
		LeaseID:       "database/creds/my-role/abc123",
		LeaseDuration: 60,
		Renewable:     true,
	}
	secret := transformSecret(vaultSecret)
	stopCh := make(chan struct{})
	d := &renewalTestDep{
		secret:      secret,
		vaultSecret: vaultSecret,
		stopCh:      stopCh,
	}

	// Simulate the Fetch() loop: call renewSecret in a tight loop and count
	// returns. The watcher should block via backoff — 0 returns expected in 5s.
	var returns int32
	go func() {
		for {
			select {
			case <-stopCh:
				return
			default:
			}
			renewSecret(clients, d) //nolint:errcheck
			atomic.AddInt32(&returns, 1)
		}
	}()

	time.Sleep(5 * time.Second)
	close(stopCh)

	got := int(atomic.LoadInt32(&returns))
	t.Logf("renewSecret returned %d times, renew endpoint hit %d times in 5s (TTL=60s)",
		got, atomic.LoadInt32(&renewAttempts))

	if got > 0 {
		t.Errorf("renewSecret returned %d times in 5s with a 60s TTL and failing renewals "+
			"(want 0); RenewBehaviorIgnoreErrors backoff is not working — "+
			"credential creation would be unbounded on renewal failure", got)
	}
}

// renewalTestDep is a minimal implementation of the renewer interface for
// use in TestRenewSecretBoundedOnRenewalFailure.
type renewalTestDep struct {
	secret      *Secret
	vaultSecret *api.Secret
	stopCh      chan struct{}
}

func (d *renewalTestDep) stopChan() chan struct{}          { return d.stopCh }
func (d *renewalTestDep) secrets() (*Secret, *api.Secret) { return d.secret, d.vaultSecret }
func (d *renewalTestDep) CanShare() bool                  { return false }
func (d *renewalTestDep) Fetch(*ClientSet, *QueryOptions) (interface{}, *ResponseMetadata, error) {
	return nil, nil, nil
}
func (d *renewalTestDep) Stop()          {}
func (d *renewalTestDep) String() string { return "test.renewal" }
func (d *renewalTestDep) Type() Type     { return TypeVault }
