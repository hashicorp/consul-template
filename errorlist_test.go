package main

import (
	"errors"
	// "reflect"
	"strings"
	"testing"
)

// Test that the title is set
func TestNewErrorList_setsTitle(t *testing.T) {
	title := "the title"
	list := NewErrorList(title)

	if list.title != title {
		t.Fatalf("expected title to be %s, was %s", title, list.title)
	}
}

// Test that the errors is an empty array
func TestNewErrorList_setsErrors(t *testing.T) {
	list := NewErrorList("title")

	if len(list.errors) > 0 {
		t.Fatalf("expected errors to be empty")
	}
}

// Test that a single error is appended
func TestAppend_singleError(t *testing.T) {
	list := NewErrorList("title")
	err := errors.New("error")
	list.Append(err)

	if len(list.errors) == 0 {
		t.Fatal("expected errors, but slice was empty")
	}

	if list.errors[0] != err {
		t.Fatalf("expected error to be %q, but was %q", err, list.errors[0])
	}
}

// Test that a multiple errors are appended
func TestAppend_multipleErrors(t *testing.T) {
	list := NewErrorList("title")
	err_1 := errors.New("error 1")
	err_2 := errors.New("error 2")

	list.Append(err_1, err_2)

	if len(list.errors) != 2 {
		t.Fatalf("expected 2 errors, but slice had %d", len(list.errors))
	}

	if list.errors[0] != err_1 {
		t.Fatalf("expected error to be %q, but was %q", err_1, list.errors[0])
	}

	if list.errors[1] != err_2 {
		t.Fatalf("expected error to be %q, but was %q", err_2, list.errors[0])
	}
}

// Test that appending an ErrorList prefixes values
func TestAppend_errorList(t *testing.T) {
	parent := NewErrorList("parsing config")
	parent.Append(errors.New("missing key 'magic'"))
	parent.Append(errors.New("missing key 'ponies'"))

	child := NewErrorList("parsing wait time")
	child.Append(errors.New("must be a duration"))
	child.Append(errors.New("must be less than 5s"))
	parent.Append(child)

	// Intentionally add the infant after the child has been added to the parent
	// to ensure the pointers are still followed when printing
	infant := NewErrorList("converting to seconds")
	infant.Append(errors.New("failed to convert because of the lunar eclipse"))
	child.Append(infant)

	if len(parent.errors) != 3 {
		t.Fatalf("expected 3 errors, but slice had %d", len(parent.errors))
	}

	expected := strings.TrimSpace(`
5 error(s) parsing config:
* missing key 'magic'
* missing key 'ponies'
* parsing wait time: must be a duration
* parsing wait time: must be less than 5s
* parsing wait time: converting to seconds: failed to convert because of the lunar eclipse
`)

	trimmedError := strings.TrimSpace(parent.Error())
	if trimmedError != expected {
		t.Errorf("expected %q to equal %q", trimmedError, expected)
	}
}

// Test that Appendf converts strings to errors
func TestAppendf_stringErrors(t *testing.T) {
	list := NewErrorList("title")
	list.Appendf("error")

	if len(list.errors) == 0 {
		t.Fatalf("expected 1 errors, but slice had %d", len(list.errors))
	}
}

// Test that the Error method returns a string
func TestError_returnsString(t *testing.T) {
	list := NewErrorList("parsing the config")
	list.Append(errors.New("something bad"))

	expected := "1 error(s) parsing the config:\n* something bad"
	if !strings.Contains(list.Error(), expected) {
		t.Errorf("expected %q to contain %q", list.Error(), expected)
	}
}

// Test that GetError returns nil when there are no errors
func TestGetError_returnsNil(t *testing.T) {
	list := NewErrorList("title")
	if list.GetError() != nil {
		t.Errorf("expected error to be nil, but was %q", list.GetError())
	}
}

// Test that GetError returns an Error object
func TestGetError_returnsError(t *testing.T) {
	list := NewErrorList("title")
	list.Append(errors.New("something bad"))
	list.Append(errors.New("something worse"))

	err, ok := list.GetError().(*Error)
	if !ok {
		t.Fatal("could not convert to Error")
	}

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	if err.title != "title" {
		t.Errorf("expected title to be %q, was %q", "title", err.title)
	}

	if len(err.errors) != 2 {
		t.Fatalf("expected 2 errors, but slice had %d", len(err.errors))
	}
}

// Test that the Error returns the properly formatted string
func TestErrorError_string(t *testing.T) {
	list := NewErrorList("parsing config")
	list.Append(errors.New("something bad"))
	list.Append(errors.New("something worse"))

	err := list.GetError()

	expected := "2 error(s) parsing config:\n* something bad\n* something worse"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}
