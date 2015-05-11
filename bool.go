package main

// Bool is a special bool type that provides the ability to be "unset".
type Bool int16

const (
	// BoolTrue, BoolUnset, and BoolFalse are true, "nil", and false respectively.
	BoolTrue  Bool = 1
	BoolUnset Bool = 0
	BoolFalse Bool = -1
)
