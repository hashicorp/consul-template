// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"errors"
	"runtime"
	"testing"
	"time"
)

func TestVaultPKIQuery_FetchStopsWhileSleeping(t *testing.T) {
	d, err := NewVaultPKIQuery("pki/issue/example-dot-com", "/dev/null", nil)
	if err != nil {
		t.Fatal(err)
	}

	d.sleepCh <- time.Hour

	errCh := make(chan error, 1)
	go func() {
		_, _, err := d.Fetch(nil, nil)
		errCh <- err
	}()

	// Wait until Fetch has consumed the duration and entered its sleep. This
	// makes the test exercise cancellation of an in-progress sleep rather than
	// the early stop check at the beginning of Fetch.
	deadline := time.Now().Add(time.Second)
	for len(d.sleepCh) != 0 && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if len(d.sleepCh) != 0 {
		t.Fatal("Fetch did not start sleeping")
	}

	d.Stop()

	select {
	case err := <-errCh:
		if !errors.Is(err, ErrStopped) {
			t.Fatalf("Fetch returned %v after Stop, want ErrStopped", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Fetch did not return after Stop (goroutine leak)")
	}
}
