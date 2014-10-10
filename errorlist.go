package main

import (
	"fmt"
	"strings"
)

// ErrorList represents the combined error returned from an ErrorList
type ErrorList struct {
	title  string
	errors []error
}

// GoString returns the detailed format of this object
func (e *ErrorList) GoString() string {
	return fmt.Sprintf("*%#v", *e)
}

// Append adds a new error or errors onto the ErrorList
func (e *ErrorList) Append(errs ...error) {
	e.errors = append(e.errors, errs...)
}

// Appendf adds a new error to the list, converting the given string to a proper
// error object.
func (e *ErrorList) Appendf(text string, args ...interface{}) {
	e.errors = append(e.errors, fmt.Errorf(text, args...))
}

// ErrorList returns a formatted error with the title and each error as a bullet. If
// there are no errors in the list, ErrorList will return nil.
func (e *ErrorList) GetError() error {
	if len(e.errors) == 0 {
		return nil
	}
	return e
}

// Error implements the Error interface
func (e *ErrorList) Error() string {
	buff := make([]string, 0)
	for _, err := range e.errors {
		switch err.(type) {
		case *ErrorList:
			typed, ok := err.(*ErrorList)
			if !ok {
				panic("could not convert error to ErrorList")
			}
			buff = e.recursiveError(buff, typed.errors, typed.title)
		default:
			buff = append(buff, fmt.Sprintf("* %s", err))
		}
	}

	return fmt.Sprintf("%d error(s) %s:\n%s", len(buff), e.title, strings.Join(buff, "\n"))
}

func (e *ErrorList) recursiveError(buff []string, errs []error, title string) []string {
	for _, err := range errs {
		switch err.(type) {
		case *ErrorList:
			typed, ok := err.(*ErrorList)
			if !ok {
				panic("could not convert error to ErrorList")
			}
			buff = e.recursiveError(buff, typed.errors, fmt.Sprintf("%s: %s", title, typed.title))
		default:
			buff = append(buff, fmt.Sprintf("* %s: %s", title, err))
		}
	}

	return buff
}

// NewErrorList creates a new ErrorList
func NewErrorList(title string) *ErrorList {
	return &ErrorList{
		title:  title,
		errors: []error{},
	}
}
