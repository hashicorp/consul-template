package dependency

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

var (
	// Ensure implements
	_ Dependency = (*VaultReadQuery)(nil)
)

// VaultReadQuery is the dependency to Vault for a secret
type VaultReadQuery struct {
	stopCh chan struct{}

	path   string
	secret *Secret

	// vaultSecret is the actual Vault secret which we are renewing
	vaultSecret *api.Secret

	// isKV2 is possibly set non-false on the first read and indicates that a
	// transformation will be needed to the path and to reading the secrets.
	isKV2     bool
	mountPath string
}

// NewVaultReadQuery creates a new datacenter dependency.
func NewVaultReadQuery(s string) (*VaultReadQuery, error) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	if s == "" {
		return nil, fmt.Errorf("vault.read: invalid format: %q", s)
	}

	return &VaultReadQuery{
		stopCh: make(chan struct{}, 1),
		path:   s,
	}, nil
}

// Fetch queries the Vault API
func (d *VaultReadQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	opts = opts.Merge(&QueryOptions{})

	if d.secret != nil {
		if vaultSecretRenewable(d.secret) {
			log.Printf("[TRACE] %s: starting renewer", d)

			renewer, err := clients.Vault().NewRenewer(&api.RenewerInput{
				Grace:  opts.VaultGrace,
				Secret: d.vaultSecret,
			})
			if err != nil {
				return nil, nil, errors.Wrap(err, d.String())
			}
			go renewer.Renew()
			defer renewer.Stop()

		RENEW:
			for {
				select {
				case err := <-renewer.DoneCh():
					if err != nil {
						log.Printf("[WARN] %s: failed to renew: %s", d, err)
					}
					log.Printf("[WARN] %s: renewer returned (maybe the lease expired)", d)
					break RENEW
				case renewal := <-renewer.RenewCh():
					log.Printf("[TRACE] %s: successfully renewed", d)
					printVaultWarnings(d, renewal.Secret.Warnings)
					updateSecret(d.secret, renewal.Secret, d.isKV2)
				case <-d.stopCh:
					return nil, nil, ErrStopped
				}
			}
		} else {
			// The secret isn't renewable, probably the generic secret backend.
			dur := vaultRenewDuration(d.secret)
			log.Printf("[TRACE] %s: secret is not renewable, sleeping for %s", d, dur)
			select {
			case <-time.After(dur):
				// The lease is almost expired, it's time to request a new one.
			case <-d.stopCh:
				return nil, nil, ErrStopped
			}
		}
	}

	// We don't have a secret, or the prior renewal failed

	if err := d.checkForKV2(clients, opts); err != nil {
		return nil, nil, errors.Wrap(err, "checking for K/V version 2 for "+d.String())
	}

	vaultSecret, err := d.readSecret(clients, opts)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	// Print any warnings
	printVaultWarnings(d, vaultSecret.Warnings)

	// Create the cloned secret which will be exposed to the template.
	d.vaultSecret = vaultSecret
	d.secret = transformSecret(vaultSecret, d.isKV2)

	return respWithMetadata(d.secret)
}

// CanShare returns if this dependency is shareable.
func (d *VaultReadQuery) CanShare() bool {
	return false
}

// Stop halts the given dependency's fetch.
func (d *VaultReadQuery) Stop() {
	close(d.stopCh)
}

// String returns the human-friendly version of this dependency.
func (d *VaultReadQuery) String() string {
	return fmt.Sprintf("vault.read(%s)", d.path)
}

// Type returns the type of this dependency.
func (d *VaultReadQuery) Type() Type {
	return TypeVault
}

// checkForKV2 checks to see if the secret is on a K/V version 2 (or
// equivalent) mount.  Mutates d to set isKV2 and mountPath.
//
// Vault K/V version 2 requires path mutations and adjusts how the secrets are
// returned.
//
// This should be used to augment d on first call, to avoid pre-flights on
// every update.  Downside to caching is that if a mount is upgraded then we
// won't catch it and we'll have to be restarted.  Do we _need_ to handle an
// upgrade?  Perhaps on a signal?
//
// Problem: there's no way to check for K/V 2 within the mount without checking
// a path which might be a data item and the API method to check is  explicitly
// marked "Due to the nature of its intended usage, there is no guarantee on
// backwards compatibility for this endpoint."
// <https://www.vaultproject.io/api/system/internal-ui-mounts.html>
//
// We have very little choice here.  To support K/V version 2 without mandating
// it for everything, we need a probe, we'll have to use the internal API and
// keep it isolated to this function, so that future breaking updates can be
// handled here.
//
// The check includes the actual mount-path; rather than assuming that it's one
// component, we'll do as the vault CLI tool does and use the mount information
// returned here.
func (d *VaultReadQuery) checkForKV2(clients *ClientSet, opts *QueryOptions) error {
	// Have checked that this API handles arbitrary strings and just inspects
	// the first component.
	preflightPath := "sys/internal/ui/mounts/" + d.path
	contextLabel := "preflight(" + d.String() + ")"
	log.Printf("[TRACE] %s: GET %s", contextLabel, "/v1/"+preflightPath)
	mountInfo, err := clients.Vault().Logical().Read(preflightPath)
	if err != nil {
		return errors.Wrap(err, contextLabel)
	}
	if mountInfo == nil {
		return fmt.Errorf("no mount information available at %q", preflightPath)
	}
	mountOptionsRaw, ok := mountInfo.Data["options"]
	if !ok {
		log.Printf("[DEBUG] %s: missing 'options' key", contextLabel)
		return nil
	}
	mountOptions, ok := mountOptionsRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("couldn't parse options in preflight response, bad map type %T", mountOptionsRaw)
	}
	versionRaw, ok := mountOptions["version"]
	if !ok {
		// This is the _normal_ case for a v1 mount
		log.Printf("[DEBUG] %s: options missing 'version' key, assuming v1 mount", contextLabel)
		return nil
	}

	// We explicitly check for 2, not 3 or any other value.  The changes for v2
	// were significant enough that we can't make any assumptions.
	versionString, ok := versionRaw.(string)
	if !ok {
		log.Printf("[DEBUG] %q: options.version is not a string: %#+v", contextLabel, versionRaw)
		return nil
	}
	if versionString != "2" {
		log.Printf("[DEBUG] %s: options.version exists but is not 2: %q", contextLabel, versionString)
		return nil
	}

	d.isKV2 = true

	if mountPathRaw, ok := mountInfo.Data["path"]; ok {
		d.mountPath, ok = mountPathRaw.(string)
		if !ok {
			log.Printf("[DEBUG] %s: mountpath in .path is not a string: %#+v", contextLabel, mountPathRaw)
		}
	}

	log.Printf("[DEBUG] %s: version %q mount at %q", contextLabel, versionString, d.mountPath)
	return nil
}

// readSecret handles changing the path as needed for special mount-points (eg,
// K/V v2) but still returns the top-level Vault *api.Secret always, because
// there's metadata which callers might need.  It's up to callers to probe down
// into the 'data' member for K/V v2.
func (d *VaultReadQuery) readSecret(clients *ClientSet, opts *QueryOptions) (*api.Secret, error) {
	var queryPath string
	if d.isKV2 {
		// d.mountPath ends with a /.
		// If someone asks us for reading the mountpoint, we'll have a shorter path
		// than a mountPath.
		if len(d.path) < len(d.mountPath) {
			return nil, fmt.Errorf("won't probe a mount-point for secrets on itself: %q", d.path)
		}
		queryPath = d.mountPath + "data/" + strings.TrimPrefix(d.path, d.mountPath)
	} else {
		queryPath = d.path
	}

	log.Printf("[TRACE] %s: GET %s", d, "/v1/"+queryPath)
	vaultSecret, err := clients.Vault().Logical().Read(queryPath)
	if err != nil {
		return nil, errors.Wrap(err, d.String())
	}
	if vaultSecret == nil {
		return nil, fmt.Errorf("no secret read from %q", queryPath)
	}

	return vaultSecret, nil
}
