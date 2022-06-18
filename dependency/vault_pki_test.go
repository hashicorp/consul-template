// go:build ignore

package dependency

import (
	"bytes"
	"encoding/pem"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/consul-template/renderer"
	"github.com/hashicorp/vault/api"
)

func Test_VaultPKI_notGoodFor(t *testing.T) {
	// only test the negation, postive is tested below with certificates
	// fetched in Vault integration tests (creating certs is non-trivial)
	_, cert, err := getCert([]byte(validCert))
	if err != nil {
		t.Error(err)
	}
	dur, ok := goodFor(cert)
	if ok != false {
		t.Error("should be false", dur)
	}
	// duration should be negative as cert has already expired
	// but still tests cert time parsing (it'd be 0 if there was an issue)
	if dur > 0 {
		t.Error("cert shouldn't positive (old cert)")
	}
}

func Test_VaultPKI_getCert(t *testing.T) {
	// tests w/ valid cert
	for _, testcert := range []string{validCert, validBad, badGood, validCertAfterCA, validCertGarbo} {
		pemBlk, cert, err := getCert([]byte(testcert))
		if err != nil {
			t.Fatal(err) // Fatal to avoid panic's below
		}
		got := strings.TrimRight(string(pem.EncodeToMemory(pemBlk)), "\n")
		want := strings.TrimRight(strings.TrimSpace(validCert), "\n")
		if got != want {
			t.Errorf("certs didn't match:\ngot: %v\nwant: %v", got, want)
		}
		if err := cert.VerifyHostname("foo.example.com"); err != nil {
			t.Error(err)
		}
	}
	// tests w/o valid cert (test error)
	expectedErr := "x509: malformed certificate"
	for _, badcert := range []string{badCert, badPlusCA, badPlusGarbo} {
		_, _, err := getCert([]byte(badcert))
		switch {
		case err == nil:
			t.Errorf("error should not be nil")
		case !strings.Contains(err.Error(), expectedErr):
			t.Errorf("wrong error; wanted: '%s', got: '%s'", expectedErr, err)
		}
	}
}

func Test_VaultPKI_fetchPEM(t *testing.T) {
	clients := testClients

	data := map[string]interface{}{
		"common_name": "foo.example.com",
		"ttl":         "2h",
		"ip_sans":     "127.0.0.1,192.168.2.2",
	}
	d, err := NewVaultPKIQuery("pki/issue/example-dot-com", "/dev/null", data)
	if err != nil {
		t.Error(err)
	}
	encPEM, err := d.fetchPEM(clients)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Contains(encPEM, []byte("CERTIFICATE")) {
		t.Errorf("certificate not fetched, got: %s", string(encPEM))
	}
	// test path error
	d, err = NewVaultPKIQuery("pki/issue/does-not-exist", "/dev/null", data)
	if err != nil {
		t.Error(err)
	}
	_, err = d.fetchPEM(clients)
	var respErr *api.ResponseError
	if !errors.As(err, &respErr) {
		t.Error(err)
	}
}

func Test_VaultPKI_refetch(t *testing.T) {
	t.Parallel() // have time waits, so parallel might help

	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(f.Name())
	defer os.Remove(f.Name())

	clients := testClients
	/// above is prep work
	data := map[string]interface{}{
		"common_name": "foo.example.com",
		"ttl":         "3s",
		"ip_sans":     "127.0.0.1,192.168.2.2",
	}
	d, err := NewVaultPKIQuery("pki/issue/example-dot-com", f.Name(), data)
	if err != nil {
		t.Fatal(err)
	}
	act1, rm, err := d.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rm == nil {
		t.Error("Fetch returned nil for response metadata.")
	}

	cert1, ok := act1.(string)
	if !ok || !strings.Contains(cert1, "BEGIN") {
		t.Fatalf("expected a cert but found: %s", cert1)
	}

	// Fake template rendering file to disk
	_, err = renderer.Render(&renderer.RenderInput{
		Contents: []byte(cert1),
		Path:     f.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// re-fetch, should be the same cert pulled from the file
	// if re-fetched from Vault it will be different
	<-d.sleepCh // drain sleepCh so we don't wait and reuse the cached copy
	act2, rm, err := d.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rm == nil {
		t.Error("Fetch returned nil for response metadata.")
	}

	cert2, ok := act2.(string)
	if !ok || !strings.Contains(cert2, "BEGIN") {
		t.Fatalf("expected a cert but found: %s", cert2)
	}

	if cert1 != cert2 {
		t.Errorf("certs don't match and should.")
	}

	// Don't pre-drain here as we want it to get a new cert
	act3, rm, err := d.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rm == nil {
		t.Error("Fetch returned nil for response metadata.")
	}

	cert3, ok := act3.(string)
	if !ok || !strings.Contains(cert2, "BEGIN") {
		t.Fatalf("expected a cert but found: %s", cert2)
	}

	if cert2 == cert3 {
		t.Errorf("certs match and shouldn't.")
	}
}

const validCert = `
-----BEGIN CERTIFICATE-----
MIIDWTCCAkGgAwIBAgIUUARA+vQExU8zjdsX/YXMMu1K5FkwDQYJKoZIhvcNAQEL
BQAwFjEUMBIGA1UEAxMLZXhhbXBsZS5jb20wHhcNMjIwMzAxMjIzMzAzWhcNMjIw
MzA0MjIzMzMzWjAaMRgwFgYDVQQDEw9mb28uZXhhbXBsZS5jb20wggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDD3sktiNGo/CSvtL84+GIcsuDzp1VFjG++
8P682ZPiqPGjrgwe3P8ypyhQv6I8ZGOyu7helMqBN/S1mrhmHWUONy/4o95QWDsJ
CGw4H44dRil5hKC6K8BUrf79XGAGIQJr3T6I5CCwxukfYhU/+xNE3dq5AgLrIIB2
BtzZA6m1T5CmgAzSzI1byTjaRpxOJjucI37iKzkx7AkYS5hGfVsFmJgGi/UXhvzK
uwnHHIq9rLItx7p261dJV8mxRDFaf4x+4bZh2kYkEaG8REOfyHSCJ78RniWbF/DN
Jtgh8bT2/938/ecBtWcTN+psICD62DJii6988FD2qS+Yd8Eu8M5rAgMBAAGjgZow
gZcwDgYDVR0PAQH/BAQDAgOoMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcD
AjAdBgNVHQ4EFgQUfmm32UJb3xJNxfA7ZB0Q5RXsQIkwHwYDVR0jBBgwFoAUDoYJ
CtobWJrR1xmTsYJd9buj2jwwJgYDVR0RBB8wHYIPZm9vLmV4YW1wbGUuY29thwR/
AAABhwTAqAEpMA0GCSqGSIb3DQEBCwUAA4IBAQBzB+RM2PSZPmDG3xJssS1litV8
TOlGtBAOUi827W68kx1lprp35c9Jyy7l4AAu3Q1+az3iDQBfYBazq89GOZeXRvml
x9PVCjnXP2E7mH9owA6cE+Z1cLN/5h914xUZCb4t9Ahu04vpB3/bnoucXdM5GJsZ
EJylY99VsC/bZKPCheZQnC/LtFBC31WEGYb8rnB7gQxmH99H91+JxnJzYhT1a6lw
arHERAKScrZMTrYPLt2YqYoeyO//aCuT9YW6YdIa9jPQhzjeMKXywXLetE+Ip18G
eB01bl42Y5WwHl0IrjfbEevzoW0+uhlUlZ6keZHr7bLn/xuRCUkVfj3PRlMl
-----END CERTIFICATE-----
`

const validCA = `
-----BEGIN CERTIFICATE-----
MIIDNTCCAh2gAwIBAgIUbJ/1ELw6X6OUg6YeVtsTqg20G2swDQYJKoZIhvcNAQEL
BQAwFjEUMBIGA1UEAxMLZXhhbXBsZS5jb20wHhcNMjIwMzAxMjIzMjQ3WhcNMjMw
MzAxMjIzMzE2WjAWMRQwEgYDVQQDEwtleGFtcGxlLmNvbTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAPAiGf1Shr7e/AA7VGsaQiMKz9+HoTgrt4mJIAPu
aelNavl7umr1AckEc7g+zGYauJmhfsPalS5BlvZ1hjBhgi4g1MsKf/64BLfxkxJH
HLX2kAEUBzFWBeX0EE/rl0pb81afZv+6Kyi2X6cN3kFC0gEtF1BScAoWEKWYx9xz
oPzB7Qql4BKaZ8KXgeryDIQ4Zbg2yKwSdS9TILGMylvqCdne5UkQP7bGW0i4C7r9
noDtJZIo83vzH6YlN99C66pLm8m3qnKgWk5clIizlh9lw0XEQKZQ69tRNuxRSF5r
6XVaDWoYEOm+gJ7DRoKEHqA4ov1BbfLEocvEzjfcrulb1bsCAwEAAaN7MHkwDgYD
VR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFA6GCQraG1ia
0dcZk7GCXfW7o9o8MB8GA1UdIwQYMBaAFA6GCQraG1ia0dcZk7GCXfW7o9o8MBYG
A1UdEQQPMA2CC2V4YW1wbGUuY29tMA0GCSqGSIb3DQEBCwUAA4IBAQC2yfszyLyX
7Yhuzvda3EYKTsfiXA6+Cqx7TZVyIHF0AgEkeIaDmmB5Gh6cizvtpHpwLwB94UWq
LNda6gzocChpC34A2xZ5QLCk+xeNrC1rHH+wk90K9ac+G5rtVQhzNKXBMup6GZFu
zOi+yS9f/oKtx0obrjG/NVtYcxAZ/2Zv1Mu4MLuw9EGrWztvEOImd8G22sgigsbJ
WR7VgKRCphRGmfp/SlI3c/zYScHHanZ3umQvilPmftPX+BxVFA5FhCUUimkZEJMq
+hmBcwd8e1LZLz7Mz5IsrN6+lRZ78T+zV8sxNdF1mIWs+ZsrmzQBZq4Q0uLjDiD/
v57ZY8gIDE5E
-----END CERTIFICATE-----
`

const (
	validCertAfterCA = validCA + validCert
	validCertGarbo   = `
aa983w4;/amndsfm908q26035vc;ng902338(%@%!@QY!&DVLMNSALX>PT(RQ!QO*%@
` + validCert + `
!Q)(*@^YUO!Q#MN%$#WP(G^&+_!%)!+^%$Y	:!#QLKENFVJ)	!#*&%YHTM
`
)

const badCert = `
-----BEGIN CERTIFICATE-----
MIIDWTCCAkGgAwIBAgIUUARA+vQExU8zjdsX/YXMMu1K5FkwDQYJKoZIhvcNAQEL
eB01bl42Y5WwHl0IrjfbEevzoW0+uhlUlZ6keZHr7bLn/xuRCUkVfj3PRlMl
-----END CERTIFICATE-----
`

const (
	badPlusCA    = badCert + validCA
	badPlusGarbo = `
aa983w4;/amndsfm908q26035vc;ng902338(%@%!@QY!&DVLMNSALX>PT(RQ!QO*%@
` + badCert + `
!Q)(*@^YUO!Q#MN%$#WP(G^&+_!%)!+^%$Y	:!#QLKENFVJ)	!#*&%YHTM
`
)

const (
	badGood  = badCert + validCert
	validBad = validCert + badCert
)
