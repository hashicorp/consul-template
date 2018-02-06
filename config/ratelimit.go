package config

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	// DefaultRandomBackoff is the default max delay added between successful calls.
	DefaultRandomBackoff = 33 * time.Millisecond
	// DefaultMinDelayBetweenUpdates is the default delay between successful.
	DefaultMinDelayBetweenUpdates = 100 * time.Millisecond
)

// RateLimitFunc is the signature of a function to sleep between calls
type RateLimitFunc func(time.Duration) (bool, time.Duration)

// RateLimitConfig is a shared configuration
type RateLimitConfig struct {

	// Minimum Delay between 2 consecutive HTTP calls
	RandomBackoff *time.Duration `mapstructure:"random_backoff"`

	// Minimum Delay of 2 calls (includes download)
	MinDelayBetweenUpdates *time.Duration `mapstructure:"min_delay_between_updates"`

	// Enabled signals if this retry is enabled.
	Enabled *bool
}

// DefaultRateLimitConfig returns a configuration that is populated with the
// default values.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{}
}

// Copy returns a deep copy of this configuration.
func (c *RateLimitConfig) Copy() *RateLimitConfig {
	if c == nil {
		return nil
	}

	var o RateLimitConfig

	o.RandomBackoff = c.RandomBackoff

	o.MinDelayBetweenUpdates = c.MinDelayBetweenUpdates

	o.Enabled = c.Enabled

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *RateLimitConfig) Merge(o *RateLimitConfig) *RateLimitConfig {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.RandomBackoff != nil {
		r.RandomBackoff = o.RandomBackoff
	}
	if o.MinDelayBetweenUpdates != nil {
		r.MinDelayBetweenUpdates = o.MinDelayBetweenUpdates
	}

	if o.Enabled != nil {
		r.Enabled = o.Enabled
	}

	return r
}

// RateLimitFunc returns the RateLimit function associated with this configuration.
func (c *RateLimitConfig) RateLimitFunc() RateLimitFunc {
	return func(lastCallDuration time.Duration) (bool, time.Duration) {
		if !BoolVal(c.Enabled) {
			return false, 0
		}
		remaining := *c.MinDelayBetweenUpdates - lastCallDuration
		if remaining < 0 {
			remaining = 0
		}
		if c.RandomBackoff.Nanoseconds() < 1 {
			return remaining > 0, remaining
		}
		random := time.Duration(rand.Int63n(int64(c.RandomBackoff.Nanoseconds())))
		return true, (remaining + random)
	}
}

// Finalize ensures there no nil pointers.
func (c *RateLimitConfig) Finalize() {

	if c.RandomBackoff == nil {
		c.RandomBackoff = TimeDuration(DefaultRandomBackoff)
	}

	if c.MinDelayBetweenUpdates == nil {
		c.MinDelayBetweenUpdates = TimeDuration(DefaultMinDelayBetweenUpdates)
	}

	if c.Enabled == nil {
		c.Enabled = Bool(true)
	}
}

// GoString defines the printable version of this struct.
func (c *RateLimitConfig) GoString() string {
	if c == nil {
		return "(*RateLimitConfig)(nil)"
	}

	return fmt.Sprintf("&RateLimitConfig{"+
		"RandomBackoff:%s, "+
		"MinDelayBetweenUpdates:%s, "+
		"Enabled:%s"+
		"}",
		TimeDurationGoString(c.RandomBackoff),
		TimeDurationGoString(c.MinDelayBetweenUpdates),
		BoolGoString(c.Enabled),
	)
}
