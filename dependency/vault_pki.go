package dependency

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Ensure implements
var _ Dependency = (*VaultPKIQuery)(nil)

// VaultPKIQuery is the dependency to Vault for a secret
type VaultPKIQuery struct {
	stopCh  chan struct{}
	sleepCh chan time.Duration

	pkiPath  string
	data     map[string]interface{}
	filePath string
}

// NewVaultReadQuery creates a new datacenter dependency.
func NewVaultPKIQuery(urlpath, filepath string, data map[string]interface{}) (*VaultPKIQuery, error) {
	urlpath = strings.TrimSpace(urlpath)
	urlpath = strings.Trim(urlpath, "/")
	if urlpath == "" {
		return nil, fmt.Errorf("vault.read: invalid format: %q", urlpath)
	}

	secretURL, err := url.Parse(urlpath)
	if err != nil {
		return nil, err
	}

	return &VaultPKIQuery{
		stopCh:   make(chan struct{}, 1),
		sleepCh:  make(chan time.Duration, 1),
		pkiPath:  secretURL.Path,
		data:     data,
		filePath: filepath,
	}, nil
}

// Fetch queries the Vault API
func (d *VaultPKIQuery) Fetch(clients *ClientSet, opts *QueryOptions,
) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}
	select {
	case dur := <-d.sleepCh:
		time.Sleep(dur)
	default:
	}

	needsRenewal := fmt.Errorf("needs renewal")
	getPEM := func(renew bool) ([]byte, error) {
		var encPEM []byte
		var err error
		encPEM, err = os.ReadFile(d.filePath)
		if renew || err != nil || len(encPEM) == 0 {
			encPEM, err = d.fetchPEM(clients)
			// no need to write cert to file as it is the template dest
		}
		if err != nil {
			return nil, err
		}

		block, cert, err := getCert(encPEM)
		if err != nil {
			return nil, err
		}

		if sleepFor, ok := goodFor(cert); ok {
			d.sleepCh <- sleepFor
			return pem.EncodeToMemory(block), nil
		}
		return []byte{}, needsRenewal
	}

	pemBytes, err := getPEM(false)
	switch err {
	case nil:
	case needsRenewal:
		pemBytes, err = getPEM(true)
		if err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, err
	}
	return respWithMetadata(string(pemBytes))
}

// returns time left in ~90% of the original lease and a boolean
// that returns false if cert needs renewing, true otherwise
func goodFor(cert *x509.Certificate) (time.Duration, bool) {
	// These are all int64's with Seconds since the Epoch
	// handy for the math
	start, end := cert.NotBefore.Unix(), cert.NotAfter.Unix()
	now := time.Now().UTC().Unix()
	if end <= now { // already expired
		return 0, false
	}
	lifespan := end - start        // full ttl of cert
	duration := end - now          // duration remaining
	gooddur := (duration * 9) / 10 // 90% of duration
	mindur := (lifespan / 10)      // 10% of lifespan
	if gooddur <= mindur {
		return 0, false // almost expired, get a new one
	}
	if gooddur > 100 { // 100 seconds
		// add jitter if big enough for it to matter
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		// between 87% and 93%
		gooddur = gooddur + ((gooddur / 100) * int64(r.Intn(6)-3))
	}
	sleepFor := time.Duration(gooddur * 1e9)
	return sleepFor, true
}

// loops through all pem encoded blocks in the byte stream
// returning the first certificate found (ignoring CAs and chain certs)
func getCert(encoded []byte) (*pem.Block, *x509.Certificate, error) {
	var block *pem.Block
	var firstErr error
	for {
		block, encoded = pem.Decode(encoded)
		if block != nil {
			cert, err := x509.ParseCertificate(block.Bytes)
			switch {
			case err == nil && !cert.IsCA:
				return block, cert, nil
			case firstErr == nil:
				firstErr = err
			}
			continue
		}
		break
	}
	return nil, nil, errors.Wrap(firstErr, "failed to parse cert")
}

//
func (d *VaultPKIQuery) fetchPEM(clients *ClientSet) ([]byte, error) {
	vaultSecret, err := clients.Vault().Logical().Write(d.pkiPath, d.data)
	switch {
	case err != nil:
		return nil, errors.Wrap(err, d.String())
	case vaultSecret == nil:
		return nil, fmt.Errorf("no secret exists at %s", d.pkiPath)
	}
	printVaultWarnings(d, vaultSecret.Warnings)
	encPEM, ok := vaultSecret.Data["certificate"].(string)
	if !ok {
		return nil, fmt.Errorf("secret didn't include cert")
	}
	return []byte(encPEM), nil
}

func (d *VaultPKIQuery) stopChan() chan struct{} {
	return d.stopCh
}

// CanShare returns if this dependency is shareable.
func (d *VaultPKIQuery) CanShare() bool {
	return false
}

// Stop halts the given dependency's fetch.
func (d *VaultPKIQuery) Stop() {
	close(d.stopCh)
}

// String returns the human-friendly version of this dependency.
func (d *VaultPKIQuery) String() string {
	return fmt.Sprintf("vault.pki(%s)", d.pkiPath)
}

// Type returns the type of this dependency.
func (d *VaultPKIQuery) Type() Type {
	return TypeVault
}
