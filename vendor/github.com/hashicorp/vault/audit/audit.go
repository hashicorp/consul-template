package audit

import (
	"github.com/hashicorp/vault/helper/salt"
	"github.com/hashicorp/vault/logical"
)

// Backend interface must be implemented for an audit
// mechanism to be made available. Audit backends can be enabled to
// sink information to different backends such as logs, file, databases,
// or other external services.
type Backend interface {
	// LogRequest is used to syncronously log a request. This is done after the
	// request is authorized but before the request is executed. The arguments
	// MUST not be modified in anyway. They should be deep copied if this is
	// a possibility.
	LogRequest(*logical.Auth, *logical.Request, error) error

	// LogResponse is used to syncronously log a response. This is done after
	// the request is processed but before the response is sent. The arguments
	// MUST not be modified in anyway. They should be deep copied if this is
	// a possibility.
	LogResponse(*logical.Auth, *logical.Request, *logical.Response, error) error

	// GetHash is used to return the given data with the backend's hash,
	// so that a caller can determine if a value in the audit log matches
	// an expected plaintext value
	GetHash(string) string
}

type BackendConfig struct {
	// The salt that should be used for any secret obfuscation
	Salt *salt.Salt

	// Config is the opaque user configuration provided when mounting
	Config map[string]string
}

// Factory is the factory function to create an audit backend.
type Factory func(*BackendConfig) (Backend, error)
