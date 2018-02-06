package config

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestRateLimitFunc(t *testing.T) {
	cases := []struct {
		name string
		c    *RateLimitConfig
		t    *time.Duration
		rc   *bool
		rs   *time.Duration
	}{
		{
			"default, 0 duration",
			&RateLimitConfig{
				RandomBackoff: TimeDuration(0 * time.Millisecond),
			},
			TimeDuration(0 * time.Millisecond),
			Bool(true),
			TimeDuration(DefaultMinDelayBetweenUpdates),
		},
		{
			"default, 5 duration",
			&RateLimitConfig{
				RandomBackoff: TimeDuration(0 * time.Millisecond),
			},
			TimeDuration(5 * time.Millisecond),
			Bool(true),
			TimeDuration(DefaultMinDelayBetweenUpdates - 5*time.Millisecond),
		},
		{
			"default, Min duration between calls",
			&RateLimitConfig{
				RandomBackoff: TimeDuration(0 * time.Millisecond),
			},
			TimeDuration(DefaultMinDelayBetweenUpdates - 1),
			Bool(true),
			TimeDuration(1),
		},
		{
			"default, Min duration between calls",
			&RateLimitConfig{
				RandomBackoff: TimeDuration(0 * time.Millisecond),
			},
			TimeDuration(DefaultMinDelayBetweenUpdates + 10),
			Bool(false),
			TimeDuration(0),
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.c.Finalize()
			wait, s := tc.c.RateLimitFunc()(*tc.t)
			if wait != *tc.rc {
				t.Errorf("\nexp sleep wait: %#v\nact: %#v", *tc.rc, wait)
			}
			if (*tc.rs) != s {
				t.Errorf("\nexp sleep time: %#v\nact: %#v", *tc.rs, s)
			}
		})
	}

}

func TestRateLimitConfig_Copy(t *testing.T) {
	cases := []struct {
		name string
		a    *RateLimitConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&RateLimitConfig{},
		},
		{
			"same_enabled",
			&RateLimitConfig{
				RandomBackoff:          TimeDuration(DefaultRandomBackoff),
				MinDelayBetweenUpdates: TimeDuration(DefaultMinDelayBetweenUpdates),
				Enabled:                Bool(true),
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

func TestRateLimitConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *RateLimitConfig
		b    *RateLimitConfig
		r    *RateLimitConfig
	}{
		{
			"nil_a",
			nil,
			&RateLimitConfig{},
			&RateLimitConfig{},
		},
		{
			"nil_b",
			&RateLimitConfig{},
			nil,
			&RateLimitConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&RateLimitConfig{},
			&RateLimitConfig{},
			&RateLimitConfig{},
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

func TestRateLimitConfig_Finalize(t *testing.T) {
	cases := []struct {
		name string
		i    *RateLimitConfig
		r    *RateLimitConfig
	}{
		{
			"empty",
			&RateLimitConfig{},
			&RateLimitConfig{
				RandomBackoff:          TimeDuration(DefaultRandomBackoff),
				MinDelayBetweenUpdates: TimeDuration(DefaultMinDelayBetweenUpdates),
				Enabled:                Bool(true),
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
