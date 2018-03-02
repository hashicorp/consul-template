package config

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDefaultsConfig_Copy(t *testing.T) {
	cases := []struct {
		name string
		a    *DefaultsConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&DefaultsConfig{},
		},
		{
			"copy",
			&DefaultsConfig{
				LeftDelim:  String("<<"),
				RightDelim: String(">>"),
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

func TestDefaultsConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *DefaultsConfig
		b    *DefaultsConfig
		r    *DefaultsConfig
	}{
		{
			"nil_a",
			nil,
			&DefaultsConfig{},
			&DefaultsConfig{},
		},
		{
			"nil_b",
			&DefaultsConfig{},
			nil,
			&DefaultsConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&DefaultsConfig{},
			&DefaultsConfig{},
			&DefaultsConfig{},
		},
		{
			"left_delim_l",
			&DefaultsConfig{LeftDelim: String("<<")},
			&DefaultsConfig{},
			&DefaultsConfig{LeftDelim: String("<<")},
		},
		{
			"left_delim_r",
			&DefaultsConfig{},
			&DefaultsConfig{LeftDelim: String("<<")},
			&DefaultsConfig{LeftDelim: String("<<")},
		},
		{
			"left_delim_r2",
			&DefaultsConfig{LeftDelim: String(">>")},
			&DefaultsConfig{LeftDelim: String("<<")},
			&DefaultsConfig{LeftDelim: String("<<")},
		},
		{
			"right_delim_l",
			&DefaultsConfig{RightDelim: String(">>")},
			&DefaultsConfig{},
			&DefaultsConfig{RightDelim: String(">>")},
		},
		{
			"right_delim_r",
			&DefaultsConfig{},
			&DefaultsConfig{RightDelim: String(">>")},
			&DefaultsConfig{RightDelim: String(">>")},
		},
		{
			"right_delim_r2",
			&DefaultsConfig{RightDelim: String("<<")},
			&DefaultsConfig{RightDelim: String(">>")},
			&DefaultsConfig{RightDelim: String(">>")},
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
