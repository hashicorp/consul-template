package dependency

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewConnectLeafQuery(t *testing.T) {
	t.Parallel()

	act := NewConnectLeafQuery("foo")
	act.stopCh = nil
	exp := &ConnectLeafQuery{service: "foo"}
	assert.Equal(t, exp, act)
}

func TestConnectLeafQuery_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("empty-service", func(t *testing.T) {
		d := NewConnectLeafQuery("")

		_, _, err := d.Fetch(testClients, nil)
		exp := "Unexpected response code: 500 (" +
			"URI must be either service or agent)"
		if errors.Cause(err).Error() != exp {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
	t.Run("with-service", func(t *testing.T) {
		d := NewConnectLeafQuery("foo")
		raw, _, err := d.Fetch(testClients, nil)
		if err != nil {
			t.Fatal(err)
		}
		cert := raw.(*api.LeafCert)
		if cert.Service != "foo" {
			t.Fatalf("Unexpected service: %v", cert.Service)
		}
		if cert.CertPEM == "" {
			t.Fatal("Empty cert PEM")
		}
		if cert.ValidAfter.After(time.Now()) {
			t.Fatalf("Bad cert: (bad ValidAfter: %v)", cert.ValidAfter)
		}
		if cert.ValidBefore.Before(time.Now()) {
			t.Fatalf("Bad cert: (bad ValidBefore: %v)", cert.ValidBefore)
		}
	})
	t.Run("double-check", func(t *testing.T) {
		d1 := NewConnectLeafQuery("foo")
		raw1, _, err := d1.Fetch(testClients, nil)
		if err != nil {
			t.Fatal(err)
		}
		cert1 := raw1.(*api.LeafCert)
		d2 := NewConnectLeafQuery("foo")
		raw2, _, err := d2.Fetch(testClients, nil)
		if err != nil {
			t.Fatal(err)
		}
		cert2 := raw2.(*api.LeafCert)
		if cert1.CertPEM != cert2.CertPEM {
			t.Fatalf("Certs should match:\n%v\n%v",
				cert1.CertPEM, cert2.CertPEM)
		}
	})
}

func TestConnectLeafQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		service string
		exp     string
	}{
		{
			"empty",
			"",
			"connect.caleaf",
		},
		{
			"service",
			"foo",
			"connect.caleaf(foo)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d := NewConnectLeafQuery(tc.service)
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
