package main

import (
	"strings"
	"testing"
	"time"
)

// Test that an error is returned when the empty string is given
func TestParse_emptyStringArgs(t *testing.T) {
	_, err := ParseWait("")
	assertError(err, "cannot specify empty wait interval", t)
}

// Test that an error is returned when a string with spaces is given
func TestParse_stringWithSpacesArgs(t *testing.T) {
	_, err := ParseWait("  ")
	assertError(err, "cannot specify empty wait interval", t)
}

// Test that an error is returned when there are too many arguments
func TestParse_tooManyArgs(t *testing.T) {
	_, err := ParseWait("5s:10s:15s")
	assertError(err, "invalid wait interval format", t)
}

// Test that the error returned from parsing is propagated
func TestParse_noUnits(t *testing.T) {
	_, err := ParseWait("5:10")
	assertError(err, "missing unit in duration", t)
}

// Test that a single wait value is correctly used
func TestParse_singleWait(t *testing.T) {
	wait, err := ParseWait("5s")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(wait.Min, time.Duration(5)*time.Second, t)
	assertEqual(wait.Max, time.Duration(20)*time.Second, t)
}

// Test that a multiple wait value is correctly used
func TestParse_multipleWait(t *testing.T) {
	wait, err := ParseWait("10s:20s")
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(wait.Min, time.Duration(10)*time.Second, t)
	assertEqual(wait.Max, time.Duration(20)*time.Second, t)
}

// Test that an error is returned the min is negative
func TestParse_minNegative(t *testing.T) {
	_, err := ParseWait("-5s")
	assertError(err, "cannot specify a negative wait interval", t)
}

// Test that an error is returned the min is negative
func TestParse_maxNegative(t *testing.T) {
	_, err := ParseWait("-5s:-10s")
	assertError(err, "cannot specify a negative wait interval", t)
}

// Test that an error is returned if the maximum larger than minimum
func TestParse_maxLargerThanMin(t *testing.T) {
	_, err := ParseWait("15s:5s")
	assertError(err, "max must be larger than min", t)
}

/*
 * Helpers
 */
func assertEqual(actual interface{}, expected interface{}, t *testing.T) {
	if actual != expected {
		t.Errorf("expected %q to equal %q", actual, expected)
	}
}

func assertError(err error, s string, t *testing.T) {
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	if !strings.Contains(err.Error(), s) {
		t.Fatalf("expected error %q to contain %q", err.Error(), s)
	}
}
