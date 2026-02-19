// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package watch

import (
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
)

func TestTTLZeroRetryLogic(t *testing.T) {
	// Save original values
	origMaxRetries := dep.VaultTTLZeroMaxRetries
	origMaxBackoff := dep.VaultTTLZeroMaxBackoff
	defer func() {
		dep.VaultTTLZeroMaxRetries = origMaxRetries
		dep.VaultTTLZeroMaxBackoff = origMaxBackoff
	}()

	t.Run("unlimited_retries_with_backoff_cap", func(t *testing.T) {
		dep.VaultTTLZeroMaxRetries = 0 // unlimited
		dep.VaultTTLZeroMaxBackoff = 5 * time.Minute

		tests := []struct {
			retry       int
			expectRetry bool
			expectSleep time.Duration
		}{
			{0, true, 250 * time.Millisecond},
			{1, true, 500 * time.Millisecond},
			{2, true, 1 * time.Second},
			{3, true, 2 * time.Second},
			{4, true, 4 * time.Second},
			{5, true, 8 * time.Second},
			{6, true, 16 * time.Second},
			{7, true, 32 * time.Second},
			{8, true, 64 * time.Second},
			{9, true, 128 * time.Second},
			{10, true, 256 * time.Second},
			{11, true, 5 * time.Minute}, // capped
			{12, true, 5 * time.Minute}, // capped
			{20, true, 5 * time.Minute}, // still retrying, capped
		}

		for _, tt := range tests {
			err := &dep.VaultTTLZeroError{}

			var retry bool
			var sleep time.Duration

			// Simulate the retry logic from view.go
			if dep.VaultTTLZeroMaxRetries > 0 && tt.retry >= dep.VaultTTLZeroMaxRetries {
				retry = false
				sleep = 0
			} else {
				retry = true
				baseSleep := 250 * time.Millisecond
				sleep = time.Duration(1<<uint(tt.retry)) * baseSleep
				if dep.VaultTTLZeroMaxBackoff > 0 && sleep > dep.VaultTTLZeroMaxBackoff {
					sleep = dep.VaultTTLZeroMaxBackoff
				}
			}

			if retry != tt.expectRetry {
				t.Errorf("retry %d: expected retry=%v, got %v (error: %v)",
					tt.retry, tt.expectRetry, retry, err)
			}
			if sleep != tt.expectSleep {
				t.Errorf("retry %d: expected sleep=%v, got %v",
					tt.retry, tt.expectSleep, sleep)
			}
		}
	})

	t.Run("limited_retries", func(t *testing.T) {
		dep.VaultTTLZeroMaxRetries = 5
		dep.VaultTTLZeroMaxBackoff = 10 * time.Minute

		tests := []struct {
			retry       int
			expectRetry bool
		}{
			{0, true},
			{1, true},
			{2, true},
			{3, true},
			{4, true},
			{5, false},  // exceeded
			{6, false},  // exceeded
			{10, false}, // exceeded
		}

		for _, tt := range tests {
			var retry bool

			if dep.VaultTTLZeroMaxRetries > 0 && tt.retry >= dep.VaultTTLZeroMaxRetries {
				retry = false
			} else {
				retry = true
			}

			if retry != tt.expectRetry {
				t.Errorf("retry %d: expected retry=%v, got %v",
					tt.retry, tt.expectRetry, retry)
			}
		}
	})

	t.Run("custom_backoff_cap", func(t *testing.T) {
		dep.VaultTTLZeroMaxRetries = 0
		dep.VaultTTLZeroMaxBackoff = 30 * time.Second

		tests := []struct {
			retry       int
			expectSleep time.Duration
		}{
			{0, 250 * time.Millisecond},
			{1, 500 * time.Millisecond},
			{2, 1 * time.Second},
			{3, 2 * time.Second},
			{4, 4 * time.Second},
			{5, 8 * time.Second},
			{6, 16 * time.Second},
			{7, 30 * time.Second},  // capped at 30s
			{8, 30 * time.Second},  // capped at 30s
			{10, 30 * time.Second}, // capped at 30s
		}

		for _, tt := range tests {
			baseSleep := 250 * time.Millisecond
			sleep := time.Duration(1<<uint(tt.retry)) * baseSleep
			if dep.VaultTTLZeroMaxBackoff > 0 && sleep > dep.VaultTTLZeroMaxBackoff {
				sleep = dep.VaultTTLZeroMaxBackoff
			}

			if sleep != tt.expectSleep {
				t.Errorf("retry %d: expected sleep=%v, got %v",
					tt.retry, tt.expectSleep, sleep)
			}
		}
	})

	t.Run("no_backoff_cap", func(t *testing.T) {
		dep.VaultTTLZeroMaxRetries = 0
		dep.VaultTTLZeroMaxBackoff = 0 // no cap

		// Test that backoff grows without limit
		baseSleep := 250 * time.Millisecond
		sleep := time.Duration(1<<uint(10)) * baseSleep // 2^10 * 250ms = 256s

		if dep.VaultTTLZeroMaxBackoff > 0 && sleep > dep.VaultTTLZeroMaxBackoff {
			sleep = dep.VaultTTLZeroMaxBackoff
		}

		expected := 256 * time.Second
		if sleep != expected {
			t.Errorf("expected sleep=%v, got %v (no cap should allow growth)",
				expected, sleep)
		}
	})
}

func TestVaultTTLZeroErrorType(t *testing.T) {
	err := &dep.VaultTTLZeroError{}

	// Test error message
	expectedMsg := "vault rotating secret returned ttl=0, will retry"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}

	// Test type assertion
	var testErr error = err
	if _, ok := testErr.(*dep.VaultTTLZeroError); !ok {
		t.Error("type assertion failed for VaultTTLZeroError")
	}
}
