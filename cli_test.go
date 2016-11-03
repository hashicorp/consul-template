package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/config"
	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/go-gatedio"
	"github.com/stretchr/testify/assert"
)

func TestParseFlags_consul(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-consul", "12.34.56.78",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "12.34.56.78"
	if config.Consul != expected {
		t.Errorf("expected %q to be %q", config.Consul, expected)
	}
	if !config.WasSet("consul") {
		t.Errorf("expected consul to be set")
	}
}

func TestParseFlags_token(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-token", "abcd1234",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "abcd1234"
	if config.Token != expected {
		t.Errorf("expected %q to be %q", config.Token, expected)
	}
	if !config.WasSet("token") {
		t.Errorf("expected token to be set")
	}
}

func TestParseFlags_reloadSignal(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-reload-signal", "SIGHUP",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := syscall.SIGHUP
	if config.ReloadSignal != expected {
		t.Errorf("expected %#v to be %#v", config.ReloadSignal, expected)
	}
	if !config.WasSet("reload_signal") {
		t.Errorf("expected reload_signal to be set")
	}
}

func TestParseFlags_reloadSignal_empty(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-reload-signal", "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.ReloadSignal != nil {
		t.Errorf("expected %#v to be %#v", config.ReloadSignal, nil)
	}
	if !config.WasSet("reload_signal") {
		t.Errorf("expected reload_signal to be set")
	}
}

func TestParseFlags_dumpSignal(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-dump-signal", "SIGHUP",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := syscall.SIGHUP
	if config.DumpSignal != expected {
		t.Errorf("expected %#v to be %#v", config.DumpSignal, expected)
	}
	if !config.WasSet("dump_signal") {
		t.Errorf("expected dump_signal to be set")
	}
}

func TestParseFlags_dumpSignal_empty(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-dump-signal", "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.DumpSignal != nil {
		t.Errorf("expected %#v to be %#v", config.DumpSignal, nil)
	}
	if !config.WasSet("dump_signal") {
		t.Errorf("expected dump_signal to be set")
	}
}

func TestParseFlags_killSignal(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-kill-signal", "SIGHUP",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := syscall.SIGHUP
	if config.KillSignal != expected {
		t.Errorf("expected %#v to be %#v", config.KillSignal, expected)
	}
	if !config.WasSet("kill_signal") {
		t.Errorf("expected kill_signal to be set")
	}
}

func TestParseFlags_killSignal_empty(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-kill-signal", "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.KillSignal != nil {
		t.Errorf("expected %#v to be %#v", config.KillSignal, nil)
	}
	if !config.WasSet("kill_signal") {
		t.Errorf("expected kill_signal to be set")
	}
}

func TestParseFlags_authUsername(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-auth", "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Auth.Enabled != true {
		t.Errorf("expected auth to be enabled")
	}
	if !config.WasSet("auth.enabled") {
		t.Errorf("expected auth.enabled to be set")
	}

	expected := "test"
	if config.Auth.Username != expected {
		t.Errorf("expected %v to be %v", config.Auth.Username, expected)
	}
	if !config.WasSet("auth.username") {
		t.Errorf("expected auth.username to be set")
	}
}

func TestParseFlags_authUsernamePassword(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-auth", "test:test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Auth.Enabled != true {
		t.Errorf("expected auth to be enabled")
	}
	if !config.WasSet("auth.enabled") {
		t.Errorf("expected auth.enabled to be set")
	}

	expected := "test"
	if config.Auth.Username != expected {
		t.Errorf("expected %v to be %v", config.Auth.Username, expected)
	}
	if !config.WasSet("auth.username") {
		t.Errorf("expected auth.username to be set")
	}
	if config.Auth.Password != expected {
		t.Errorf("expected %v to be %v", config.Auth.Password, expected)
	}
	if !config.WasSet("auth.password") {
		t.Errorf("expected auth.password to be set")
	}
}

func TestParseFlags_SSL(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.SSL.Enabled != expected {
		t.Errorf("expected %v to be %v", config.SSL.Enabled, expected)
	}
	if !config.WasSet("ssl") {
		t.Errorf("expected ssl to be set")
	}
	if !config.WasSet("ssl.enabled") {
		t.Errorf("expected ssl.enabled to be set")
	}
}

func TestParseFlags_noSSL(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl=false",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := false
	if config.SSL.Enabled != expected {
		t.Errorf("expected %v to be %v", config.SSL.Enabled, expected)
	}
	if !config.WasSet("ssl") {
		t.Errorf("expected ssl to be set")
	}
	if !config.WasSet("ssl.enabled") {
		t.Errorf("expected ssl.enabled to be set")
	}
}

func TestParseFlags_SSLVerify(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-verify",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.SSL.Verify != expected {
		t.Errorf("expected %v to be %v", config.SSL.Verify, expected)
	}
	if !config.WasSet("ssl") {
		t.Errorf("expected ssl to be set")
	}
	if !config.WasSet("ssl.verify") {
		t.Errorf("expected ssl.verify to be set")
	}
}

func TestParseFlags_noSSLVerify(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-verify=false",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := false
	if config.SSL.Verify != expected {
		t.Errorf("expected %v to be %v", config.SSL.Verify, expected)
	}
	if !config.WasSet("ssl") {
		t.Errorf("expected ssl to be set")
	}
	if !config.WasSet("ssl.verify") {
		t.Errorf("expected ssl.verify to be set")
	}
}

func TestParseFlags_SSLCert(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-cert", "/path/to/c1.pem",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/c1.pem"
	if config.SSL.Cert != expected {
		t.Errorf("expected %v to be %v", config.SSL.Cert, expected)
	}
	if !config.WasSet("ssl") {
		t.Errorf("expected ssl to be set")
	}
	if !config.WasSet("ssl.cert") {
		t.Errorf("expected ssl.cert to be set")
	}
}

func TestParseFlags_SSLKey(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-key", "/path/to/client-key.pem",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/client-key.pem"
	if config.SSL.Key != expected {
		t.Errorf("expected %v to be %v", config.SSL.Key, expected)
	}
	if !config.WasSet("ssl") {
		t.Errorf("expected ssl to be set")
	}
	if !config.WasSet("ssl.key") {
		t.Errorf("expected ssl.key to be set")
	}
}

func TestParseFlags_SSLCaCert(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-ca-cert", "/path/to/c2.pem",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/c2.pem"
	if config.SSL.CaCert != expected {
		t.Errorf("expected %v to be %v", config.SSL.CaCert, expected)
	}
	if !config.WasSet("ssl") {
		t.Errorf("expected ssl to be set")
	}
	if !config.WasSet("ssl.ca_cert") {
		t.Errorf("expected ssl.ca_cert to be set")
	}
}

func TestParseFlags_SSLCaPath(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-ca-path", "/path/to/certs",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/certs"
	if config.SSL.CaPath != expected {
		t.Errorf("expected %v to be %v", config.SSL.CaPath, expected)
	}
	if !config.WasSet("ssl") {
		t.Errorf("expected ssl to be set")
	}
	if !config.WasSet("ssl.ca_path") {
		t.Errorf("expected ssl.ca_path to be set")
	}
}

func TestParseFlags_maxStale(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-max-stale", "10h",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := 10 * time.Hour
	if config.MaxStale != expected {
		t.Errorf("expected %q to be %q", config.MaxStale, expected)
	}
}

func TestParseFlags_configTemplates(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	conf, _, _, _, err := cli.parseFlags([]string{
		"-template", "in.ctmpl:out.txt:some command",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(conf.ConfigTemplates) != 1 {
		t.Fatal("expected 1 config template")
	}

	expected := &config.ConfigTemplate{
		Source:         "in.ctmpl",
		Destination:    "out.txt",
		Command:        "some command",
		CommandTimeout: config.DefaultCommandTimeout,
		Perms:          config.DefaultFilePerms,
		Wait:           &watch.Wait{},
	}
	if !reflect.DeepEqual(conf.ConfigTemplates[0], expected) {
		t.Errorf("expected %#v to be %#v", conf.ConfigTemplates[0], expected)
	}
}

func TestParseFlags_dedup(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-dedup",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Deduplicate.Enabled != expected {
		t.Errorf("expected %v to be %v", config.Deduplicate.Enabled, expected)
	}
	if !config.WasSet("deduplicate") {
		t.Errorf("expected deduplicate to be set")
	}
	if !config.WasSet("deduplicate.enabled") {
		t.Errorf("expected deduplicate.enabled to be set")
	}
}

func TestParseFlags_exec(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-exec", "/bin/bar",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/bin/bar"
	if config.Exec.Command != "/bin/bar" {
		t.Errorf("expected %v to be %v", config.Exec.Command, expected)
	}
	if !config.WasSet("exec") {
		t.Errorf("expected exec to be set")
	}
	if !config.WasSet("exec.command") {
		t.Errorf("expected exec.command to be set")
	}
}

func TestParseFlags_execKillSignal(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-exec-kill-signal", "SIGINT",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := syscall.SIGINT
	if config.Exec.KillSignal != expected {
		t.Errorf("expected %v to be %v", config.Exec.KillSignal, expected)
	}
	if !config.WasSet("exec.kill_signal") {
		t.Errorf("expected exec.kill_signal to be set")
	}
}

func TestParseFlags_execKillTimeout(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-exec-kill-timeout", "30s",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := 30 * time.Second
	if config.Exec.KillTimeout != expected {
		t.Errorf("expected %v to be %v", config.Exec.KillTimeout, expected)
	}
	if !config.WasSet("exec.kill_timeout") {
		t.Errorf("expected exec.kill_timeout to be set")
	}
}

func TestParseFlags_execReloadSignal(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-exec-reload-signal", "SIGUSR2",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := syscall.SIGUSR2
	if config.Exec.ReloadSignal != expected {
		t.Errorf("expected %v to be %v", config.Exec.ReloadSignal, expected)
	}
	if !config.WasSet("exec.reload_signal") {
		t.Errorf("expected exec.reload_signal to be set")
	}
}

func TestParseFlags_execSplay(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-exec-splay", "100ms",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := 100 * time.Millisecond
	if config.Exec.Splay != expected {
		t.Errorf("expected %v to be %v", config.Exec.Splay, expected)
	}
	if !config.WasSet("exec.splay") {
		t.Errorf("expected exec.splay to be set")
	}
}

func TestParseFlags_syslog(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-syslog",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Syslog.Enabled != expected {
		t.Errorf("expected %v to be %v", config.Syslog.Enabled, expected)
	}
	if !config.WasSet("syslog") {
		t.Errorf("expected syslog to be set")
	}
	if !config.WasSet("syslog.enabled") {
		t.Errorf("expected syslog.enabled to be set")
	}
}

func TestParseFlags_syslogFacility(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-syslog-facility", "LOCAL5",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "LOCAL5"
	if config.Syslog.Facility != expected {
		t.Errorf("expected %v to be %v", config.Syslog.Facility, expected)
	}
	if !config.WasSet("syslog.facility") {
		t.Errorf("expected syslog.facility to be set")
	}
}

func TestParseFlags_vaultAddr(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-addr", "1.2.3.4",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "1.2.3.4"
	if config.Vault.Address != expected {
		t.Errorf("expected %v to be %v", config.Vault.Address, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.address") {
		t.Errorf("expected vault.address to be set")
	}
}

func TestParseFlags_vaultToken(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-token", "abcd1234",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "abcd1234"
	if config.Vault.Token != expected {
		t.Errorf("expected %v to be %v", config.Vault.Token, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.token") {
		t.Errorf("expected vault.token to be set")
	}
}

func TestParseFlags_vaultRenewToken(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-renew-token",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Vault.RenewToken != true {
		t.Errorf("expected %v to be %v", config.Vault.RenewToken, true)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.renew_token") {
		t.Errorf("expected vault.renew_token to be set")
	}
}

func TestParseFlags_vaultUnwrapToken(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-unwrap-token",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Vault.UnwrapToken != true {
		t.Errorf("expected %v to be %v", config.Vault.UnwrapToken, true)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.unwrap_token") {
		t.Errorf("expected vault.unwrap_token to be set")
	}
}

func TestParseFlags_vaultSSL(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-ssl",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Vault.SSL.Enabled != expected {
		t.Errorf("expected %v to be %v", config.Vault.SSL.Enabled, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.ssl") {
		t.Errorf("expected vault.ssl to be set")
	}
	if !config.WasSet("vault.ssl.enabled") {
		t.Errorf("expected vault.ssl.enabled to be set")
	}
}

func TestParseFlags_vaultNoSSL(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-ssl=false",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := false
	if config.Vault.SSL.Enabled != expected {
		t.Errorf("expected %v to be %v", config.Vault.SSL.Enabled, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.ssl") {
		t.Errorf("expected vault.ssl to be set")
	}
	if !config.WasSet("vault.ssl.enabled") {
		t.Errorf("expected vault.ssl.enabled to be set")
	}
}

func TestParseFlags_vaultSSLVerify(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-ssl-verify",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Vault.SSL.Verify != expected {
		t.Errorf("expected %v to be %v", config.Vault.SSL.Verify, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.ssl") {
		t.Errorf("expected vault.ssl to be set")
	}
	if !config.WasSet("vault.ssl.verify") {
		t.Errorf("expected vault.ssl.verify to be set")
	}
}

func TestParseFlags_vaultNoSSLVerify(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-ssl-verify=false",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := false
	if config.Vault.SSL.Verify != expected {
		t.Errorf("expected %v to be %v", config.Vault.SSL.Verify, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.ssl") {
		t.Errorf("expected vault.ssl to be set")
	}
	if !config.WasSet("vault.ssl.verify") {
		t.Errorf("expected vault.ssl.verify to be set")
	}
}

func TestParseFlags_vaultSSLCert(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-ssl-cert", "/path/to/c1.pem",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/c1.pem"
	if config.Vault.SSL.Cert != expected {
		t.Errorf("expected %v to be %v", config.Vault.SSL.Cert, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.ssl") {
		t.Errorf("expected vault.ssl to be set")
	}
	if !config.WasSet("vault.ssl.cert") {
		t.Errorf("expected vault.ssl.cert to be set")
	}
}

func TestParseFlags_vaultSSLKey(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-ssl-key", "/path/to/client-key.pem",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/client-key.pem"
	if config.Vault.SSL.Key != expected {
		t.Errorf("expected %v to be %v", config.Vault.SSL.Key, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.ssl") {
		t.Errorf("expected vault.ssl to be set")
	}
	if !config.WasSet("vault.ssl.key") {
		t.Errorf("expected vault.ssl.key to be set")
	}
}

func TestParseFlags_vaultSSLCaCert(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-ssl-ca-cert", "/path/to/c2.pem",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/c2.pem"
	if config.Vault.SSL.CaCert != expected {
		t.Errorf("expected %v to be %v", config.Vault.SSL.CaCert, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.ssl") {
		t.Errorf("expected vault.ssl to be set")
	}
	if !config.WasSet("vault.ssl.ca_cert") {
		t.Errorf("expected vault.ssl.ca_cert to be set")
	}
}

func TestParseFlags_vaultSSLCaPath(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-vault-ssl-ca-path", "/path/to/certs",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/certs"
	if config.Vault.SSL.CaPath != expected {
		t.Errorf("expected %v to be %v", config.Vault.SSL.CaPath, expected)
	}
	if !config.WasSet("vault") {
		t.Errorf("expected vault to be set")
	}
	if !config.WasSet("vault.ssl") {
		t.Errorf("expected vault.ssl to be set")
	}
	if !config.WasSet("vault.ssl.ca_path") {
		t.Errorf("expected vault.ssl.ca_path to be set")
	}
}

func TestParseFlags_wait(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-wait", "10h:11h",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := &watch.Wait{
		Min: 10 * time.Hour,
		Max: 11 * time.Hour,
	}
	if !reflect.DeepEqual(config.Wait, expected) {
		t.Errorf("expected %v to be %v", config.Wait, expected)
	}
	if !config.WasSet("wait") {
		t.Errorf("expected wait to be set")
	}
}

func TestParseFlags_waitError(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, _, err := cli.parseFlags([]string{
		"-wait", "watermelon:bacon",
	})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "invalid value"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestParseFlags_config(t *testing.T) {
	f, err := ioutil.TempFile("", "parse-flags-config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(`consul = "1.2.3.4"`); err != nil {
		t.Fatal(err)
	}

	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-config", f.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}

	assert := assert.New(t)
	assert.Equal(config.Consul, "1.2.3.4")
	assert.True(config.WasSet("consul"))
}

func TestParseFlags_configMultiple(t *testing.T) {
	f1, err := ioutil.TempFile("", "parse-flags-config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f1.Name())

	if _, err := f1.WriteString(`consul = "1.2.3.4"`); err != nil {
		t.Fatal(err)
	}

	f2, err := ioutil.TempFile("", "parse-flags-config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f2.Name())

	if _, err := f2.WriteString(`token = "abcd1234"`); err != nil {
		t.Fatal(err)
	}

	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-config", f1.Name(),
		"-config", f2.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}

	assert := assert.New(t)
	assert.Equal(config.Consul, "1.2.3.4")
	assert.True(config.WasSet("consul"))
	assert.Equal(config.Token, "abcd1234")
	assert.True(config.WasSet("token"))
}

func TestParseFlags_retry(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-retry", "10h",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := 10 * time.Hour
	if config.Retry != expected {
		t.Errorf("expected %v to be %v", config.Retry, expected)
	}
	if !config.WasSet("retry") {
		t.Errorf("expected retry to be set")
	}
}

func TestParseFlags_logLevel(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-log-level", "debug",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "debug"
	if config.LogLevel != expected {
		t.Errorf("expected %v to be %v", config.LogLevel, expected)
	}
	if !config.WasSet("log_level") {
		t.Errorf("expected log_level to be set")
	}
}

func TestParseFlags_pidFile(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-pid-file", "/path/to/pid",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/pid"
	if config.PidFile != expected {
		t.Errorf("expected %v to be %v", config.PidFile, expected)
	}
	if !config.WasSet("pid_file") {
		t.Errorf("expected pid_file to be set")
	}
}

func TestParseFlags_once(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, once, _, _, err := cli.parseFlags([]string{
		"-once",
	})
	if err != nil {
		t.Fatal(err)
	}

	if once != true {
		t.Errorf("expected once to be true")
	}
}

func TestParseFlags_dry(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, dry, _, err := cli.parseFlags([]string{
		"-dry",
	})
	if err != nil {
		t.Fatal(err)
	}

	if dry != true {
		t.Errorf("expected dry to be true")
	}
}

func TestParseFlags_version(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, version, err := cli.parseFlags([]string{
		"-version",
	})
	if err != nil {
		t.Fatal(err)
	}

	if version != true {
		t.Errorf("expected version to be true")
	}
}

func TestParseFlags_v(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, version, err := cli.parseFlags([]string{
		"-v",
	})
	if err != nil {
		t.Fatal(err)
	}

	if version != true {
		t.Errorf("expected version to be true")
	}
}

func TestParseFlags_errors(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, _, err := cli.parseFlags([]string{
		"-totally", "-not", "-valid",
	})

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}
}

func TestParseFlags_badArgs(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, _, err := cli.parseFlags([]string{
		"foo", "bar",
	})

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}
}

func TestRun_printsErrors(t *testing.T) {
	outStream, errStream := gatedio.NewByteBuffer(), gatedio.NewByteBuffer()
	cli := NewCLI(outStream, errStream)
	args := strings.Split("consul-template -bacon delicious", " ")

	status := cli.Run(args)
	if status == ExitCodeOK {
		t.Fatal("expected not OK exit code")
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

func TestRun_parseError(t *testing.T) {
	outStream, errStream := gatedio.NewByteBuffer(), gatedio.NewByteBuffer()
	cli := NewCLI(outStream, errStream)
	args := strings.Split("consul-template -bacon delicious", " ")

	status := cli.Run(args)
	if status != ExitCodeParseFlagsError {
		t.Errorf("expected %q to eq %q", status, ExitCodeParseFlagsError)
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Fatalf("expected %q to contain %q", errStream.String(), expected)
	}
}

func TestRun_onceFlag(t *testing.T) {
	t.Parallel()

	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	consul.SetKV("foo", []byte("bar"))

	template := test.CreateTempfile([]byte(`
	{{key "foo"}}
  `), t)
	defer test.DeleteTempfile(template, t)

	out := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(out, t)

	outStream := gatedio.NewByteBuffer()
	cli := NewCLI(outStream, outStream)

	command := fmt.Sprintf("consul-template -consul %s -template %s:%s -once -log-level debug",
		consul.HTTPAddr, template.Name(), out.Name())
	args := strings.Split(command, " ")

	ch := make(chan int, 1)
	go func() {
		ch <- cli.Run(args)
	}()

	select {
	case status := <-ch:
		if status != ExitCodeOK {
			t.Errorf("expected %d to eq %d", status, ExitCodeOK)
			t.Errorf("out: %s", outStream.String())
		}
	case <-time.After(2 * time.Second):
		t.Errorf("expected exit, did not exit after 2s")
		t.Errorf("out: %s", outStream.String())
	}
}

func TestReload_sighup(t *testing.T) {
	t.Parallel()

	template := test.CreateTempfile([]byte("initial value"), t)
	defer test.DeleteTempfile(template, t)

	out := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(out, t)

	outStream := gatedio.NewByteBuffer()
	cli := NewCLI(outStream, outStream)

	command := fmt.Sprintf("consul-template -template %s:%s", template.Name(), out.Name())
	args := strings.Split(command, " ")

	go func(args []string) {
		if exit := cli.Run(args); exit != 0 {
			t.Fatalf("bad exit code: %d", exit)
		}
	}(args)
	defer cli.stop()

	// Ensure we have run at least once
	test.WaitForContents(t, 2*time.Second, out.Name(), "initial value")

	newValue := []byte("new value")
	ioutil.WriteFile(template.Name(), newValue, 0644)
	syscall.Kill(syscall.Getpid(), syscall.SIGHUP)

	test.WaitForContents(t, 2*time.Second, out.Name(), "new value")
}

func TestErr_exitStatus(t *testing.T) {
	t.Parallel()

	out := gatedio.NewByteBuffer()
	cli := NewCLI(out, out)

	args := []string{"", "-exec", "bash -c 'exit 123'"}
	exit := cli.Run(args)
	if exit != 123 {
		t.Errorf("expected %d to be %d\n\n%s", exit, 123, out.String())
	}
}
