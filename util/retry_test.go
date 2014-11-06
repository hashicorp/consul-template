package util

import (
  "strings"
  "testing"
  "time"
)

// Test that an error is returned when the empty string is given
func TestParse_emptyStringArgs(t *testing.T) {
  _, err := ParseRetry("")

  if err == nil {
    t.Fatal("expected error, but nothing was returned")
  }

  expectedErr := "cannot specify empty retry period"
  if !strings.Contains(err.Error(), expectedErr) {
    t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
  }
}

// Test that an error is returned when a string with spaces is given
func TestParse_stringWithSpacesArgs(t *testing.T) {
  _, err := ParseRetry("  ")

  if err == nil {
    t.Fatal("expected error, but nothing was returned")
  }

  expectedErr := "cannot specify empty retry period"
  if !strings.Contains(err.Error(), expectedErr) {
    t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
  }
}

// Test that an error is returned when there are too many arguments
func TestParse_tooManyArgs(t *testing.T) {
  _, err := ParseRetry("15s:10:15s")

  if err == nil {
    t.Fatal("expected error, but nothing was returned")
  }

  expectedErr := "invalid retry period format"
  if !strings.Contains(err.Error(), expectedErr) {
    t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
  }
}

// Test that the error returned from parsing is propagated
func TestParse_noUnits(t *testing.T) {
  _, err := ParseRetry("25:10")

  if err == nil {
    t.Fatal("expected error, but nothing was returned")
  }

  expectedErr := "missing unit in duration"
  if !strings.Contains(err.Error(), expectedErr) {
    t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
  }
}

// Test that a single wait value is correctly used
func TestParse_justInitial(t *testing.T) {
  retry, err := ParseRetry("30s")

  if err != nil {
    t.Fatal(err)
  }

  expectedInitial := time.Duration(30) * time.Second
  if retry.Initial != expectedInitial {
    t.Errorf("expected %q to equal %q", retry.Initial, expectedInitial)
  }

  expectedGrowth := 1
  if retry.Growth != expectedGrowth {
    t.Errorf("expected %q to equal %q", retry.Growth, expectedGrowth)
  }
}

// Test that both arguments are parsed properly
func TestParse_bothArgs(t *testing.T) {
  retry, err := ParseRetry("10s:1.5")
  if err != nil {
    t.Fatal(err)
  }

  expectedInitial := time.Duration(10) * time.Second
  if retry.Initial != expectedInitial {
    t.Errorf("expected %q to equal %q", retry.Initial, expectedInitial)
  }

  expectedGrowth := 1.5
  if retry.Growth != expectedGrowth {
    t.Errorf("expected %q to equal %q", retry.Growth, expectedGrowth)
  }
}

// Test that an error is returned when initial retry is negative
func TestParse_negativeRetry(t *testing.T) {
  _, err := ParseRetry("-5s")

  if err == nil {
    t.Fatal("expected error, but nothing was returned")
  }

  expectedErr := "cannot specify a negative initial retry period"
  if !strings.Contains(err.Error(), expectedErr) {
    t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
  }
}

// Test that an error is returned the growth is less than 1
func TestParse_suboneGrowth(t *testing.T) {
  _, err := ParseRetry("15s:-10")

  if err == nil {
    t.Fatal("expected error, but nothing was returned")
  }

  expectedErr := "cannot specify a growth factor of less than 1"
  if !strings.Contains(err.Error(), expectedErr) {
    t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
  }
}
