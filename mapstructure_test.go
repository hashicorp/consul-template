package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/mitchellh/mapstructure"
)

func TestStringToFileMode(t *testing.T) {
	f := StringToFileMode()
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
		actual, err := mapstructure.DecodeHookExec(f, tc.f, tc.t, tc.data)
		if (err != nil) != tc.err {
			t.Errorf("case %d: %s", i, err)
		}
		if !reflect.DeepEqual(actual, tc.expected) {
			t.Errorf("case %d: expected %#v to be %#v", i, actual, tc.expected)
		}
	}
}
