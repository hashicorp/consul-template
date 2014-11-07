package util

import (
  "time"
  "errors"
  "strings"
  "strconv"
)

const (
  maxRetry = 1 * time.Hour
)

// Retry is the Min/Max duration used by the Watcher
type Retry struct {
  // Initial is the initial time to wait after an erro
  Initial time.Duration

  // Growth is the factor of which to increase the initial retry period for
  // additional failures
  Growth float64

  Next time.Duration
}

func ParseRetry(s string) (*Retry, error) {
  if len(strings.TrimSpace(s)) < 1 {
    return nil, errors.New("cannot specify empty retry period")
  }

  parts := strings.Split(s, ":")

  var initial time.Duration
  var growth float64
  var err error

  if len(parts) == 1 {
    initial, err = time.ParseDuration(strings.TrimSpace(parts[0]))
    if err != nil {
      return nil, err
    }

    // Default growth
    growth = 1.0
  } else if len(parts) == 2 {
    initial, err = time.ParseDuration(strings.TrimSpace(parts[0]))
    if err != nil {
      return nil, err
    }

    growth, err = strconv.ParseFloat(parts[1], 64)
    if err != nil {
      return nil, err
    }
  } else {
    return nil, errors.New("invalid retry period format")
  }

  if initial <= 0 {
    return nil, errors.New("cannot specify a negative initial retry period")
  }

  if growth < 1 {
    return nil, errors.New("cannot specify a growth factor of less than 1")
  }

  return &Retry {
    Initial: initial,
    Growth: growth,
    Next: initial,
  }, nil
}

func (r *Retry) Tick() time.Duration {
  wait := r.Next
  r.Next = time.Duration(float64(r.Next) * r.Growth)
  if r.Next > maxRetry {
    r.Next = maxRetry
  }
  return wait
}
