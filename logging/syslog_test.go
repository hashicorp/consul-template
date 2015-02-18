package logging

import (
	"os"
	"runtime"
	"testing"

	"github.com/hashicorp/go-syslog"
	"github.com/hashicorp/logutils"
)

func TestSyslogFilter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	// Travis does not support syslog for some reason
	if travis := os.Getenv("TRAVIS"); travis != "" {
		t.SkipNow()
	}

	l, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, "LOCAL0", "consul-template")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	filt := NewLogFilter()
	filt.MinLevel = logutils.LogLevel("INFO")

	s := &SyslogWrapper{l, filt}
	n, err := s.Write([]byte("[INFO] test"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if n == 0 {
		t.Fatalf("should have logged")
	}

	n, err = s.Write([]byte("[DEBUG] test"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if n != 0 {
		t.Fatalf("should not have logged")
	}
}
