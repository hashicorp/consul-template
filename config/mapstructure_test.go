package config

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
)

func TestStringToFileModeFunc(t *testing.T) {
	f := StringToFileModeFunc()
	strType := reflect.TypeOf("")
	fmType := reflect.TypeOf(os.FileMode(0))
	u32Type := reflect.TypeOf(uint32(0))

	cases := []struct {
		f, t     reflect.Type
		data     interface{}
		expected interface{}
		err      bool
	}{
		{strType, fmType, "0600", os.FileMode(0600), false},
		{strType, fmType, "4600", os.FileMode(04600), false},

		// Prepends 0 automatically
		{strType, fmType, "600", os.FileMode(0600), false},

		// Invalid file mode
		{strType, fmType, "12345", "12345", true},

		// Invalid syntax
		{strType, fmType, "abcd", "abcd", true},

		// Different type
		{strType, strType, "0600", "0600", false},
		{strType, u32Type, "0600", "0600", false},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual, err := mapstructure.DecodeHookExec(f, tc.f, tc.t, tc.data)
			if (err != nil) != tc.err {
				t.Fatalf("%s", err)
			}
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.expected, actual)
			}
		})
	}
}

func TestStringToWaitDurationHookFunc(t *testing.T) {
	f := StringToWaitDurationHookFunc()
	strType := reflect.TypeOf("")
	waitType := reflect.TypeOf(WaitConfig{})

	cases := []struct {
		name     string
		f, t     reflect.Type
		data     interface{}
		expected interface{}
		err      bool
	}{
		{
			"min",
			strType, waitType,
			"5s",
			&WaitConfig{
				Min: TimeDuration(5 * time.Second),
				Max: TimeDuration(20 * time.Second),
			},
			false,
		},
		{
			"min_max",
			strType, waitType,
			"5s:10s",
			&WaitConfig{
				Min: TimeDuration(5 * time.Second),
				Max: TimeDuration(10 * time.Second),
			},
			false,
		},
		{
			"not_string",
			waitType, waitType,
			&WaitConfig{},
			&WaitConfig{},
			false,
		},
		{
			"not_wait",
			strType, strType,
			"test",
			"test",
			false,
		},
		{
			"bad_wait",
			strType, waitType,
			"nope",
			(*WaitConfig)(nil),
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			actual, err := mapstructure.DecodeHookExec(f, tc.f, tc.t, tc.data)
			if (err != nil) != tc.err {
				t.Fatalf("%s", err)
			}
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.expected, actual)
			}
		})
	}
}

func TestConsulStringToStructFunc(t *testing.T) {
	f := ConsulStringToStructFunc()
	strType := reflect.TypeOf("")
	consulType := reflect.TypeOf(ConsulConfig{})

	cases := []struct {
		name     string
		f, t     reflect.Type
		data     interface{}
		expected interface{}
		err      bool
	}{
		{
			"address",
			strType, consulType,
			"1.2.3.4",
			&ConsulConfig{
				Address: String("1.2.3.4"),
			},
			false,
		},
		{
			"not_string",
			consulType, consulType,
			&ConsulConfig{},
			&ConsulConfig{},
			false,
		},
		{
			"not_consul",
			strType, strType,
			"test",
			"test",
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			actual, err := mapstructure.DecodeHookExec(f, tc.f, tc.t, tc.data)
			if (err != nil) != tc.err {
				t.Fatalf("%s", err)
			}
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.expected, actual)
			}
		})
	}
}
