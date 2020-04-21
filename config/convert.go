package config

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/consul-template/signals"
)

// Bool returns a pointer to the given bool.
func Bool(b bool) *bool {
	return &b
}

// BoolVal returns the value of the boolean at the pointer, or false if the
// pointer is nil.
func BoolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// BoolCopy returns a copy of the boolean pointer
func BoolCopy(b *bool) *bool {
	if b == nil {
		return nil
	}

	return Bool(*b)
}

// BoolGoString returns the value of the boolean for printing in a string.
func BoolGoString(b *bool) string {
	if b == nil {
		return "(*bool)(nil)"
	}
	return fmt.Sprintf("%t", *b)
}

// BoolPresent returns a boolean indicating if the pointer is nil, or if the
// pointer is pointing to the zero value..
func BoolPresent(b *bool) bool {
	if b == nil {
		return false
	}
	return true
}

// FileMode returns a pointer to the given os.FileMode.
func FileMode(o os.FileMode) *os.FileMode {
	return &o
}

// FileModeVal returns the value of the os.FileMode at the pointer, or 0 if the
// pointer is nil.
func FileModeVal(o *os.FileMode) os.FileMode {
	if o == nil {
		return 0
	}
	return *o
}

// FileModeCopy returns a copy of the os.FireMode
func FileModeCopy(o *os.FileMode) *os.FileMode {
	if o == nil {
		return nil
	}

	return FileMode(*o)
}

// FileModeGoString returns the value of the os.FileMode for printing in a
// string.
func FileModeGoString(o *os.FileMode) string {
	if o == nil {
		return "(*os.FileMode)(nil)"
	}
	return fmt.Sprintf("%q", *o)
}

// FileModePresent returns a boolean indicating if the pointer is nil, or if
// the pointer is pointing to the zero value.
func FileModePresent(o *os.FileMode) bool {
	if o == nil {
		return false
	}
	return *o != 0
}

// Int returns a pointer to the given int.
func Int(i int) *int {
	return &i
}

// IntVal returns the value of the int at the pointer, or 0 if the pointer is
// nil.
func IntVal(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// IntCopy returns a copy of the int pointer
func IntCopy(i *int) *int {
	if i == nil {
		return nil
	}

	return Int(*i)
}

// IntGoString returns the value of the int for printing in a string.
func IntGoString(i *int) string {
	if i == nil {
		return "(*int)(nil)"
	}
	return fmt.Sprintf("%d", *i)
}

// IntPresent returns a boolean indicating if the pointer is nil, or if the
// pointer is pointing to the zero value.
func IntPresent(i *int) bool {
	if i == nil {
		return false
	}
	return *i != 0
}

// Uint returns a pointer to the given uint.
func Uint(i uint) *uint {
	return &i
}

// UintVal returns the value of the uint at the pointer, or 0 if the pointer is
// nil.
func UintVal(i *uint) uint {
	if i == nil {
		return 0
	}
	return *i
}

// UintCopy returns a copy of the uint pointer
func UintCopy(i *uint) *uint {
	if i == nil {
		return nil
	}

	return Uint(*i)
}

// UintGoString returns the value of the uint for printing in a string.
func UintGoString(i *uint) string {
	if i == nil {
		return "(*uint)(nil)"
	}
	return fmt.Sprintf("%d", *i)
}

// UintPresent returns a boolean indicating if the pointer is nil, or if the
// pointer is pointing to the zero value.
func UintPresent(i *uint) bool {
	if i == nil {
		return false
	}
	return *i != 0
}

// Signal returns a pointer to the given os.Signal.
func Signal(s os.Signal) *os.Signal {
	return &s
}

// SignalVal returns the value of the os.Signal at the pointer, or 0 if the
// pointer is nil.
func SignalVal(s *os.Signal) os.Signal {
	if s == nil {
		return (os.Signal)(nil)
	}
	return *s
}

// SignalCopy returns a copy of the os.Signal
func SignalCopy(s *os.Signal) *os.Signal {
	if s == nil {
		return nil
	}

	return Signal(*s)
}

// SignalGoString returns the value of the os.Signal for printing in a string.
func SignalGoString(s *os.Signal) string {
	if s == nil {
		return "(*os.Signal)(nil)"
	}
	if *s == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q", *s)
}

// SignalPresent returns a boolean indicating if the pointer is nil, or if the pointer is pointing to the zero value..
func SignalPresent(s *os.Signal) bool {
	if s == nil {
		return false
	}
	return *s != signals.SIGNIL
}

// String returns a pointer to the given string.
func String(s string) *string {
	return &s
}

// StringVal returns the value of the string at the pointer, or "" if the
// pointer is nil.
func StringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// StringCopy returns a copy of the string pointer
func StringCopy(s *string) *string {
	if s == nil {
		return nil
	}

	return String(*s)
}

// StringGoString returns the value of the string for printing in a string.
func StringGoString(s *string) string {
	if s == nil {
		return "(*string)(nil)"
	}
	return fmt.Sprintf("%q", *s)
}

// StringPresent returns a boolean indicating if the pointer is nil, or if the pointer is pointing to the zero value..
func StringPresent(s *string) bool {
	if s == nil {
		return false
	}
	return *s != ""
}

// TimeDuration returns a pointer to the given time.Duration.
func TimeDuration(t time.Duration) *time.Duration {
	return &t
}

// TimeDurationVal returns the value of the string at the pointer, or 0 if the
// pointer is nil.
func TimeDurationVal(t *time.Duration) time.Duration {
	if t == nil {
		return time.Duration(0)
	}
	return *t
}

// TimeDurationCopy returns a copy of the time.Duration pointer
func TimeDurationCopy(t *time.Duration) *time.Duration {
	if t == nil {
		return nil
	}

	return TimeDuration(*t)
}

// TimeDurationGoString returns the value of the time.Duration for printing in a
// string.
func TimeDurationGoString(t *time.Duration) string {
	if t == nil {
		return "(*time.Duration)(nil)"
	}
	return fmt.Sprintf("%s", t)
}

// TimeDurationPresent returns a boolean indicating if the pointer is nil, or if the pointer is pointing to the zero value..
func TimeDurationPresent(t *time.Duration) bool {
	if t == nil {
		return false
	}
	return *t != 0
}
