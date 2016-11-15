package config

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestEnvConfig_Copy(t *testing.T) {
	cases := []struct {
		name string
		a    *EnvConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&EnvConfig{},
		},
		{
			"copy",
			&EnvConfig{
				Blacklist: []string{"blacklist"},
				Custom:    []string{"custom"},
				Pristine:  Bool(true),
				Whitelist: []string{"whitelist"},
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

func TestEnvConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *EnvConfig
		b    *EnvConfig
		r    *EnvConfig
	}{
		{
			"nil_a",
			nil,
			&EnvConfig{},
			&EnvConfig{},
		},
		{
			"nil_b",
			&EnvConfig{},
			nil,
			&EnvConfig{},
		},
		{
			"nil_both",
			&EnvConfig{},
			nil,
			&EnvConfig{},
		},
		{
			"empty_a",
			&EnvConfig{},
			&EnvConfig{Pristine: Bool(true)},
			&EnvConfig{Pristine: Bool(true)},
		},
		{
			"empty_b",
			&EnvConfig{Pristine: Bool(true)},
			&EnvConfig{},
			&EnvConfig{Pristine: Bool(true)},
		},
		{
			"empty_both",
			&EnvConfig{},
			&EnvConfig{},
			&EnvConfig{},
		},
		{
			"blacklist_appends",
			&EnvConfig{Blacklist: []string{"blacklist"}},
			&EnvConfig{Blacklist: []string{"different"}},
			&EnvConfig{Blacklist: []string{"blacklist", "different"}},
		},
		{
			"blacklist_empty_one",
			&EnvConfig{Blacklist: []string{"blacklist"}},
			&EnvConfig{},
			&EnvConfig{Blacklist: []string{"blacklist"}},
		},
		{
			"blacklist_empty_two",
			&EnvConfig{},
			&EnvConfig{Blacklist: []string{"blacklist"}},
			&EnvConfig{Blacklist: []string{"blacklist"}},
		},
		{
			"custom_appends",
			&EnvConfig{Custom: []string{"custom"}},
			&EnvConfig{Custom: []string{"different"}},
			&EnvConfig{Custom: []string{"custom", "different"}},
		},
		{
			"custom_empty_one",
			&EnvConfig{Custom: []string{"custom"}},
			&EnvConfig{},
			&EnvConfig{Custom: []string{"custom"}},
		},
		{
			"custom_empty_two",
			&EnvConfig{},
			&EnvConfig{Custom: []string{"custom"}},
			&EnvConfig{Custom: []string{"custom"}},
		},
		{
			"pristine_overrides",
			&EnvConfig{Pristine: Bool(true)},
			&EnvConfig{Pristine: Bool(false)},
			&EnvConfig{Pristine: Bool(false)},
		},
		{
			"pristine_empty_one",
			&EnvConfig{Pristine: Bool(true)},
			&EnvConfig{},
			&EnvConfig{Pristine: Bool(true)},
		},
		{
			"pristine_empty_two",
			&EnvConfig{},
			&EnvConfig{Pristine: Bool(true)},
			&EnvConfig{Pristine: Bool(true)},
		},
		{
			"pristine_same",
			&EnvConfig{Pristine: Bool(true)},
			&EnvConfig{Pristine: Bool(true)},
			&EnvConfig{Pristine: Bool(true)},
		},
		{
			"whitelist_appends",
			&EnvConfig{Whitelist: []string{"whitelist"}},
			&EnvConfig{Whitelist: []string{"different"}},
			&EnvConfig{Whitelist: []string{"whitelist", "different"}},
		},
		{
			"whitelist_empty_one",
			&EnvConfig{Whitelist: []string{"whitelist"}},
			&EnvConfig{},
			&EnvConfig{Whitelist: []string{"whitelist"}},
		},
		{
			"whitelist_empty_two",
			&EnvConfig{},
			&EnvConfig{Whitelist: []string{"whitelist"}},
			&EnvConfig{Whitelist: []string{"whitelist"}},
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

func TestExecConfig_Env(t *testing.T) {
	cases := []struct {
		name string
		c    *EnvConfig
		r    []string
	}{
		{
			"no_args",
			&EnvConfig{},
			os.Environ(),
		},
		{
			"pristine",
			&EnvConfig{
				Pristine: Bool(true),
			},
			[]string{}, // IMPORTANT: should not be nil!
		},
		{
			"custom",
			&EnvConfig{
				Custom: []string{"a=b", "c=d"},
			},
			append(os.Environ(), "a=b", "c=d"),
		},
		{
			"whitelist",
			&EnvConfig{
				Whitelist: []string{"GOPATH"},
			},
			[]string{"GOPATH=" + os.Getenv("GOPATH")},
		},
		{
			"blacklist",
			&EnvConfig{
				Blacklist: []string{"*"},
			},
			[]string{},
		},
		{
			"pristine_custom",
			&EnvConfig{
				Pristine: Bool(true),
				Custom:   []string{"a=b", "c=d"},
			},
			[]string{"a=b", "c=d"},
		},
		{
			"whitelist_blacklist",
			&EnvConfig{
				Whitelist: []string{"GOPATH"},
				Blacklist: []string{"GO*"},
			},
			[]string{},
		},
		{
			"custom_whitelist_blacklist",
			&EnvConfig{
				Custom:    []string{"a=b", "c=d"},
				Whitelist: []string{"GOPATH"},
				Blacklist: []string{"GO*"},
			},
			[]string{"a=b", "c=d"},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			result := tc.c.Env()
			if !reflect.DeepEqual(tc.r, result) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, result)
			}
		})
	}
}

func TestEnvConfig_Finalize(t *testing.T) {
	cases := []struct {
		name string
		i    *EnvConfig
		r    *EnvConfig
	}{
		{
			"empty",
			&EnvConfig{},
			&EnvConfig{
				Blacklist: []string{},
				Custom:    []string{},
				Pristine:  Bool(false),
				Whitelist: []string{},
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
