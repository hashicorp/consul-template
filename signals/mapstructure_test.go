package signals

import (
	"os"
	"reflect"
	"syscall"
	"testing"

	"github.com/mitchellh/mapstructure"
)

func TestStringToSignalFunc(t *testing.T) {
	f := StringToSignalFunc()
	strType := reflect.TypeOf("")
	sigType := reflect.TypeOf((*os.Signal)(nil)).Elem()

	cases := []struct {
		f, t     reflect.Type
		data     interface{}
		expected interface{}
		err      bool
	}{
		{strType, sigType, "SIGTERM", syscall.SIGTERM, false},
		{strType, sigType, "SIGINT", syscall.SIGINT, false},

		// Invalid signal name
		{strType, sigType, "BACON", nil, true},
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
