package main

import (
	"errors"
	"strings"
	"time"
)

// Wait is the Min/Max duration used by the Watcher
type Wait struct {
	// Min is the minimum time to wait before rendering a new template to disk
	Min time.Duration

	// Max is the maximum time to wait before rendering a new template to disk
	Max time.Duration
}

// ParseWait parses a string of the format `minimum(:maximum)` into a Wait
// struct.
func ParseWait(s string) (*Wait, error) {
	if len(strings.TrimSpace(s)) < 1 {
		return nil, errors.New("cannot specify empty wait interval")
	}

	parts := strings.Split(s, ":")

	var min, max time.Duration
	var err error

	switch len(parts) {
	case 1:
		min, err = time.ParseDuration(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}

		max = 4 * min
	case 2:
		min, err = time.ParseDuration(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}

		max, err = time.ParseDuration(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid wait interval format")
	}

	if min < 0 || max < 0 {
		return nil, errors.New("cannot specify a negative wait interval")
	}

	if max < min {
		return nil, errors.New("wait interval max must be larger than min")
	}

	return &Wait{min, max}, nil
}
