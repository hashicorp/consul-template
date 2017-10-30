package mssql

import "testing"

func TestBadOpen(t *testing.T) {
	drv := driverWithProcess(t)
	_, err := drv.open("port=bad")
	if err == nil {
		t.Fail()
	}
}

func TestIsProc(t *testing.T) {
	list := []struct {
		s  string
		is bool
	}{
		{"proc", true},
		{"select 1;", false},
		{"[proc 1]", true},
		{"[proc\n1]", false},
	}

	for _, item := range list {
		got := isProc(item.s)
		if got != item.is {
			t.Errorf("for %q, got %t want %t", item.s, got, item.is)
		}
	}
}
