package main

import "fmt"

type ErrExitable interface {
	ExitStatus() int
}

var _ error = new(ErrChildDied)
var _ ErrExitable = new(ErrChildDied)

type ErrChildDied struct {
	code int
}

func NewErrChildDied(c int) *ErrChildDied {
	return &ErrChildDied{code: c}
}

func (e *ErrChildDied) Error() string {
	return fmt.Sprintf("child process died with exit code %d", e.code)
}

func (e *ErrChildDied) ExitStatus() int {
	return e.code
}
