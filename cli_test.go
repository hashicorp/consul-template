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
	gatedio "github.com/hashicorp/go-gatedio"
)

func TestCLI_ParseFlags(t *testing.T) {
	t.Parallel()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	cases := []struct {
		name string
		f    []string
		e    *config.Config
		err  bool
	}{
		{
			"config",
			[]string{"-config", f.Name()},
			&config.Config{},
			false,
		},
		{
			"consul_addr",
			[]string{"-consul-addr", "1.2.3.4"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Address: config.String("1.2.3.4"),
				},
			},
			false,
		},
		{
			"consul_auth_empty",
			[]string{"-consul-auth", ""},
			nil,
			true,
		},
		{
			"consul_auth_username",
			[]string{"-consul-auth", "username"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Auth: &config.AuthConfig{
						Username: config.String("username"),
					},
				},
			},
			false,
		},
		{
			"consul_auth_username_password",
			[]string{"-consul-auth", "username:password"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Auth: &config.AuthConfig{
						Username: config.String("username"),
						Password: config.String("password"),
					},
				},
			},
			false,
		},
		{
			"consul-retry",
			[]string{"-consul-retry"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul-retry-attempts",
			[]string{"-consul-retry-attempts", "20"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Attempts: config.Int(20),
					},
				},
			},
			false,
		},
		{
			"consul-retry-backoff",
			[]string{"-consul-retry-backoff", "30s"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Backoff: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul-retry-max-backoff",
			[]string{"-consul-retry-max-backoff", "60s"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						MaxBackoff: config.TimeDuration(60 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul-ssl",
			[]string{"-consul-ssl"},
			&config.Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul-ssl-ca-cert",
			[]string{"-consul-ssl-ca-cert", "ca_cert"},
			&config.Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						CaCert: config.String("ca_cert"),
					},
				},
			},
			false,
		},
		{
			"consul-ssl-ca-path",
			[]string{"-consul-ssl-ca-path", "ca_path"},
			&config.Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						CaPath: config.String("ca_path"),
					},
				},
			},
			false,
		},
		{
			"consul-ssl-cert",
			[]string{"-consul-ssl-cert", "cert"},
			&config.Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Cert: config.String("cert"),
					},
				},
			},
			false,
		},
		{
			"consul-ssl-key",
			[]string{"-consul-ssl-key", "key"},
			&config.Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Key: config.String("key"),
					},
				},
			},
			false,
		},
		{
			"consul-ssl-server-name",
			[]string{"-consul-ssl-server-name", "server_name"},
			&config.Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						ServerName: config.String("server_name"),
					},
				},
			},
			false,
		},
		{
			"consul-ssl-verify",
			[]string{"-consul-ssl-verify"},
			&config.Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Verify: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul-token",
			[]string{"-consul-token", "token"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Token: config.String("token"),
				},
			},
			false,
		},
		{
			"consul-transport-dial-keep-alive",
			[]string{"-consul-transport-dial-keep-alive", "30s"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						DialKeepAlive: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul-transport-dial-timeout",
			[]string{"-consul-transport-dial-timeout", "30s"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						DialTimeout: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul-transport-disable-keep-alives",
			[]string{"-consul-transport-disable-keep-alives"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						DisableKeepAlives: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul-transport-max-idle-conns-per-host",
			[]string{"-consul-transport-max-idle-conns-per-host", "100"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						MaxIdleConnsPerHost: config.Int(100),
					},
				},
			},
			false,
		},
		{
			"consul-transport-tls-handshake-timeout",
			[]string{"-consul-transport-tls-handshake-timeout", "30s"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						TLSHandshakeTimeout: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"dedup",
			[]string{"-dedup"},
			&config.Config{
				Dedup: &config.DedupConfig{
					Enabled: config.Bool(true),
				},
			},
			false,
		},
		{
			"exec",
			[]string{"-exec", "command"},
			&config.Config{
				Exec: &config.ExecConfig{
					Enabled: config.Bool(true),
					Command: config.String("command"),
				},
			},
			false,
		},
		{
			"exec-kill-signal",
			[]string{"-exec-kill-signal", "SIGUSR1"},
			&config.Config{
				Exec: &config.ExecConfig{
					KillSignal: config.Signal(syscall.SIGUSR1),
				},
			},
			false,
		},
		{
			"exec-kill-timeout",
			[]string{"-exec-kill-timeout", "10s"},
			&config.Config{
				Exec: &config.ExecConfig{
					KillTimeout: config.TimeDuration(10 * time.Second),
				},
			},
			false,
		},
		{
			"exec-reload-signal",
			[]string{"-exec-reload-signal", "SIGUSR1"},
			&config.Config{
				Exec: &config.ExecConfig{
					ReloadSignal: config.Signal(syscall.SIGUSR1),
				},
			},
			false,
		},
		{
			"exec-splay",
			[]string{"-exec-splay", "10s"},
			&config.Config{
				Exec: &config.ExecConfig{
					Splay: config.TimeDuration(10 * time.Second),
				},
			},
			false,
		},
		{
			"kill-signal",
			[]string{"-kill-signal", "SIGUSR1"},
			&config.Config{
				KillSignal: config.Signal(syscall.SIGUSR1),
			},
			false,
		},
		{
			"log-level",
			[]string{"-log-level", "DEBUG"},
			&config.Config{
				LogLevel: config.String("DEBUG"),
			},
			false,
		},
		{
			"max-stale",
			[]string{"-max-stale", "10s"},
			&config.Config{
				MaxStale: config.TimeDuration(10 * time.Second),
			},
			false,
		},
		{
			"pid-file",
			[]string{"-pid-file", "/var/pid/file"},
			&config.Config{
				PidFile: config.String("/var/pid/file"),
			},
			false,
		},
		{
			"reload-signal",
			[]string{"-reload-signal", "SIGUSR1"},
			&config.Config{
				ReloadSignal: config.Signal(syscall.SIGUSR1),
			},
			false,
		},
		{
			"retry",
			[]string{"-retry", "30s"},
			&config.Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Backoff: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"syslog",
			[]string{"-syslog"},
			&config.Config{
				Syslog: &config.SyslogConfig{
					Enabled: config.Bool(true),
				},
			},
			false,
		},
		{
			"syslog-facility",
			[]string{"-syslog-facility", "LOCAL0"},
			&config.Config{
				Syslog: &config.SyslogConfig{
					Facility: config.String("LOCAL0"),
				},
			},
			false,
		},
		{
			"template",
			[]string{"-template", "/tmp/in.tpl"},
			&config.Config{
				Templates: &config.TemplateConfigs{
					&config.TemplateConfig{
						Source: config.String("/tmp/in.tpl"),
					},
				},
			},
			false,
		},
		{
			"vault-addr",
			[]string{"-vault-addr", "vault_addr"},
			&config.Config{
				Vault: &config.VaultConfig{
					Address: config.String("vault_addr"),
				},
			},
			false,
		},
		{
			"vault-grace",
			[]string{"-vault-grace", "10s"},
			&config.Config{
				Vault: &config.VaultConfig{
					Grace: config.TimeDuration(10 * time.Second),
				},
			},
			false,
		},
		{
			"vault-retry",
			[]string{"-vault-retry"},
			&config.Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"vault-retry-attempts",
			[]string{"-vault-retry-attempts", "20"},
			&config.Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Attempts: config.Int(20),
					},
				},
			},
			false,
		},
		{
			"vault-retry-backoff",
			[]string{"-vault-retry-backoff", "30s"},
			&config.Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Backoff: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-retry-max-backoff",
			[]string{"-vault-retry-max-backoff", "60s"},
			&config.Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						MaxBackoff: config.TimeDuration(60 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-renew-token",
			[]string{"-vault-renew-token"},
			&config.Config{
				Vault: &config.VaultConfig{
					RenewToken: config.Bool(true),
				},
			},
			false,
		},
		{
			"vault-ssl",
			[]string{"-vault-ssl"},
			&config.Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"vault-ssl-ca-cert",
			[]string{"-vault-ssl-ca-cert", "ca_cert"},
			&config.Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						CaCert: config.String("ca_cert"),
					},
				},
			},
			false,
		},
		{
			"vault-ssl-ca-path",
			[]string{"-vault-ssl-ca-path", "ca_path"},
			&config.Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						CaPath: config.String("ca_path"),
					},
				},
			},
			false,
		},
		{
			"vault-ssl-cert",
			[]string{"-vault-ssl-cert", "cert"},
			&config.Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Cert: config.String("cert"),
					},
				},
			},
			false,
		},
		{
			"vault-ssl-key",
			[]string{"-vault-ssl-key", "key"},
			&config.Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Key: config.String("key"),
					},
				},
			},
			false,
		},
		{
			"vault-ssl-server-name",
			[]string{"-vault-ssl-server-name", "server_name"},
			&config.Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						ServerName: config.String("server_name"),
					},
				},
			},
			false,
		},
		{
			"vault-ssl-verify",
			[]string{"-vault-ssl-verify"},
			&config.Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Verify: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"vault-token",
			[]string{"-vault-token", "token"},
			&config.Config{
				Vault: &config.VaultConfig{
					Token: config.String("token"),
				},
			},
			false,
		},
		{
			"vault-transport-dial-keep-alive",
			[]string{"-vault-transport-dial-keep-alive", "30s"},
			&config.Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						DialKeepAlive: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-transport-dial-timeout",
			[]string{"-vault-transport-dial-timeout", "30s"},
			&config.Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						DialTimeout: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-transport-disable-keep-alives",
			[]string{"-vault-transport-disable-keep-alives"},
			&config.Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						DisableKeepAlives: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"vault-transport-max-idle-conns-per-host",
			[]string{"-vault-transport-max-idle-conns-per-host", "100"},
			&config.Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						MaxIdleConnsPerHost: config.Int(100),
					},
				},
			},
			false,
		},
		{
			"vault-transport-tls-handshake-timeout",
			[]string{"-vault-transport-tls-handshake-timeout", "30s"},
			&config.Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						TLSHandshakeTimeout: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-unwrap-token",
			[]string{"-vault-unwrap-token"},
			&config.Config{
				Vault: &config.VaultConfig{
					UnwrapToken: config.Bool(true),
				},
			},
			false,
		},
		{
			"wait_min",
			[]string{"-wait", "10s"},
			&config.Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(10 * time.Second),
					Max: config.TimeDuration(40 * time.Second),
				},
			},
			false,
		},
		{
			"wait_min_max",
			[]string{"-wait", "10s:30s"},
			&config.Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(10 * time.Second),
					Max: config.TimeDuration(30 * time.Second),
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			out := gatedio.NewByteBuffer()
			cli := NewCLI(out, out)

			a, _, _, _, _, err := cli.ParseFlags(tc.f)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if tc.e != nil {
				tc.e = config.DefaultConfig().Merge(tc.e)
			}

			if !reflect.DeepEqual(tc.e, a) {
				t.Errorf("\nexp: %#v\nact: %#v\nout: %q", tc.e, a, out.String())
			}
		})
	}
}

func TestCLI_Run(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		f    func(*testing.T, int, string)
	}{
		{
			"help",
			[]string{"-h"},
			func(t *testing.T, i int, s string) {
				if i != 0 {
					t.Error("expected 0 exit")
				}
			},
		},
		{
			"version",
			[]string{"-v"},
			func(t *testing.T, i int, s string) {
				if i != 0 {
					t.Error("expected 0 exit")
				}
			},
		},
		{
			"too_many_args",
			[]string{"foo", "bar", "baz"},
			func(t *testing.T, i int, s string) {
				if i == 0 {
					t.Error("expected error")
				}
				if !strings.Contains(s, "extra args") {
					t.Errorf("\nexp: %q\nact: %q", "extra args", s)
				}
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			out := gatedio.NewByteBuffer()
			cli := NewCLI(out, out)

			tc.args = append([]string{"consul-template"}, tc.args...)
			exit := cli.Run(tc.args)
			tc.f(t, exit, out.String())
		})
	}

	t.Run("once", func(t *testing.T) {
		t.Parallel()

		f, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		if _, err := f.WriteString(`{{ key "once-foo" }}`); err != nil {
			t.Fatal(err)
		}

		dest, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(dest.Name())

		testConsul.SetKVString(t, "once-foo", "bar")

		out := gatedio.NewByteBuffer()
		cli := NewCLI(out, out)

		ch := make(chan int, 1)
		go func() {
			ch <- cli.Run([]string{"consul-template",
				"-once",
				"-consul-addr", testConsul.HTTPAddr,
				"-vault-renew-token=false",
				"-template", f.Name() + ":" + dest.Name(),
			})
		}()

		select {
		case status := <-ch:
			if status != ExitCodeOK {
				t.Errorf("\nexp: %#v\nact: %#v", status, ExitCodeOK)
			}
			b, err := ioutil.ReadFile(dest.Name())
			if err != nil {
				t.Errorf("\nerror reading file: %s\nout: %s", err, out.String())
			}
			contents := string(b)
			if !strings.Contains("bar", contents) {
				t.Errorf("\nexp: %v\nact: %v\nout: %s", "bar", contents, out.String())
			}
		case <-time.After(2 * time.Second):
			t.Errorf("timeout: %q", out.String())
		}
	})

	t.Run("reload", func(t *testing.T) {
		t.Parallel()

		f, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		if _, err := f.WriteString(`hello`); err != nil {
			t.Fatal(err)
		}

		out := gatedio.NewByteBuffer()
		cli := NewCLI(out, out)
		defer cli.stop()

		ch := make(chan int, 1)
		go func() {
			ch <- cli.Run([]string{"consul-template",
				"-dry",
				"-consul-addr", testConsul.HTTPAddr,
				"-template", f.Name(),
			})
		}()

		// Wait for the file to be available
		test.WaitForContents(t, 2*time.Second, f.Name(), "hello")

		// Write new contents, which wil not be picked up until a reload
		if _, err := f.WriteString(`world`); err != nil {
			t.Fatal(err)
		}

		// Trigger a reload
		cli.signalCh <- syscall.SIGHUP

		// Wait for the file contents
		test.WaitForContents(t, 2*time.Second, f.Name(), "helloworld")

		// We are done now
		cli.stop()

		select {
		case status := <-ch:
			if status != ExitCodeOK {
				t.Errorf("\nexp: %#v\nact: %#v", status, ExitCodeOK)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("timeout: %q", out.String())
		}
	})
}
