package config

import (
	"fmt"
	"testing"
	"time"
)

func TestBool(t *testing.T) {
	cases := []struct {
		name string
		b    bool
	}{
		{
			"true",
			true,
		},
		{
			"false",
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			b := Bool(tc.b)
			if *b != tc.b {
				t.Errorf("\nexp: %t\nact: %t", tc.b, *b)
			}
		})
	}
}

func TestBoolVal(t *testing.T) {
	cases := []struct {
		name string
		b    *bool
	}{
		{
			"true",
			Bool(true),
		},
		{
			"false",
			Bool(false),
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			b := BoolVal(tc.b)
			if b != *tc.b {
				t.Errorf("\nexp: %t\nact: %t", *tc.b, b)
			}
		})
	}
}

func TestString(t *testing.T) {
	cases := []struct {
		name string
		s    string
	}{
		{
			"input",
			"hello world",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			s := String(tc.s)
			if *s != tc.s {
				t.Errorf("\nexp: %s\nact: %s", tc.s, *s)
			}
		})
	}
}

func TestStringVal(t *testing.T) {
	cases := []struct {
		name string
		s    *string
	}{
		{
			"input",
			String("hello world"),
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			s := StringVal(tc.s)
			if s != *tc.s {
				t.Errorf("\nexp: %s\nact: %s", *tc.s, s)
			}
		})
	}
}

func TestTimeDuration(t *testing.T) {
	cases := []struct {
		name string
		d    time.Duration
	}{
		{
			"input",
			10 * time.Second,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d := TimeDuration(tc.d)
			if *d != tc.d {
				t.Errorf("\nexp: %s\nact: %s", tc.d, *d)
			}
		})
	}
}

func TestTimeDurationVal(t *testing.T) {
	cases := []struct {
		name string
		d    *time.Duration
	}{
		{
			"input",
			TimeDuration(10 * time.Second),
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d := TimeDurationVal(tc.d)
			if d != *tc.d {
				t.Errorf("\nexp: %s\nact: %s", *tc.d, d)
			}
		})
	}
}
