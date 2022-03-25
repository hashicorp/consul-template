package dependency

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	// Ensure implements
	_ Dependency = (*VaultPKIQuery)(nil)
)

// VaultPKIQuery is the dependency to Vault for a secret
type VaultPKIQuery struct {
	stopCh  chan struct{}
	sleepCh chan time.Duration

	pkiPath  string
	data     map[string]interface{}
	filePath string
	// populated from cert data
	pemBlock   *pem.Block
	expiration time.Time
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

	var encPEM []byte
	var err error
	if encPEM, err = os.ReadFile(d.filePath); err != nil || len(encPEM) == 0 {
		// no need to write to file as it is the template dest
		encPEM, err = d.fetchPEM(clients)
	}
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	block, cert, err := getCert(encPEM)
	if err != nil {
		return nil, nil, err
	}

	if sleepFor, ok := goodFor(cert); ok {
		d.sleepCh <- sleepFor
		return string(pem.EncodeToMemory(block)), nil, nil
	}
	d.pemBlock = nil
	return "", &ResponseMetadata{
		LastContact: 0,
		LastIndex:   0,
	}, nil
}

// returns time left in 90% of the original lease
// boolean returns false if time has already past
func goodFor(cert *x509.Certificate) (time.Duration, bool) {
	start, end := cert.NotBefore.Unix(), cert.NotAfter.Unix()
	// use 90% of the cert duration
	goodfor := ((end - start) * 9) / 10
	sleepFor := time.Until(time.Unix(start+goodfor, 0))
	return sleepFor, sleepFor > 0
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
