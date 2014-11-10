package util

import (
	"strings"
	"testing"
	"time"
)

// Test that an error is returned when the empty string is given
func TestWaitParse_emptyStringArgs(t *testing.T) {
	_, err := ParseWait("")

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify empty wait interval"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that an error is returned when a string with spaces is given
func TestWaitParse_stringWithSpacesArgs(t *testing.T) {
	_, err := ParseWait("  ")

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify empty wait interval"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that an error is returned when there are too many arguments
func TestWaitParse_tooManyArgs(t *testing.T) {
	_, err := ParseWait("5s:10s:15s")

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "invalid wait interval format"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that the error returned from parsing is propagated
func TestWaitParse_noUnits(t *testing.T) {
	_, err := ParseWait("5:10")

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "missing unit in duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that a single wait value is correctly used
func TestWaitParse_singleWait(t *testing.T) {
	wait, err := ParseWait("5s")

	if err != nil {
		t.Fatal(err)
	}

	expectedMin := time.Duration(5) * time.Second
	if wait.Min != expectedMin {
		t.Errorf("expected %q to equal %q", wait.Min, expectedMin)
	}

	expectedMax := time.Duration(20) * time.Second
	if wait.Max != expectedMax {
		t.Errorf("expected %q to equal %q", wait.Max, expectedMax)
	}
}

// Test that a multiple wait value is correctly used
func TestWaitParse_multipleWait(t *testing.T) {
	wait, err := ParseWait("10s:20s")
	if err != nil {
		t.Fatal(err)
	}

	expectedMin := time.Duration(10) * time.Second
	if wait.Min != expectedMin {
		t.Errorf("expected %q to equal %q", wait.Min, expectedMin)
	}

	expectedMax := time.Duration(20) * time.Second
	if wait.Max != expectedMax {
		t.Errorf("expected %q to equal %q", wait.Max, expectedMax)
	}
}

// Test that an error is returned the min is negative
func TestWaitParse_minNegative(t *testing.T) {
	_, err := ParseWait("-5s")

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify a negative wait interval"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that an error is returned the min is negative
func TestWaitParse_maxNegative(t *testing.T) {
	_, err := ParseWait("-5s:-10s")

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify a negative wait interval"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that an error is returned if the maximum larger than minimum
func TestWaitParse_maxLargerThanMin(t *testing.T) {
	_, err := ParseWait("15s:5s")

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "max must be larger than min"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}
