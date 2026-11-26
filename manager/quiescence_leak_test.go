// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package manager

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/config"
)

// The timer goroutine must be able to exit when nothing is receiving from the
// quiescence channel any more, as happens after the runner stops.
func TestRunner_quiescenceTimerExitsWithoutReceiver(t *testing.T) {
	c := config.DefaultConfig().Merge(&config.Config{
		Templates: &config.TemplateConfigs{
			&config.TemplateConfig{Contents: config.String("hello")},
		},
	})
	c.Finalize()

	r, err := NewRunner(c, true)
	if err != nil {
		t.Fatal(err)
	}

	q := newQuiescence(r.quiescenceCh, time.Millisecond, 5*time.Millisecond, r.templates[0])
	q.tick()
	r.Stop()

	const timerGoroutine = "manager.(*quiescence).tick."
	deadline := time.Now().Add(2 * time.Second)
	for {
		buf := make([]byte, 1<<20)
		stacks := string(buf[:runtime.Stack(buf, true)])
		if !strings.Contains(stacks, timerGoroutine) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("quiescence timer goroutine is still blocked after the runner stopped")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
