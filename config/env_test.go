package config

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestEnvConfig_Copy(t *testing.T) {
	t.Parallel()

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
				Denylist:            []string{"denylist"},
				DenylistDeprecated:  []string{},
				Custom:              []string{"custom"},
				Pristine:            Bool(true),
				Allowlist:           []string{"allowlist"},
				AllowlistDeprecated: []string{},
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
	t.Parallel()

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
			"denylist_appends",
			&EnvConfig{Denylist: []string{"denylist"}},
			&EnvConfig{Denylist: []string{"different"}},
			&EnvConfig{Denylist: []string{"denylist", "different"}},
		},
		{
			"denylist_empty_one",
			&EnvConfig{Denylist: []string{"denylist"}},
			&EnvConfig{},
			&EnvConfig{Denylist: []string{"denylist"}},
		},
		{
			"denylist_empty_two",
			&EnvConfig{},
			&EnvConfig{Denylist: []string{"denylist"}},
			&EnvConfig{Denylist: []string{"denylist"}},
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
			"allowlist_appends",
			&EnvConfig{Allowlist: []string{"allowlist"}},
			&EnvConfig{Allowlist: []string{"different"}},
			&EnvConfig{Allowlist: []string{"allowlist", "different"}},
		},
		{
			"allowlist_empty_one",
			&EnvConfig{Allowlist: []string{"allowlist"}},
			&EnvConfig{},
			&EnvConfig{Allowlist: []string{"allowlist"}},
		},
		{
			"allowlist_empty_two",
			&EnvConfig{},
			&EnvConfig{Allowlist: []string{"allowlist"}},
			&EnvConfig{Allowlist: []string{"allowlist"}},
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
	t.Parallel()

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
			"allowlist",
			&EnvConfig{
				Allowlist: []string{"GOPATH"},
			},
			[]string{"GOPATH=" + os.Getenv("GOPATH")},
		},
		{
			"allowlist_deprecated",
			&EnvConfig{
				AllowlistDeprecated: []string{"GOPATH"},
			},
			[]string{"GOPATH=" + os.Getenv("GOPATH")},
		},
		{
			"denylist",
			&EnvConfig{
				Denylist: []string{"*"},
			},
			[]string{},
		},
		{
			"denylist_deprecated",
			&EnvConfig{
				DenylistDeprecated: []string{"*"},
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
			"allowlist_denylist",
			&EnvConfig{
				Allowlist: []string{"GOPATH"},
				Denylist:  []string{"GO*"},
			},
			[]string{},
		},
		{
			"custom_allowlist_denylist",
			&EnvConfig{
				Custom:    []string{"a=b", "c=d"},
				Allowlist: []string{"GOPATH"},
				Denylist:  []string{"GO*"},
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
	t.Parallel()

	cases := []struct {
		name string
		i    *EnvConfig
		r    *EnvConfig
	}{
		{
			"empty",
			&EnvConfig{},
			&EnvConfig{
				Denylist:            []string{},
				DenylistDeprecated:  []string{},
				Custom:              []string{},
				Pristine:            Bool(false),
				Allowlist:           []string{},
				AllowlistDeprecated: []string{},
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
