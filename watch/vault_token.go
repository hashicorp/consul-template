package watch

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/hashicorp/consul-template/config"
	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/vault/api"
)

// VaultTokenWatcher monitors the vault token for updates
func VaultTokenWatcher(clients *dep.ClientSet, c *config.VaultConfig) chan error {
	// c.Vault.Token is populated by the config code from all places
	// vault tokens are supported. So if there is no token set here,
	// tokens are not being used.
	raw_token := strings.TrimSpace(config.StringVal(c.Token))
	if raw_token == "" {
		return nil
	}

	unwrap := config.BoolVal(c.UnwrapToken)
	vault := clients.Vault()
	// buffer 1 error to allow for sequential errors to send and return
	errChan := make(chan error, 1)

	// get/set token once when kicked off, async after that..
	token, err := getToken(vault, raw_token, unwrap)
	if err != nil {
		errChan <- err
		return errChan
	}
	vault.SetToken(token)

	var once sync.Once
	var watcher *Watcher
	getWatcher := func() *Watcher {
		once.Do(func() {
			watcher = NewWatcher(&NewWatcherInput{
				Clients:        clients,
				RetryFuncVault: RetryFunc(c.Retry.RetryFunc()),
			})
		})
		return watcher
	}

	// Vault Agent Token File process //
	tokenFile := strings.TrimSpace(config.StringVal(c.VaultAgentTokenFile))
	if tokenFile != "" {
		atf, err := dep.NewVaultAgentTokenQuery(tokenFile)
		if err != nil {
			errChan <- fmt.Errorf("vaultwatcher: %w", err)
			return errChan
		}
		w := getWatcher()
		if _, err := w.Add(atf); err != nil {
			errChan <- fmt.Errorf("vaultwatcher: %w", err)
			return errChan
		}
		go func() {
			for {
				raw_token, err = waitforToken(w, raw_token, unwrap)
				if err != nil {
					errChan <- err
					return
				}
			}
		}()
	}

	// Vault Token Renewal process //
	renewVault := vault.Token() != "" && config.BoolVal(c.RenewToken)
	if renewVault {
		go func() {
			vt, err := dep.NewVaultTokenQuery(token)
			if err != nil {
				errChan <- fmt.Errorf("vaultwatcher: %w", err)
			}
			w := getWatcher()
			if _, err := w.Add(vt); err != nil {
				errChan <- fmt.Errorf("vaultwatcher: %w", err)
			}

			// VaultTokenQuery loops internally and never returns data,
			// so we only care about if it errors out.
			errChan <- <-w.ErrCh()
		}()
	}

	return errChan
}

// waitforToken blocks until the tokenfile is updated, and it given the new
// data on the watcher's DataCh(annel)
// (as a variable to swap out in tests)
var waitforToken = func(w *Watcher, old_raw_token string, unwrap bool) (string, error) {
	vault := w.clients.Vault()
	var new_raw_token string
	select {
	case v := <-w.DataCh():
		new_raw_token = strings.TrimSpace(v.Data().(string))
		if new_raw_token == old_raw_token {
			break
		}
		switch token, err := getToken(vault, new_raw_token, unwrap); err {
		case nil:
			vault.SetToken(token)
		default:
			log.Printf("[INFO] %s", err)
		}
	case err := <-w.ErrCh():
		return "", err
	}
	return new_raw_token, nil
}

type vaultClient interface {
	SetToken(string)
	Logical() *api.Logical
}

// getToken grabs the real token from raw_token (unwrap, etc)
func getToken(client vaultClient, token string, unwrap bool) (string, error) {
	// If vault agent specifies wrap_ttl for the token it is returned as
	// a SecretWrapInfo struct marshalled into JSON instead of the normal raw
	// token. This checks for that and pulls out the token if it is the case.
	var wrapinfo api.SecretWrapInfo
	if err := json.Unmarshal([]byte(token), &wrapinfo); err == nil {
		token = wrapinfo.Token
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	if unwrap {
		client.SetToken(token) // needs to be set to unwrap
		secret, err := client.Logical().Unwrap(token)
		switch {
		case err != nil:
			return token, fmt.Errorf("vault unwrap: %s", err)
		case secret == nil:
			return token, fmt.Errorf("vault unwrap: no secret")
		case secret.Auth == nil:
			return token, fmt.Errorf("vault unwrap: no secret auth")
		case secret.Auth.ClientToken == "":
			return token, fmt.Errorf("vault unwrap: no token returned")
		default:
			token = secret.Auth.ClientToken
		}
	}
	return token, nil
}
