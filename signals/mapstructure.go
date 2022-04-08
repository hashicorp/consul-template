package signals

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// StringToSignalFunc parses a string as a signal based on the signal lookup
// table. If the user supplied an empty string or nil, a special "nil signal"
// is returned. Clients should check for this value and set the response back
// nil after mapstructure finishes parsing.
func StringToSignalFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		//TODO(schmichael): I added the syscall.Signal check because I
		//can't get `t` to have type os.Signal here!
		//
		// How do you reflect to a type of os.Signal?
		// If I pass a ValueOf(os.Signal)(nil) to DecodeHookExec, it
		// panics because you can't get a concrete type out of that.
		//
		// If I pass a ValueOf(os.Interrupt), then DecodeHookExec gets
		// the *concrete* type (syscall.Signal).
		//
		// Not sure what else to try
		if t.String() != "syscall.Signal" && t.String() != "os.Signal" {
			return data, nil
		}

		if data == nil || data.(string) == "" {
			return SIGNIL, nil
		}

		return Parse(data.(string))
	}
}
