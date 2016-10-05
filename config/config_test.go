package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
)

func TestMerge_emptyConfig(t *testing.T) {
	config := DefaultConfig()
	config.Merge(&Config{})

	expected := DefaultConfig()
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_topLevel(t *testing.T) {
	config1 := Must(`
		consul        = "consul-1"
		token         = "token-1"
		reload_signal = "SIGUSR1"
		dump_signal   = "SIGUSR2"
		kill_signal   = "SIGTERM"
		max_stale     = "1s"
		retry         = "1s"
		wait          = "1s"
		pid_file      = "/pid-1"
		log_level     = "log_level-1"
	`)
	config2 := Must(`
		consul        = "consul-2"
		token         = "token-2"
		reload_signal = "SIGINT"
		dump_signal   = "SIGQUIT"
		kill_signal   = "SIGKILL"
		max_stale     = "2s"
		retry         = "2s"
		wait          = "2s"
		pid_file      = "/pid-2"
		log_level     = "log_level-2"
	`)
	config1.Merge(config2)

	if !reflect.DeepEqual(config1, config2) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config1, config2)
	}
}

func TestMerge_deduplicate(t *testing.T) {
	config := Must(`
		deduplicate {
			prefix  = "foobar/"
			enabled = true
		}
	`)
	config.Merge(Must(`
		deduplicate {
			prefix  = "abc/"
			enabled = true
		}
	`))

	expected := &DeduplicateConfig{
		Prefix:  "abc/",
		Enabled: true,
		TTL:     15 * time.Second,
	}

	if !reflect.DeepEqual(config.Deduplicate, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Deduplicate, expected)
	}
}

func TestMerge_vault(t *testing.T) {
	config := Must(`
		vault {
			address = "1.1.1.1"
			token = "1"
			unwrap_token = true
			renew = true
		}
	`)
	config.Merge(Must(`
		vault {
			address = "2.2.2.2"
			renew = false
		}
	`))

	expected := &VaultConfig{
		Address:     "2.2.2.2",
		Token:       "1",
		UnwrapToken: true,
		RenewToken:  false,
		SSL: &SSLConfig{
			Enabled: true,
			Verify:  true,
			Cert:    "",
			CaCert:  "",
		},
	}

	if !reflect.DeepEqual(config.Vault, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Vault, expected)
	}
}

func TestMerge_vaultSSL(t *testing.T) {
	config := Must(`
		vault {
			ssl {
				enabled = true
				verify = true
				cert = "1.pem"
				ca_cert = "ca-1.pem"
			}
		}
	`)
	config.Merge(Must(`
		vault {
			ssl {
				enabled = false
			}
		}
	`))

	expected := &VaultConfig{
		SSL: &SSLConfig{
			Enabled: false,
			Verify:  true,
			Cert:    "1.pem",
			CaCert:  "ca-1.pem",
		},
	}

	if !reflect.DeepEqual(config.Vault.SSL, expected.SSL) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Vault.SSL, expected.SSL)
	}
}

func TestMerge_auth(t *testing.T) {
	config := Must(`
		auth {
			enabled = true
			username = "1"
			password = "1"
		}
	`)
	config.Merge(Must(`
		auth {
			password = "2"
		}
	`))

	expected := &AuthConfig{
		Enabled:  true,
		Username: "1",
		Password: "2",
	}

	if !reflect.DeepEqual(config.Auth, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Auth, expected)
	}
}

func TestMerge_SSL(t *testing.T) {
	config := Must(`
		ssl {
			enabled = true
			verify  = true
			cert    = "1.pem"
			ca_cert = "ca-1.pem"
		}
	`)
	config.Merge(Must(`
		ssl {
			enabled = false
		}
	`))

	expected := &SSLConfig{
		Enabled: false,
		Verify:  true,
		Cert:    "1.pem",
		CaCert:  "ca-1.pem",
	}

	if !reflect.DeepEqual(config.SSL, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.SSL, expected)
	}
}

func TestMerge_Exec(t *testing.T) {
	config := Must(`
		exec {
			command      = "a"
			splay        = "100s"
			kill_signal  = "SIGUSR2"
			kill_timeout = "10s"
		}
	`)
	config.Merge(Must(`
		exec {
			command = "b"
			splay   = "50s"
		}
	`))

	expected := &ExecConfig{
		Command:     "b",
		Splay:       50 * time.Second,
		KillSignal:  syscall.SIGUSR2,
		KillTimeout: 10 * time.Second,
	}

	if !reflect.DeepEqual(config.Exec, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Exec, expected)
	}
}

func TestMerge_syslog(t *testing.T) {
	config := Must(`
		syslog {
			enabled  = true
			facility = "1"
		}
	`)
	config.Merge(Must(`
		syslog {
			facility = "2"
		}
	`))

	expected := &SyslogConfig{
		Enabled:  true,
		Facility: "2",
	}

	if !reflect.DeepEqual(config.Syslog, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Syslog, expected)
	}
}

func TestMerge_configTemplates(t *testing.T) {
	config := Must(`
		template {
			source          = "1"
			destination     = "1"
			contents        = "foo"
			command         = "1"
			command_timeout = "60s"
			perms           = 0600
			backup          = false
			left_delimiter  = "<%"
			right_delimiter = "%>"
		}
	`)
	config.Merge(Must(`
		template {
			source          = "2"
			destination     = "2"
			contents        = "bar"
			command         = "2"
			command_timeout = "2h"
			perms           = 0755
			backup          = true
			wait            = "6s"
		}
	`))

	expected := []*ConfigTemplate{
		&ConfigTemplate{
			Source:           "1",
			Destination:      "1",
			EmbeddedTemplate: "foo",
			Command:          "1",
			CommandTimeout:   60 * time.Second,
			Perms:            0600,
			Backup:           false,
			LeftDelim:        "<%",
			RightDelim:       "%>",
			Wait:             &watch.Wait{},
		},
		&ConfigTemplate{
			Source:           "2",
			Destination:      "2",
			EmbeddedTemplate: "bar",
			Command:          "2",
			CommandTimeout:   2 * time.Hour,
			Perms:            0755,
			Backup:           true,
			Wait: &watch.Wait{
				Min: 6 * time.Second,
				Max: 24 * time.Second,
			},
		},
	}

	if !reflect.DeepEqual(config.ConfigTemplates, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.ConfigTemplates[0], expected[0])
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.ConfigTemplates[1], expected[1])
	}
}

func TestMerge_wait(t *testing.T) {
	config := Must(`
		wait = "1s:1s"
	`)
	config.Merge(Must(`
		wait = "2s:2s"
	`))

	expected := &watch.Wait{
		Min: 2 * time.Second,
		Max: 2 * time.Second,
	}

	if !reflect.DeepEqual(config.Wait, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Wait, expected)
	}
}

func TestParse_emptySignal(t *testing.T) {
	config := Must(`
		reload_signal = ""
	`)

	if config.ReloadSignal != nil {
		t.Errorf("expected %#v to be %#v", config.ReloadSignal, nil)
	}
}

// There is a custom mapstructure function that tests this as well, so this is
// more of an integration test to ensure we are parsing permissions correctly.
func TestParse_jsonFilePerms(t *testing.T) {
	config := Must(`
		{
			"template": {
				"perms": "0600"
			}
		}
	`)

	if len(config.ConfigTemplates) != 1 {
		t.Fatalf("expected %d to be %d", len(config.ConfigTemplates), 1)
	}

	tpl := config.ConfigTemplates[0]
	expected := os.FileMode(0600)
	if tpl.Perms != expected {
		t.Errorf("expected %q to to be %q", tpl.Perms, expected)
	}
}

func TestParse_hclFilePerms(t *testing.T) {
	config := Must(`
		template {
			perms = 0600
		}

		template {
			perms = "0600"
		}
	`)

	if len(config.ConfigTemplates) != 2 {
		t.Fatalf("expected %d to be %d", len(config.ConfigTemplates), 1)
	}

	expected := os.FileMode(0600)

	for i, tpl := range config.ConfigTemplates {
		if tpl.Perms != expected {
			t.Errorf("case %d: expected %q to be %q", i, tpl.Perms, expected)
		}
	}
}

func TestFromFile_readFileError(t *testing.T) {
	_, err := FromFile(path.Join(os.TempDir(), "config.json"))
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "no such file or directory"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParse_vaultDeprecation(t *testing.T) {
	config := Must(`
		vault {
			address = "vault.service.consul"
			token   = "abcd1234"

			// "renew" is renamed to "renew_token"
			renew = true
		}
	`)

	if config.Vault.RenewToken != true {
		t.Errorf("expected renew to be true")
	}
}

func TestParse_correctValues(t *testing.T) {
	config := Must(`
		consul        = "nyc1.demo.consul.io"
		max_stale     = "5s"
		token         = "abcd1234"
		reload_signal = "SIGUSR1"
		dump_signal   = "SIGUSR2"
		kill_signal   = "SIGTERM"
		wait          = "5s:10s"
		retry         = "10s"
		pid_file      = "/var/run/ct"
		log_level     = "warn"

		vault {
			address      = "vault.service.consul"
			token        = "efgh5678"
			renew_token  = true
			unwrap_token = true
			ssl {
				enabled = false
			}
		}

		auth {
			enabled  = true
			username = "test"
			password = "test"
		}

		ssl {
			enabled = true
			verify  = false
			cert    = "c1.pem"
			ca_cert = "c2.pem"
		}

		syslog {
			enabled  = true
			facility = "LOCAL5"
		}

		exec {
			reload_signal = "SIGUSR1"
			kill_signal   = "SIGUSR2"
			kill_timeout  = "100ms"
		}

		template {
			source      = "nginx.conf.ctmpl"
			destination = "/etc/nginx/nginx.conf"
		}

		template {
			source          = "redis.conf.ctmpl"
			destination     = "/etc/redis/redis.conf"
			command         = "service redis restart"
			command_timeout = "60s"
			perms           = 0755
			wait            = "3s:7s"
		}

		template {
			contents    = "foo"
			destination = "embedded.conf"
		}

		deduplicate {
			prefix  = "my-prefix/"
			enabled = true
		}
  `)

	expected := &Config{
		PidFile:      "/var/run/ct",
		Consul:       "nyc1.demo.consul.io",
		ReloadSignal: syscall.SIGUSR1,
		DumpSignal:   syscall.SIGUSR2,
		KillSignal:   syscall.SIGTERM,
		MaxStale:     time.Second * 5,
		Vault: &VaultConfig{
			Address:     "vault.service.consul",
			Token:       "efgh5678",
			RenewToken:  true,
			UnwrapToken: true,
			SSL: &SSLConfig{
				Enabled: false,
				Verify:  true,
				Cert:    "",
				CaCert:  "",
			},
		},
		Auth: &AuthConfig{
			Enabled:  true,
			Username: "test",
			Password: "test",
		},
		SSL: &SSLConfig{
			Enabled: true,
			Verify:  false,
			Cert:    "c1.pem",
			CaCert:  "c2.pem",
		},
		Syslog: &SyslogConfig{
			Enabled:  true,
			Facility: "LOCAL5",
		},
		Token: "abcd1234",
		Wait: &watch.Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		Exec: &ExecConfig{
			ReloadSignal: syscall.SIGUSR1,
			KillSignal:   syscall.SIGUSR2,
			KillTimeout:  100 * time.Millisecond,
		},
		Retry:    10 * time.Second,
		LogLevel: "warn",
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:         "nginx.conf.ctmpl",
				Destination:    "/etc/nginx/nginx.conf",
				CommandTimeout: DefaultCommandTimeout,
				Perms:          0644,
				Wait:           &watch.Wait{},
			},
			&ConfigTemplate{
				Source:         "redis.conf.ctmpl",
				Destination:    "/etc/redis/redis.conf",
				Command:        "service redis restart",
				CommandTimeout: 60 * time.Second,
				Perms:          0755,
				Wait: &watch.Wait{
					Min: 3 * time.Second,
					Max: 7 * time.Second,
				},
			},
			&ConfigTemplate{
				EmbeddedTemplate: "foo",
				Destination:      "embedded.conf",
				CommandTimeout:   DefaultCommandTimeout,
				Perms:            0644,
				Wait:             &watch.Wait{},
			},
		},
		Deduplicate: &DeduplicateConfig{
			Prefix:  "my-prefix/",
			Enabled: true,
			TTL:     15 * time.Second,
		},
		setKeys: config.setKeys,
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected \n%#v\n to be \n%#v\n", config, expected)
	}
}

func TestParse_mapstructureError(t *testing.T) {
	_, err := Parse("consul = true")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "unconvertible type 'bool'"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParse_ssh_key_should_enable_ssl(t *testing.T) {
	config := Must(`
		ssl {
			key     = "private-key.pem"
			verify  = true
			cert    = "1.pem"
			ca_cert = "ca-1.pem"
		}
	`)

	expected := &SSLConfig{
		Enabled: true,
		Verify:  true,
		Cert:    "1.pem",
		Key:     "private-key.pem",
		CaCert:  "ca-1.pem",
	}

	if !reflect.DeepEqual(config.SSL, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.SSL, expected)
	}
}

func TestParse_extraKeys(t *testing.T) {
	_, err := Parse(`
		fake_key         = "nope"
		another_fake_key = "never"
	`)
	if err == nil {
		t.Fatal("expected error")
	}

	expected := "invalid keys: another_fake_key, fake_key"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestParse_parseMaxStaleError(t *testing.T) {
	_, err := Parse(`
		max_stale = "bacon pants"
	`)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParse_parseRetryError(t *testing.T) {
	_, err := Parse(`
		retry = "bacon pants"
	`)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParse_parseWaitError(t *testing.T) {
	_, err := Parse(`
		wait = "not_valid:duration"
	`)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestFromPath_singleFile(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		consul = "127.0.0.1"
	`), t)
	defer test.DeleteTempfile(configFile, t)

	config, err := FromPath(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "127.0.0.1"
	if config.Consul != expected {
		t.Errorf("expected %q to be %q", config.Consul, expected)
	}
}

func TestFromPath_NonExistentDirectory(t *testing.T) {
	// Create a directory and then delete it
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(configDir); err != nil {
		t.Fatal(err)
	}

	_, err = FromPath(configDir)
	if err == nil {
		t.Fatalf("expected error, but nothing was returned")
	}

	expected := "missing file/folder"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestFromPath_EmptyDirectory(t *testing.T) {
	// Create a directory with no files
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(configDir)

	_, err = FromPath(configDir)
	if err != nil {
		t.Fatalf("empty directories are allowed")
	}
}

func TestFromPath_configDir(t *testing.T) {
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	configFile1, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	config1 := []byte(`
		consul = "127.0.0.1:8500"
	`)
	_, err = configFile1.Write(config1)
	if err != nil {
		t.Fatal(err)
	}
	configFile2, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	config2 := []byte(`
		template {
		  source = "/path/on/disk/to/template"
		  destination = "/path/on/disk/where/template/will/render"
		  command = "optional command to run when the template is updated"
		}
	`)
	_, err = configFile2.Write(config2)
	if err != nil {
		t.Fatal(err)
	}

	config, err := FromPath(configDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedConfig := Config{
		Consul: "127.0.0.1:8500",
		ConfigTemplates: []*ConfigTemplate{{
			Source:      "/path/on/disk/to/template",
			Destination: "/path/on/disk/where/template/will/render",
			Command:     "optional command to run when the template is updated",
		}},
	}
	if expectedConfig.Consul != config.Consul {
		t.Fatalf("Config files failed to combine. Expected Consul to be %s but got %s", expectedConfig.Consul, config.Consul)
	}
	if len(config.ConfigTemplates) != len(expectedConfig.ConfigTemplates) {
		t.Fatalf("Expected %d ConfigTemplate but got %d", len(expectedConfig.ConfigTemplates), len(config.ConfigTemplates))
	}
	for i, expectTemplate := range expectedConfig.ConfigTemplates {
		actualTemplate := config.ConfigTemplates[i]
		if actualTemplate.Source != expectTemplate.Source {
			t.Fatalf("Expected template Source to be %s but got %s", expectTemplate.Source, actualTemplate.Source)
		}
		if actualTemplate.Destination != expectTemplate.Destination {
			t.Fatalf("Expected template Destination to be %s but got %s", expectTemplate.Destination, actualTemplate.Destination)
		}
		if actualTemplate.Command != expectTemplate.Command {
			t.Fatalf("Expected template Command to be %s but got %s", expectTemplate.Command, actualTemplate.Command)
		}
	}
}

func TestAuthString_disabled(t *testing.T) {
	a := &AuthConfig{Enabled: false}
	expected := ""
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabledNoPassword(t *testing.T) {
	a := &AuthConfig{Enabled: true, Username: "username"}
	expected := "username"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabled(t *testing.T) {
	a := &AuthConfig{Enabled: true, Username: "username", Password: "password"}
	expected := "username:password"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestParseConfigTemplate_emptyStringArgs(t *testing.T) {
	_, err := ParseConfigTemplate("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify empty template declaration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigTemplate_stringWithSpacesArgs(t *testing.T) {
	_, err := ParseConfigTemplate("  ")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify empty template declaration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigurationTemplate_tooManyArgs(t *testing.T) {
	_, err := ParseConfigTemplate("foo:bar:blitz:baz")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "invalid template declaration format"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigurationTemplate_windowsDrives(t *testing.T) {
	ct, err := ParseConfigTemplate(`C:\abc\123:D:\xyz\789:some command`)
	if err != nil {
		t.Fatalf("failed parsing windows drive letters: %s", err)
	}

	expected := &ConfigTemplate{
		Source:         `C:\abc\123`,
		Destination:    `D:\xyz\789`,
		Command:        "some command",
		CommandTimeout: DefaultCommandTimeout,
		Perms:          0644,
		Wait:           &watch.Wait{},
	}

	if !reflect.DeepEqual(ct, expected) {
		t.Fatalf("unexpected result parsing windows drives: %#v", ct)
	}
}

func TestParseConfigurationTemplate_source(t *testing.T) {
	source := "/tmp/config.ctmpl"
	template, err := ParseConfigTemplate(source)
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}
}

func TestParseConfigurationTemplate_destination(t *testing.T) {
	source, destination := "/tmp/config.ctmpl", "/tmp/out"
	template, err := ParseConfigTemplate(fmt.Sprintf("%s:%s", source, destination))
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}

	if template.Destination != destination {
		t.Errorf("expected %q to equal %q", template.Destination, destination)
	}
}

func TestParseConfigurationTemplate_command(t *testing.T) {
	source, destination, command := "/tmp/config.ctmpl", "/tmp/out", "reboot"
	template, err := ParseConfigTemplate(fmt.Sprintf("%s:%s:%s", source, destination, command))
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}

	if template.Destination != destination {
		t.Errorf("expected %q to equal %q", template.Destination, destination)
	}

	if template.Command != command {
		t.Errorf("expected %q to equal %q", template.Command, command)
	}
}
