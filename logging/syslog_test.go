package logging

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	gsyslog "github.com/hashicorp/go-syslog"
	"github.com/hashicorp/logutils"
)

func TestSyslogFilter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	// Travis does not support syslog for some reason
	for _, ci_env := range []string{"TRAVIS", "CIRCLECI"} {
		if ci := os.Getenv(ci_env); ci != "" {
			t.SkipNow()
		}
	}

	l, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, "LOCAL0", "consul-template")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	filt, err := newLogFilter(ioutil.Discard, logutils.LogLevel("INFO"))
	if err != nil {
		t.Fatal(err)
	}

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
