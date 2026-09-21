// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

// These tests use an HTTP fixture and do not need Consul, Nomad or a Vault binary.
package pkirefresh_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/config"
	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/manager"
	"github.com/stretchr/testify/require"
)

func init() { dep.SetVaultLeaseRenewalThreshold(0.9) }

func certificate(t *testing.T, serial int64) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	c := &x509.Certificate{SerialNumber: big.NewInt(serial), NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(90 * 24 * time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, c, c, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func fixture(t *testing.T, replacement string) (*dep.ClientSet, *httptest.Server, *atomic.Int32, *atomic.Bool) {
	t.Helper()
	calls, fail := new(atomic.Int32), new(atomic.Bool)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/pki/issue/test" || r.Header.Get("X-Vault-Token") != "unchanged-token" {
			t.Errorf("unexpected request: %s, token matches: %v", r.URL.Path, r.Header.Get("X-Vault-Token") == "unchanged-token")
			w.WriteHeader(403)
			return
		}
		calls.Add(1)
		if fail.Load() {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"errors":["temporarily unavailable"]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"certificate": replacement}})
	}))
	t.Cleanup(s.Close)
	clients := dep.NewClientSet()
	require.NoError(t, clients.CreateVaultClient(&dep.CreateVaultClientInput{Address: s.URL, Token: "unchanged-token"}))
	clients.Vault().SetMaxRetries(0)
	t.Cleanup(clients.Stop)
	return clients, s, calls, fail
}

func TestForceRefreshBypassesValidDestinationAndRetries(t *testing.T) {
	old, newCert := certificate(t, 1), certificate(t, 2)
	clients, _, calls, fail := fixture(t, newCert)
	dest := filepath.Join(t.TempDir(), "certificate.pem")
	require.NoError(t, os.WriteFile(dest, []byte(old), 0600))
	query, err := dep.NewVaultPKIQuery("pki/issue/test", dest, nil)
	require.NoError(t, err)
	defer query.Stop()
	value, _, err := query.Fetch(clients, nil)
	require.NoError(t, err)
	require.Equal(t, old, value.(dep.PemEncoded).Cert)
	require.Zero(t, calls.Load())
	fail.Store(true)
	query.ForceRefresh()
	query.ForceRefresh() // queued requests coalesce before the next fetch
	_, _, err = query.Fetch(clients, nil)
	require.Error(t, err)
	require.EqualValues(t, 1, calls.Load())
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, old, string(data))
	fail.Store(false)
	value, _, err = query.Fetch(clients, nil)
	require.NoError(t, err)
	require.Equal(t, newCert, value.(dep.PemEncoded).Cert)
	require.EqualValues(t, 2, calls.Load())
	require.Equal(t, "unchanged-token", clients.Vault().Token())
}

func TestForceRefreshInterruptsWaitAndStop(t *testing.T) {
	for _, stop := range []bool{false, true} {
		t.Run(fmt.Sprintf("stop=%v", stop), func(t *testing.T) {
			old, newCert := certificate(t, 1), certificate(t, 2)
			clients, _, calls, _ := fixture(t, newCert)
			dest := filepath.Join(t.TempDir(), "certificate.pem")
			require.NoError(t, os.WriteFile(dest, []byte(old), 0600))
			query, err := dep.NewVaultPKIQuery("pki/issue/test", dest, nil)
			require.NoError(t, err)
			_, _, err = query.Fetch(clients, nil)
			require.NoError(t, err)
			done := make(chan error, 1)
			go func() { _, _, err := query.Fetch(clients, nil); done <- err }()
			select {
			case err := <-done:
				t.Fatalf("90-day wait returned early: %v", err)
			case <-time.After(50 * time.Millisecond):
			}
			if stop {
				query.Stop()
			} else {
				query.ForceRefresh()
				defer query.Stop()
			}
			select {
			case err := <-done:
				if stop {
					require.ErrorIs(t, err, dep.ErrStopped)
					require.Zero(t, calls.Load())
				} else {
					require.NoError(t, err)
					require.EqualValues(t, 1, calls.Load())
				}
			case <-time.After(2 * time.Second):
				t.Fatal("fetch did not wake")
			}
		})
	}
}

func TestRunnerForcePKIRefreshRendersAndExecutesOnlySelectedDestination(t *testing.T) {
	// DefaultConfig also reads the environment; never inherit a real Vault address.
	t.Setenv("VAULT_ADDR", "")
	t.Setenv("VAULT_TOKEN", "")
	t.Setenv("VAULT_NAMESPACE", "")
	old, newCert := certificate(t, 1), certificate(t, 2)
	_, server, calls, _ := fixture(t, newCert)
	dir := t.TempDir()
	dest := filepath.Join(dir, "selected.pem")
	other := filepath.Join(dir, "other.pem")
	hook := filepath.Join(dir, "exec.pem")
	for _, p := range []string{dest, other} {
		require.NoError(t, os.WriteFile(p, []byte(old), 0600))
	}
	cfg := config.DefaultConfig()
	cfg.Vault.Address = config.String(server.URL)
	cfg.Vault.Token = config.String("unchanged-token")
	cfg.Vault.RenewToken = config.Bool(false)
	cfg.Vault.SSL.Enabled = config.Bool(false)
	cfg.Templates = &config.TemplateConfigs{
		{Contents: config.String(`{{ with pkiCert "pki/issue/test" }}{{ .Cert }}{{ end }}`), Destination: config.String(dest), Exec: &config.ExecConfig{Command: []string{os.Args[0], "-test.run=^TestExecHelper$", "--", "pki-refresh-helper", dest, hook}}},
		{Contents: config.String(`{{ with pkiCert "pki/issue/test" }}{{ .Cert }}{{ end }}`), Destination: config.String(other)},
	}
	runner, err := manager.NewRunner(cfg, false)
	require.NoError(t, err)
	done := make(chan struct{})
	go func() { runner.Start(); close(done) }()
	defer func() {
		runner.Stop()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("runner did not stop")
		}
	}()
	require.Eventually(t, func() bool { return len(runner.RenderEvents()) == 2 }, 3*time.Second, 10*time.Millisecond)
	require.Zero(t, runner.ForcePKIRefresh(filepath.Join(dir, "unknown.pem")))
	require.Equal(t, 1, runner.ForcePKIRefresh(dest))
	require.Eventually(t, func() bool { b, _ := os.ReadFile(hook); return string(b) == newCert }, 5*time.Second, 10*time.Millisecond)
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, newCert, string(data))
	data, err = os.ReadFile(other)
	require.NoError(t, err)
	require.Equal(t, old, string(data))
	require.EqualValues(t, 1, calls.Load())
}

func TestExecHelper(t *testing.T) {
	args := os.Args
	if len(args) < 4 || args[len(args)-3] != "pki-refresh-helper" {
		return
	}
	data, err := os.ReadFile(args[len(args)-2])
	if err != nil {
		os.Exit(1)
	}
	if os.WriteFile(args[len(args)-1], data, 0600) != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
