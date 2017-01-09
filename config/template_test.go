package config

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestTemplateConfig_Copy(t *testing.T) {
	cases := []struct {
		name string
		a    *TemplateConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&TemplateConfig{},
		},
		{
			"same_enabled",
			&TemplateConfig{
				Backup:         Bool(true),
				Command:        String("command"),
				CommandTimeout: TimeDuration(10 * time.Second),
				Contents:       String("contents"),
				Destination:    String("destination"),
				Exec:           &ExecConfig{Command: String("command")},
				Perms:          FileMode(0600),
				Source:         String("source"),
				Wait:           &WaitConfig{Min: TimeDuration(10)},
				LeftDelim:      String("left_delim"),
				RightDelim:     String("right_delim"),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Copy()
			if !reflect.DeepEqual(tc.a, r) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.a, r)
			}
		})
	}
}

func TestTemplateConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *TemplateConfig
		b    *TemplateConfig
		r    *TemplateConfig
	}{
		{
			"nil_a",
			nil,
			&TemplateConfig{},
			&TemplateConfig{},
		},
		{
			"nil_b",
			&TemplateConfig{},
			nil,
			&TemplateConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TemplateConfig{},
			&TemplateConfig{},
			&TemplateConfig{},
		},
		{
			"backup_overrides",
			&TemplateConfig{Backup: Bool(true)},
			&TemplateConfig{Backup: Bool(false)},
			&TemplateConfig{Backup: Bool(false)},
		},
		{
			"backup_empty_one",
			&TemplateConfig{Backup: Bool(true)},
			&TemplateConfig{},
			&TemplateConfig{Backup: Bool(true)},
		},
		{
			"backup_empty_two",
			&TemplateConfig{},
			&TemplateConfig{Backup: Bool(true)},
			&TemplateConfig{Backup: Bool(true)},
		},
		{
			"backup_same",
			&TemplateConfig{Backup: Bool(true)},
			&TemplateConfig{Backup: Bool(true)},
			&TemplateConfig{Backup: Bool(true)},
		},
		{
			"command_overrides",
			&TemplateConfig{Command: String("command")},
			&TemplateConfig{Command: String("")},
			&TemplateConfig{Command: String("")},
		},
		{
			"command_empty_one",
			&TemplateConfig{Command: String("command")},
			&TemplateConfig{},
			&TemplateConfig{Command: String("command")},
		},
		{
			"command_empty_two",
			&TemplateConfig{},
			&TemplateConfig{Command: String("command")},
			&TemplateConfig{Command: String("command")},
		},
		{
			"command_same",
			&TemplateConfig{Command: String("command")},
			&TemplateConfig{Command: String("command")},
			&TemplateConfig{Command: String("command")},
		},
		{
			"command_timeout_overrides",
			&TemplateConfig{CommandTimeout: TimeDuration(10 * time.Second)},
			&TemplateConfig{CommandTimeout: TimeDuration(0 * time.Second)},
			&TemplateConfig{CommandTimeout: TimeDuration(0 * time.Second)},
		},
		{
			"command_timeout_empty_one",
			&TemplateConfig{CommandTimeout: TimeDuration(10 * time.Second)},
			&TemplateConfig{},
			&TemplateConfig{CommandTimeout: TimeDuration(10 * time.Second)},
		},
		{
			"command_timeout_empty_two",
			&TemplateConfig{},
			&TemplateConfig{CommandTimeout: TimeDuration(10 * time.Second)},
			&TemplateConfig{CommandTimeout: TimeDuration(10 * time.Second)},
		},
		{
			"command_timeout_same",
			&TemplateConfig{CommandTimeout: TimeDuration(10 * time.Second)},
			&TemplateConfig{CommandTimeout: TimeDuration(10 * time.Second)},
			&TemplateConfig{CommandTimeout: TimeDuration(10 * time.Second)},
		},
		{
			"contents_overrides",
			&TemplateConfig{Contents: String("contents")},
			&TemplateConfig{Contents: String("")},
			&TemplateConfig{Contents: String("")},
		},
		{
			"contents_empty_one",
			&TemplateConfig{Contents: String("contents")},
			&TemplateConfig{},
			&TemplateConfig{Contents: String("contents")},
		},
		{
			"contents_empty_two",
			&TemplateConfig{},
			&TemplateConfig{Contents: String("contents")},
			&TemplateConfig{Contents: String("contents")},
		},
		{
			"contents_same",
			&TemplateConfig{Contents: String("contents")},
			&TemplateConfig{Contents: String("contents")},
			&TemplateConfig{Contents: String("contents")},
		},
		{
			"destination_overrides",
			&TemplateConfig{Destination: String("destination")},
			&TemplateConfig{Destination: String("")},
			&TemplateConfig{Destination: String("")},
		},
		{
			"destination_empty_one",
			&TemplateConfig{Destination: String("destination")},
			&TemplateConfig{},
			&TemplateConfig{Destination: String("destination")},
		},
		{
			"destination_empty_two",
			&TemplateConfig{},
			&TemplateConfig{Destination: String("destination")},
			&TemplateConfig{Destination: String("destination")},
		},
		{
			"destination_same",
			&TemplateConfig{Destination: String("destination")},
			&TemplateConfig{Destination: String("destination")},
			&TemplateConfig{Destination: String("destination")},
		},
		{
			"exec_overrides",
			&TemplateConfig{Exec: &ExecConfig{Command: String("command")}},
			&TemplateConfig{Exec: &ExecConfig{Command: String("")}},
			&TemplateConfig{Exec: &ExecConfig{Command: String("")}},
		},
		{
			"exec_empty_one",
			&TemplateConfig{Exec: &ExecConfig{Command: String("command")}},
			&TemplateConfig{Exec: &ExecConfig{}},
			&TemplateConfig{Exec: &ExecConfig{Command: String("command")}},
		},
		{
			"exec_empty_two",
			&TemplateConfig{Exec: &ExecConfig{}},
			&TemplateConfig{Exec: &ExecConfig{Command: String("command")}},
			&TemplateConfig{Exec: &ExecConfig{Command: String("command")}},
		},
		{
			"exec_same",
			&TemplateConfig{Exec: &ExecConfig{Command: String("command")}},
			&TemplateConfig{Exec: &ExecConfig{Command: String("command")}},
			&TemplateConfig{Exec: &ExecConfig{Command: String("command")}},
		},
		{
			"perms_overrides",
			&TemplateConfig{Perms: FileMode(0600)},
			&TemplateConfig{Perms: FileMode(0000)},
			&TemplateConfig{Perms: FileMode(0000)},
		},
		{
			"perms_empty_one",
			&TemplateConfig{Perms: FileMode(0600)},
			&TemplateConfig{},
			&TemplateConfig{Perms: FileMode(0600)},
		},
		{
			"perms_empty_two",
			&TemplateConfig{},
			&TemplateConfig{Perms: FileMode(0600)},
			&TemplateConfig{Perms: FileMode(0600)},
		},
		{
			"perms_same",
			&TemplateConfig{Perms: FileMode(0600)},
			&TemplateConfig{Perms: FileMode(0600)},
			&TemplateConfig{Perms: FileMode(0600)},
		},
		{
			"source_overrides",
			&TemplateConfig{Source: String("source")},
			&TemplateConfig{Source: String("")},
			&TemplateConfig{Source: String("")},
		},
		{
			"source_empty_one",
			&TemplateConfig{Source: String("source")},
			&TemplateConfig{},
			&TemplateConfig{Source: String("source")},
		},
		{
			"source_empty_two",
			&TemplateConfig{},
			&TemplateConfig{Source: String("source")},
			&TemplateConfig{Source: String("source")},
		},
		{
			"source_same",
			&TemplateConfig{Source: String("source")},
			&TemplateConfig{Source: String("source")},
			&TemplateConfig{Source: String("source")},
		},
		{
			"wait_overrides",
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(10)}},
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(0)}},
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(0)}},
		},
		{
			"wait_empty_one",
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(10)}},
			&TemplateConfig{Wait: &WaitConfig{}},
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(10)}},
		},
		{
			"wait_empty_two",
			&TemplateConfig{Wait: &WaitConfig{}},
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(10)}},
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(10)}},
		},
		{
			"wait_same",
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(10)}},
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(10)}},
			&TemplateConfig{Wait: &WaitConfig{Min: TimeDuration(10)}},
		},
		{
			"left_delim_overrides",
			&TemplateConfig{LeftDelim: String("left_delim")},
			&TemplateConfig{LeftDelim: String("")},
			&TemplateConfig{LeftDelim: String("")},
		},
		{
			"left_delim_empty_one",
			&TemplateConfig{LeftDelim: String("left_delim")},
			&TemplateConfig{},
			&TemplateConfig{LeftDelim: String("left_delim")},
		},
		{
			"left_delim_empty_two",
			&TemplateConfig{},
			&TemplateConfig{LeftDelim: String("left_delim")},
			&TemplateConfig{LeftDelim: String("left_delim")},
		},
		{
			"left_delim_same",
			&TemplateConfig{LeftDelim: String("left_delim")},
			&TemplateConfig{LeftDelim: String("left_delim")},
			&TemplateConfig{LeftDelim: String("left_delim")},
		},
		{
			"right_delim_overrides",
			&TemplateConfig{RightDelim: String("right_delim")},
			&TemplateConfig{RightDelim: String("")},
			&TemplateConfig{RightDelim: String("")},
		},
		{
			"right_delim_empty_one",
			&TemplateConfig{RightDelim: String("right_delim")},
			&TemplateConfig{},
			&TemplateConfig{RightDelim: String("right_delim")},
		},
		{
			"right_delim_empty_two",
			&TemplateConfig{},
			&TemplateConfig{RightDelim: String("right_delim")},
			&TemplateConfig{RightDelim: String("right_delim")},
		},
		{
			"right_delim_same",
			&TemplateConfig{RightDelim: String("right_delim")},
			&TemplateConfig{RightDelim: String("right_delim")},
			&TemplateConfig{RightDelim: String("right_delim")},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			if !reflect.DeepEqual(tc.r, r) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, r)
			}
		})
	}
}

func TestTemplateConfig_Finalize(t *testing.T) {
	cases := []struct {
		name string
		i    *TemplateConfig
		r    *TemplateConfig
	}{
		{
			"empty",
			&TemplateConfig{},
			&TemplateConfig{
				Backup:         Bool(false),
				Command:        String(""),
				CommandTimeout: TimeDuration(DefaultTemplateCommandTimeout),
				Contents:       String(""),
				Destination:    String(""),
				Exec: &ExecConfig{
					Command: String(""),
					Enabled: Bool(false),
					Env: &EnvConfig{
						Blacklist: []string{},
						Custom:    []string{},
						Pristine:  Bool(false),
						Whitelist: []string{},
					},
					KillSignal:   Signal(DefaultExecKillSignal),
					KillTimeout:  TimeDuration(DefaultExecKillTimeout),
					ReloadSignal: Signal(DefaultExecReloadSignal),
					Splay:        TimeDuration(0 * time.Second),
					Timeout:      TimeDuration(DefaultTemplateCommandTimeout),
				},
				Perms:  FileMode(DefaultTemplateFilePerms),
				Source: String(""),
				Wait: &WaitConfig{
					Enabled: Bool(false),
					Max:     TimeDuration(0 * time.Second),
					Min:     TimeDuration(0 * time.Second),
				},
				LeftDelim:  String(""),
				RightDelim: String(""),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize()
			if !reflect.DeepEqual(tc.r, tc.i) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, tc.i)
			}
		})
	}
}

func TestTemplateConfig_Display(t *testing.T) {
	cases := []struct {
		name string
		c    *TemplateConfig
		e    string
	}{
		{
			"nil",
			nil,
			"",
		},
		{
			"with_source",
			&TemplateConfig{
				Source: String("/var/my.tpl"),
			},
			`"/var/my.tpl" => ""`,
		},
		{
			"with_contents",
			&TemplateConfig{
				Contents: String("hello"),
			},
			`"(dynamic)" => ""`,
		},
		{
			"with_destination",
			&TemplateConfig{
				Source:      String("/var/my.tpl"),
				Destination: String("/var/my.txt"),
			},
			`"/var/my.tpl" => "/var/my.txt"`,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			a := tc.c.Display()
			if tc.e != a {
				t.Errorf("\nexp: %#v\nact: %#v", tc.e, a)
			}
		})
	}
}

func TestParseTemplateConfig(t *testing.T) {
	cases := []struct {
		name string
		i    string
		e    *TemplateConfig
		err  bool
	}{
		{
			"empty",
			"",
			nil,
			true,
		},
		{
			"empty_with_spaces",
			" ",
			nil,
			true,
		},
		{
			"too_many_args",
			"foo:bar:zip:zap",
			nil,
			true,
		},
		{
			"default",
			"/tmp/a.txt:/tmp/b.txt:command",
			&TemplateConfig{
				Source:      String("/tmp/a.txt"),
				Destination: String("/tmp/b.txt"),
				Command:     String("command"),
			},
			false,
		},
		{
			"windows_drives",
			`C:\abc\123:D:\xyz\789:command`,
			&TemplateConfig{
				Source:      String(`C:\abc\123`),
				Destination: String(`D:\xyz\789`),
				Command:     String(`command`),
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			c, err := ParseTemplateConfig(tc.i)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.e, c) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.e, c)
			}
		})
	}
}
