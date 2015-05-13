package main

import (
	"strconv"
	"time"
)

// funcVar is a type of flag that accepts a function that is the string given
// by the user.
type funcVar func(s string) error

func (f funcVar) Set(s string) error { return f(s) }
func (f funcVar) String() string     { return "" }
func (f funcVar) IsBoolFlag() bool   { return false }

// funcBoolVar is a type of flag that accepts a function, converts the user's
// value to a bool, and then calls the given function.
type funcBoolVar func(b bool) error

func (f funcBoolVar) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	return f(v)
}
func (f funcBoolVar) String() string   { return "" }
func (f funcBoolVar) IsBoolFlag() bool { return true }

// funcDurationVar is a type of flag that accepts a function, converts the
// user's value to a duration, and then calls the given function.
type funcDurationVar func(d time.Duration) error

func (f funcDurationVar) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	return f(v)
}
func (f funcDurationVar) String() string   { return "" }
func (f funcDurationVar) IsBoolFlag() bool { return false }
