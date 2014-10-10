package main

import (
	"fmt"
	"strings"
)

// Error represents the combined error returned from an ErrorList
type Error struct {
	title  string
	errors []error
}

// Error implements the Error interface
func (e *Error) Error() string {
	buff := make([]string, 0)
	for _, err := range e.errors {
		switch err.(type) {
		case *Error:
			typed, ok := err.(*Error)
			if !ok {
				panic("could not convert error to Error")
			}
			buff = e.recursiveError(buff, typed.errors, typed.title)
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

func (e *Error) recursiveError(buff []string, errs []error, title string) []string {
	for _, err := range errs {
		switch err.(type) {
		case *Error:
			typed, ok := err.(*Error)
			if !ok {
				panic("could not convert error to Error")
			}
			buff = e.recursiveError(buff, typed.errors, fmt.Sprintf("%s: %s", title, typed.title))
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

// ErrorList is an accumulator of error objects with some handy helpers
type ErrorList Error

// NewErrorList creates a new ErrorList
func NewErrorList(title string) *ErrorList {
	return &ErrorList{
		title:  title,
		errors: []error{},
	}
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

// ErrorList implements the Error interface
func (e *ErrorList) Error() string {
	return e.GetError().Error()
}

// Error returns a formatted error with the title and each error as a bullet. If
// there are no errors in the list, Error will return nil.
func (e *ErrorList) GetError() error {
	if len(e.errors) == 0 {
		return nil
	}

	return &Error{
		title:  e.title,
		errors: e.errors,
	}
}
